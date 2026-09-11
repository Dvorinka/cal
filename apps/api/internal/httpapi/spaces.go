package httpapi

// Phase 8 surface: workspaces, feature modules, dashboard aggregation,
// file tags, time-entry create/edit (solidtime parity), saved filters,
// link-preview refresh, entry dependencies.

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"cal/apps/api/internal/store"

	"github.com/gin-gonic/gin"
)

func splitTags(raw string) []string {
	out := []string{}
	for _, t := range strings.Split(raw, ",") {
		if t = strings.TrimSpace(strings.TrimPrefix(t, "#")); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func nilIfEmptyStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// --- Workspaces ---

func (s *Server) listWorkspaces(c *gin.Context) {
	items, err := s.store.ListWorkspaces(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) createWorkspace(c *gin.Context) {
	var body struct {
		Name  string `json:"name"`
		Color string `json:"color"`
		Icon  string `json:"icon"`
	}
	if !bind(c, &body) || strings.TrimSpace(body.Name) == "" || len(body.Name) > 60 {
		c.String(http.StatusBadRequest, "name required")
		return
	}
	w, err := s.store.CreateWorkspace(c.Request.Context(), currentUser(c).ID, strings.TrimSpace(body.Name), body.Color, body.Icon)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusCreated, w)
}

func (s *Server) updateWorkspace(c *gin.Context) {
	var body struct {
		Name     string `json:"name"`
		Color    string `json:"color"`
		Icon     string `json:"icon"`
		Position *int   `json:"position"`
	}
	_ = c.ShouldBindJSON(&body)
	if len(body.Name) > 60 {
		c.String(http.StatusBadRequest, "name too long")
		return
	}
	err := s.store.UpdateWorkspace(c.Request.Context(), currentUser(c).ID, c.Param("id"), body.Name, body.Color, body.Icon, body.Position)
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "not found")
		return
	} else if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) deleteWorkspace(c *gin.Context) {
	err := s.store.DeleteWorkspace(c.Request.Context(), currentUser(c).ID, c.Param("id"))
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "not found")
		return
	} else if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Dashboard ---

// dashboard returns the Today-page widget payload in one request:
// counts, completion, deadlines, weekly activity, merged feed, timer.
func (s *Server) dashboard(c *gin.Context) {
	d, err := s.store.Dashboard(c.Request.Context(), currentUser(c).ID, c.Query("workspace"))
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, d)
}

// --- Files: tags + workspace patch ---

func (s *Server) updateFile(c *gin.Context) {
	var raw map[string]any
	if !bind(c, &raw) {
		c.String(http.StatusBadRequest, "invalid")
		return
	}
	var tags []string
	var ws *string
	if v, ok := raw["tags"]; ok {
		list, ok := v.([]any)
		if !ok {
			c.String(http.StatusBadRequest, "tags must be an array")
			return
		}
		for _, t := range list {
			if s, ok := t.(string); ok {
				tags = append(tags, s)
			}
		}
	}
	if v, ok := raw["workspaceId"]; ok {
		if s, ok := v.(string); ok {
			ws = &s // "" clears to Personal
		} else if v == nil {
			empty := ""
			ws = &empty
		}
	}
	f, err := s.store.UpdateFile(c.Request.Context(), currentUser(c).ID, c.Param("id"), tags, ws)
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "not found")
		return
	} else if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, f)
}

// --- Time entries: manual create + edit (solidtime parity) ---

func (s *Server) createTimeEntry(c *gin.Context) {
	var body struct {
		EntryID   *string  `json:"entryId"`
		Note      string   `json:"note"`
		StartAt   *string  `json:"startAt"`
		EndAt     *string  `json:"endAt"`
		Planned   int      `json:"planned"`
		Billable  bool     `json:"billable"`
		Rate      *float64 `json:"rate"`
		ProjectID *string  `json:"projectId"`
		Tags      []string `json:"tags"`
	}
	if !bind(c, &body) {
		c.String(http.StatusBadRequest, "invalid")
		return
	}
	startAt, err := parseTimePtr(body.StartAt)
	if err != nil {
		c.String(http.StatusBadRequest, "bad startAt")
		return
	}
	endAt, err := parseTimePtr(body.EndAt)
	if err != nil {
		c.String(http.StatusBadRequest, "bad endAt")
		return
	}
	if startAt != nil && endAt != nil && !endAt.After(*startAt) {
		c.String(http.StatusBadRequest, "end must be after start")
		return
	}
	if body.EntryID != nil && *body.EntryID == "" {
		body.EntryID = nil
	}
	if body.ProjectID != nil && *body.ProjectID == "" {
		body.ProjectID = nil
	}
	if body.Rate == nil {
		if st, err := s.store.Settings(c.Request.Context(), currentUser(c).ID); err == nil {
			body.Rate = st.DefaultRate
		}
	}
	t, err := s.store.CreateTimeEntry(c.Request.Context(), currentUser(c).ID, body.EntryID, body.Note, startAt, endAt, body.Planned, body.Billable, body.Rate, body.ProjectID, body.Tags)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusCreated, t)
}

func (s *Server) updateTimeEntry(c *gin.Context) {
	var raw map[string]json.RawMessage
	if !bind(c, &raw) {
		c.String(http.StatusBadRequest, "invalid")
		return
	}
	patch, ok := parseTimePatch(raw)
	if !ok {
		c.String(http.StatusBadRequest, "invalid")
		return
	}
	t, err := s.store.UpdateTimeEntry(c.Request.Context(), currentUser(c).ID, c.Param("id"), patch)
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "not found")
		return
	} else if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, t)
}

func parseTimePtr(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02 15:04"} {
		if t, err := time.Parse(layout, *s); err == nil {
			return &t, nil
		}
	}
	return nil, errors.New("unparsable time")
}

// parseTimePatch decodes a PATCH body with explicit-null semantics:
// {"endAt": null} clears the field; absent keys are left alone.
func parseTimePatch(raw map[string]json.RawMessage) (store.TimeEntryPatch, bool) {
	var p store.TimeEntryPatch
	for key, value := range raw {
		isNull := string(value) == "null"
		switch key {
		case "startAt":
			var v string
			if isNull || json.Unmarshal(value, &v) != nil {
				return p, false
			}
			t, err := parseTimePtr(&v)
			if err != nil {
				return p, false
			}
			p.StartAt = t
		case "endAt":
			if isNull {
				p.ClearEnd = true
				continue
			}
			var v string
			if json.Unmarshal(value, &v) != nil {
				return p, false
			}
			t, err := parseTimePtr(&v)
			if err != nil {
				return p, false
			}
			p.EndAt = t
		case "note":
			var v string
			if isNull || json.Unmarshal(value, &v) != nil {
				return p, false
			}
			p.Note = &v
		case "tags":
			var v []string
			if isNull || json.Unmarshal(value, &v) != nil {
				return p, false
			}
			p.Tags = v
		case "billable":
			var v bool
			if isNull || json.Unmarshal(value, &v) != nil {
				return p, false
			}
			p.Billable = &v
		case "rate":
			if isNull {
				p.ClearRate = true
				continue
			}
			var v float64
			if json.Unmarshal(value, &v) != nil || v < 0 {
				return p, false
			}
			p.Rate = &v
		case "projectId":
			if isNull {
				p.ClearProject = true
				continue
			}
			var v string
			if json.Unmarshal(value, &v) != nil {
				return p, false
			}
			if v == "" {
				p.ClearProject = true
			} else {
				p.ProjectID = &v
			}
		case "entryId":
			if isNull {
				p.ClearEntry = true
				continue
			}
			var v string
			if json.Unmarshal(value, &v) != nil {
				return p, false
			}
			if v == "" {
				p.ClearEntry = true
			} else {
				p.EntryID = &v
			}
		case "planned":
			if isNull {
				p.ClearPlanned = true
				continue
			}
			var v int
			if json.Unmarshal(value, &v) != nil || v < 0 {
				return p, false
			}
			p.Planned = &v
		default:
			return p, false
		}
	}
	return p, true
}

// --- Saved filters ---

func (s *Server) listFilters(c *gin.Context) {
	items, err := s.store.ListFilters(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) createFilter(c *gin.Context) {
	var body struct {
		Name   string         `json:"name"`
		Filter map[string]any `json:"filter"`
	}
	if !bind(c, &body) || strings.TrimSpace(body.Name) == "" || len(body.Name) > 60 {
		c.String(http.StatusBadRequest, "name required")
		return
	}
	f, err := s.store.CreateFilter(c.Request.Context(), currentUser(c).ID, strings.TrimSpace(body.Name), body.Filter)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusCreated, f)
}

func (s *Server) deleteFilter(c *gin.Context) {
	err := s.store.DeleteFilter(c.Request.Context(), currentUser(c).ID, c.Param("id"))
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "not found")
		return
	} else if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.Status(http.StatusNoContent)
}

// refreshLink re-fetches unfurl/oEmbed metadata for a link entry.
func (s *Server) refreshLink(c *gin.Context) {
	entry, err := s.store.Entry(c.Request.Context(), currentUser(c).ID, c.Param("id"))
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "not found")
		return
	} else if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	if entry.LinkURL == "" {
		c.String(http.StatusBadRequest, "not a link")
		return
	}
	go s.enrichLink(currentUser(c).ID, entry.ID, entry.LinkURL)
	c.Status(http.StatusAccepted)
}
