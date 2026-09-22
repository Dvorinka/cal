package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
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
