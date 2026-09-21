package httpapi

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
)

type previewBody struct {
	Entries  struct{ Total, New, Existing, Invalid int }           `json:"entries"`
	People   struct{ Total, New, Existing, Invalid int }           `json:"people"`
	Links    struct{ Total, New, Existing, Orphaned int }          `json:"links"`
	Timeline struct{ Total, New, Existing, Orphaned int }          `json:"timeline"`
	Files    struct{ Total, New, Existing, NoBinary, Invalid int } `json:"files"`
	DryRun   bool                                                  `json:"dryRun"`
}

func decodePreview(t *testing.T, raw string) previewBody {
	t.Helper()
	var p previewBody
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("preview json: %v — %s", err, raw)
	}
	return p
}

const exportFixture = `{
  "entries": [{"id": "11111111-1111-1111-1111-111111111111", "title": "Pay rent", "type": "task", "date": "2026-03-01"}],
  "people": [{"id": "22222222-2222-2222-2222-222222222222", "name": "Eva"}],
  "personLinks": [{"personId": "22222222-2222-2222-2222-222222222222", "otherId": "99999999-9999-9999-9999-999999999999", "kind": "family"}],
  "personTimeline": [{"id": "33333333-3333-3333-3333-333333333333", "personId": "22222222-2222-2222-2222-222222222222", "title": "Met for coffee", "occurredOn": "2026-02-14"}],
  "files": [{"id": "44444444-4444-4444-4444-444444444444", "name": "abc123.png", "origName": "a.png", "size": 4, "mime": "image/png"}]
}`

func TestRestoreDryRunWritesNothing(t *testing.T) {
	srv, session := newTestServer(t)

	p := decodePreview(t, doJSON(t, srv, http.MethodPost, "/api/restore?dry=1", exportFixture, session))
	if !p.DryRun || p.Entries.New != 1 || p.People.New != 1 || p.Links.Orphaned != 1 || p.Files.NoBinary != 1 {
		t.Fatalf("unexpected preview: %+v", p)
	}
	// Dry-run must not persist — a real restore of the same payload still counts all new.
	res := doJSON(t, srv, http.MethodPost, "/api/restore", exportFixture, session)
	if !strings.Contains(res, `"restored":1`) {
		t.Fatalf("restore response: %s", res)
	}

	p2 := decodePreview(t, doJSON(t, srv, http.MethodPost, "/api/restore?dry=1", exportFixture, session))
	if p2.Entries.Existing != 1 || p2.Entries.New != 0 || p2.People.Existing != 1 {
		t.Fatalf("second preview should report existing rows: %+v", p2)
	}
}

func TestUploadDedupSameBytes(t *testing.T) {
	srv, session := newTestServer(t)

	upload := func() string {
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		part, _ := w.CreateFormFile("file", "note.png")
		part.Write([]byte("identical-payload"))
		w.Close()
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/files", &body)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+session)
		res, err := srv.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		buf := new(bytes.Buffer)
		buf.ReadFrom(res.Body)
		return buf.String()
	}

	first := upload()
	second := upload()
	if !strings.Contains(second, `"deduped":true`) {
		t.Fatalf("second upload should dedupe: %s", second)
	}
	var a, b struct {
		ID string `json:"id"`
	}
	json.Unmarshal([]byte(first), &a)
	json.Unmarshal([]byte(second), &b)
	if a.ID == "" || a.ID != b.ID {
		t.Fatalf("dedup should return the original row: %s vs %s", first, second)
	}
}

func TestRestoreZipUnpacksBinary(t *testing.T) {
	srv, session := newTestServer(t)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	manifest, _ := zw.Create("cal-export.json")
	manifest.Write([]byte(`{"files":[{"id":"55555555-5555-5555-5555-555555555555","name":"zipfile.png","origName":"z.png","size":7,"mime":"image/png"}]}`))
	bin, _ := zw.Create("files/zipfile.png")
	bin.Write([]byte("pngdata"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/restore", &buf)
	req.Header.Set("Content-Type", "application/zip")
	req.Header.Set("Authorization", "Bearer "+session)
	res, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	// The restored row must serve its binary back.
	got := doJSON(t, srv, http.MethodGet, "/api/files/zipfile.png", "", session)
	if got != "pngdata" {
		t.Fatalf("restored binary: %q", got)
	}
}

func TestMCPPersonResource(t *testing.T) {
	srv, session := newTestServer(t)

	created := doJSON(t, srv, http.MethodPost, "/api/people", `{"name":"Ada Kramer","birthday":"1990-04-02"}`, session)
	var person struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(created), &person); err != nil || person.ID == "" {
		t.Fatalf("create person: %s", created)
	}

	// MCP auth is the API token, not the session cookie.
	settings := doJSON(t, srv, http.MethodGet, "/api/settings", "", session)
	var st struct {
		ApiToken string `json:"apiToken"`
	}
	if err := json.Unmarshal([]byte(settings), &st); err != nil || st.ApiToken == "" {
		t.Fatalf("settings: %s", settings)
	}

	read := func(uri string) string {
		return doJSON(t, srv, http.MethodPost, "/api/mcp",
			`{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"`+uri+`"}}`, st.ApiToken)
	}
	if got := read("cal://people"); !strings.Contains(got, "Ada Kramer") {
		t.Fatalf("cal://people should list the person: %s", got)
	}
	got := read("cal://people/" + person.ID)
	if !strings.Contains(got, "Ada Kramer") || !strings.Contains(got, "relations") {
		t.Fatalf("person resource: %s", got)
	}
	if got := read("cal://people/00000000-0000-0000-0000-000000000000"); strings.Contains(got, "Ada Kramer") {
		t.Fatalf("unknown person must not leak data: %s", got)
	}
}

func TestSharedPeoplePage(t *testing.T) {
	srv, session := newTestServer(t)

	created := doJSON(t, srv, http.MethodPost, "/api/people",
		`{"name":"Dad","birthday":"1960-05-20","dates":[{"label":"Retirement","date":"2025-09-01"}],"phone":"secret"}`, session)
	var person struct{ ID string }
	_ = json.Unmarshal([]byte(created), &person)

	share := doJSON(t, srv, http.MethodPost, "/api/people/share", `{"on":true}`, session)
	var tok struct {
		ShareToken *string `json:"shareToken"`
	}
	if err := json.Unmarshal([]byte(share), &tok); err != nil || tok.ShareToken == nil {
		t.Fatalf("share: %s", share)
	}

	// Anonymous read — private fields must stay home.
	got := doJSON(t, srv, http.MethodGet, "/api/shared/people/"+*tok.ShareToken, "", "")
	if !strings.Contains(got, "Dad") || strings.Contains(got, "secret") {
		t.Fatalf("public payload wrong: %s", got)
	}
	ics := doJSON(t, srv, http.MethodGet, "/api/shared/people/"+*tok.ShareToken+"/calendar.ics", "", "")
	if !strings.Contains(ics, "VEVENT") || !strings.Contains(ics, "Dad") {
		t.Fatalf("ics feed: %s", ics)
	}

	// Revoke → 404.
	doJSON(t, srv, http.MethodPost, "/api/people/share", `{"on":false}`, session)
	res, _ := srv.Client().Get(srv.URL + "/api/shared/people/" + *tok.ShareToken)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("revoked share should 404, got %d", res.StatusCode)
	}
}
