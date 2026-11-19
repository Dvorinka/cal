package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestMCPTrashRoundTrip(t *testing.T) {
	srv, session := newTestServer(t)

	settings := doJSON(t, srv, http.MethodGet, "/api/settings", "", session)
	var st struct {
		ApiToken string `json:"apiToken"`
	}
	if err := json.Unmarshal([]byte(settings), &st); err != nil || st.ApiToken == "" {
		t.Fatalf("settings: %s", settings)
	}
	call := func(name, args string) (string, bool) {
		t.Helper()
		raw := doJSON(t, srv, http.MethodPost, "/api/mcp",
			fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":%q,"arguments":%s}}`, name, args),
			st.ApiToken)
		return mcpToolText(t, raw)
	}

	created := doJSON(t, srv, http.MethodPost, "/api/entries",
		`{"title":"Old note","type":"note","date":"2026-02-01"}`, session)
	var entry struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(created), &entry); err != nil || entry.ID == "" {
		t.Fatalf("create: %s", created)
	}
	live := doJSON(t, srv, http.MethodPost, "/api/entries",
		`{"title":"Keep me","type":"task","date":"2026-02-02"}`, session)
	var keep struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(live), &keep); err != nil || keep.ID == "" {
		t.Fatalf("create live: %s", live)
	}

	text, isErr := call("list_trash", `{}`)
	if isErr || text != "[]" {
		t.Fatalf("empty trash: err=%v %s", isErr, text)
	}
	if text, isErr = call("restore_trash", `{"id":""}`); !isErr || text != "id required" {
		t.Fatalf("blank id: err=%v %s", isErr, text)
	}
	if text, isErr = call("purge_trash", fmt.Sprintf(`{"id":%q}`, keep.ID)); !isErr || text != "entry not found" {
		t.Fatalf("purge live: err=%v %s", isErr, text)
	}

	doJSON(t, srv, http.MethodDelete, "/api/entries/"+entry.ID, "", session)
	text, isErr = call("list_trash", `{}`)
	if isErr || !strings.Contains(text, entry.ID) || !strings.Contains(text, `"title":"Old note"`) || !strings.Contains(text, `"type":"note"`) {
		t.Fatalf("listed: err=%v %s", isErr, text)
	}

	text, isErr = call("restore_trash", fmt.Sprintf(`{"id":%q}`, entry.ID))
	if isErr || text != "restored" {
		t.Fatalf("restore: err=%v %s", isErr, text)
	}
	text, isErr = call("list_trash", `{}`)
	if isErr || strings.Contains(text, entry.ID) {
		t.Fatalf("after restore: err=%v %s", isErr, text)
	}

	doJSON(t, srv, http.MethodDelete, "/api/entries/"+entry.ID, "", session)
	text, isErr = call("purge_trash", fmt.Sprintf(`{"id":%q}`, entry.ID))
	if isErr || text != "purged" {
		t.Fatalf("purge: err=%v %s", isErr, text)
	}
	text, isErr = call("list_trash", `{}`)
	if isErr || strings.Contains(text, entry.ID) {
		t.Fatalf("after purge: err=%v %s", isErr, text)
	}
	if text, isErr = call("purge_trash", fmt.Sprintf(`{"id":%q}`, entry.ID)); !isErr || text != "entry not found" {
		t.Fatalf("purge again: err=%v %s", isErr, text)
	}
}
