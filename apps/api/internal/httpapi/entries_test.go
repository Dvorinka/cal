package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"cal/apps/api/internal/store"
)

// Regression: entryCols gained link_meta_at while four call sites kept
// hand-rolled scans missing it — create/update/board-cards/trash all 500'd.
// Every path below selects entryCols; the scan must stay in lockstep.
func TestEntryScanParity(t *testing.T) {
	srv, session := newTestServer(t)

	// Create — INSERT ... RETURNING entryCols.
	created := doJSON(t, srv, http.MethodPost, "/api/entries",
		`{"title":"Ship it","type":"task","date":"2026-01-05","linkUrl":"https://example.com"}`, session)
	var e struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(created), &e); err != nil || e.ID == "" {
		t.Fatalf("create entry: %s", created)
	}

	// Update — UPDATE ... RETURNING entryCols.
	updated := doJSON(t, srv, http.MethodPatch, "/api/entries/"+e.ID,
		`{"title":"Ship it v2","completed":true}`, session)
	if !strings.Contains(updated, "Ship it v2") {
		t.Fatalf("update entry: %s", updated)
	}

	// Board cards — SELECT entryCols WHERE board_id.
	board := doJSON(t, srv, http.MethodPost, "/api/boards", `{"name":"B","template":"kanban"}`, session)
	var b struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(board), &b); err != nil || b.ID == "" {
		t.Fatalf("create board: %s", board)
	}
	doJSON(t, srv, http.MethodPatch, "/api/entries/"+e.ID, `{"boardId":"`+b.ID+`"}`, session)
	view := doJSON(t, srv, http.MethodGet, "/api/boards/"+b.ID+"/view", "", session)
	if !strings.Contains(view, "Ship it v2") {
		t.Fatalf("board view: %s", view)
	}

	// Trash — SELECT entryCols WHERE deleted_at.
	doJSON(t, srv, http.MethodDelete, "/api/entries/"+e.ID, "", session)
	trash := doJSON(t, srv, http.MethodGet, "/api/trash", "", session)
	if !strings.Contains(trash, "Ship it v2") {
		t.Fatalf("trash list: %s", trash)
	}
}

// The editor always posts recur "none", including for notes and events.
// Treating that as a recurrence rule rejected every non-task create with 400.
func TestValidEntryRecurNone(t *testing.T) {
	cases := []struct {
		name  string
		input store.EntryInput
		ok    bool
	}{
		{name: "note none", input: store.EntryInput{Title: "test", Type: "note", Date: "2026-11-15", Recur: "none"}, ok: true},
		{name: "event none", input: store.EntryInput{Title: "Meet", Type: "event", Date: "2026-11-15", Recur: "none"}, ok: true},
		{name: "link none", input: store.EntryInput{Title: "Ref", Type: "link", Date: "2026-11-15", Recur: "none"}, ok: true},
		{name: "task none", input: store.EntryInput{Title: "Do", Type: "task", Date: "2026-11-15", Recur: "none"}, ok: true},
		{name: "note omitted", input: store.EntryInput{Title: "test", Type: "note", Date: "2026-11-15"}, ok: true},
		{name: "task daily", input: store.EntryInput{Title: "Do", Type: "task", Date: "2026-11-15", Recur: "daily"}, ok: true},
		{name: "note daily", input: store.EntryInput{Title: "test", Type: "note", Date: "2026-11-15", Recur: "daily"}, ok: false},
		{name: "task hourly", input: store.EntryInput{Title: "Do", Type: "task", Date: "2026-11-15", Recur: "hourly"}, ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := validEntry(tc.input); got != tc.ok {
				t.Fatalf("validEntry() = %v, want %v", got, tc.ok)
			}
		})
	}
}

func TestCreateNoteAndEventWithRecurNone(t *testing.T) {
	srv, session := newTestServer(t)
	note := `{"title":"test","type":"note","date":"2026-11-15","startTime":"","endTime":"","color":"sky","tags":[],"blockedBy":"","content":"","linkUrl":"","recur":"none","remind":null,"workspaceId":""}`
	status, body := doJSONStatus(t, srv, http.MethodPost, "/api/entries", note, session)
	if status != http.StatusCreated || !strings.Contains(body, `"type":"note"`) {
		t.Fatalf("create note: %d %s", status, body)
	}
	event := `{"title":"Standup","type":"event","date":"2026-11-15","startTime":"09:00","endTime":"09:30","color":"sky","tags":[],"recur":"none","workspaceId":"","blockedBy":""}`
	status, body = doJSONStatus(t, srv, http.MethodPost, "/api/entries", event, session)
	if status != http.StatusCreated || !strings.Contains(body, `"type":"event"`) {
		t.Fatalf("create event: %d %s", status, body)
	}
	status, body = doJSONStatus(t, srv, http.MethodPost, "/api/entries",
		`{"title":"nope","type":"note","date":"2026-11-15","recur":"daily"}`, session)
	if status != http.StatusBadRequest {
		t.Fatalf("daily note: %d %s", status, body)
	}
}
