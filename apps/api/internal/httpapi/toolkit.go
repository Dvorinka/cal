package httpapi

// Toolkit endpoints: focus timer, card activity, trash, board meta/share,
// tags, agenda export.

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"cal/apps/api/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"context"
	"encoding/json"
	"io"
	"regexp"
	"time"
)

// --- Focus timer (one running per user; solidtime model) ---

func (s *Server) startTimer(c *gin.Context) {
	var body struct {
		EntryID   *string  `json:"entryId"`
		Note      string   `json:"note"`
		Planned   int      `json:"planned"`
		Billable  bool     `json:"billable"`
		Rate      *float64 `json:"rate"`
		ProjectID *string  `json:"projectId"`
	}
	_ = c.ShouldBindJSON(&body)
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
	t, err := s.store.StartTimer(c.Request.Context(), currentUser(c).ID, body.EntryID, body.Note, body.Planned, body.Billable, body.Rate, body.ProjectID)
	if err != nil {
		c.String(http.StatusConflict, "a timer is already running")
		return
	}
	c.JSON(http.StatusCreated, t)
}

func (s *Server) stopTimer(c *gin.Context) {
	t, err := s.store.StopTimer(c.Request.Context(), currentUser(c).ID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.String(http.StatusNotFound, "no running timer")
		return
	}
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, t)
}

func (s *Server) currentTimer(c *gin.Context) {
	t, err := s.store.CurrentTimer(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	if t == nil {
		c.Status(http.StatusNoContent)
		return
	}
	c.JSON(http.StatusOK, t)
}

func (s *Server) timeSummary(c *gin.Context) {
	sum, err := s.store.TimeSummary(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, sum)
}

// timeLog — the timesheet: finished sessions in a range.
func (s *Server) timeLog(c *gin.Context) {
	from := c.DefaultQuery("from", time.Now().AddDate(0, 0, -14).Format("2006-01-02"))
	to := c.DefaultQuery("to", time.Now().Format("2006-01-02"))
	items, err := s.store.TimeLog(c.Request.Context(), currentUser(c).ID, from, to)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) deleteTimeEntry(c *gin.Context) {
	if err := s.store.DeleteTimeEntry(c.Request.Context(), currentUser(c).ID, c.Param("id")); errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "not found")
		return
	} else if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.Status(http.StatusNoContent)
}

// timeExport — solidtime-compatible CSV/JSON: description, project, start,
// end, duration, billable, rate, amount.
func (s *Server) timeExport(c *gin.Context) {
	from := c.DefaultQuery("from", time.Now().AddDate(0, 0, -90).Format("2006-01-02"))
	to := c.DefaultQuery("to", time.Now().Format("2006-01-02"))
	items, err := s.store.TimeLog(c.Request.Context(), currentUser(c).ID, from, to)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	type row struct {
		Description string   `json:"description"`
		Project     string   `json:"project"`
		Start       string   `json:"start"`
		End         string   `json:"end"`
		Duration    int      `json:"duration"` // seconds
		Billable    bool     `json:"billable"`
		Rate        *float64 `json:"rate,omitempty"`
		Amount      float64  `json:"amount"`
	}
	rows := make([]row, 0, len(items))
	for _, t := range items {
		if t.EndAt == nil {
			continue
		}
		dur := int(t.EndAt.Sub(t.StartAt).Seconds())
		var amount float64
		if t.Billable && t.Rate != nil {
			amount = float64(dur) / 3600 * *t.Rate
		}
		desc := t.Title
		if desc == "" {
			desc = t.Note
		}
		rows = append(rows, row{desc, t.Project, t.StartAt.Format(time.RFC3339), t.EndAt.Format(time.RFC3339), dur, t.Billable, t.Rate, amount})
	}
	if c.DefaultQuery("format", "csv") == "json" {
		c.JSON(http.StatusOK, rows)
		return
	}
	var b strings.Builder
	b.WriteString("description,project,start,end,duration,billable,rate,amount\n")
	for _, r := range rows {
		rate := ""
		if r.Rate != nil {
			rate = strconv.FormatFloat(*r.Rate, 'f', 2, 64)
		}
		b.WriteString(fmt.Sprintf("%q,%q,%s,%s,%d,%t,%s,%.2f\n",
			r.Description, r.Project, r.Start, r.End, r.Duration, r.Billable, rate, r.Amount))
	}
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="cal-time.csv"`)
	c.String(http.StatusOK, b.String())
}

// --- Card activity ---

func (s *Server) entryActivity(c *gin.Context) {
	items, err := s.store.EntryActivity(c.Request.Context(), currentUser(c).ID, c.Param("id"))
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, items)
}

// --- Trash ---

func (s *Server) listTrash(c *gin.Context) {
	items, err := s.store.ListTrash(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) restoreEntry(c *gin.Context) {
	if err := s.store.RestoreEntry(c.Request.Context(), currentUser(c).ID, c.Param("id")); errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "not found")
		return
	} else if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) purgeEntry(c *gin.Context) {
	if err := s.store.PurgeEntry(c.Request.Context(), currentUser(c).ID, c.Param("id")); errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "not found")
		return
	} else if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.Status(http.StatusNoContent)
}

// --- Board meta + public share ---

func (s *Server) updateBoard(c *gin.Context) {
	var body struct {
		Description *string `json:"description"`
		TargetDate  *string `json:"targetDate"`
	}
	_ = c.ShouldBindJSON(&body)
	if err := s.store.SetBoardMeta(c.Request.Context(), currentUser(c).ID, c.Param("id"), body.Description, body.TargetDate); errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "not found")
		return
	} else if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) shareBoard(c *gin.Context) {
	var body struct {
		On bool `json:"on"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.String(http.StatusBadRequest, "invalid")
		return
	}
	token, err := s.store.SetBoardShare(c.Request.Context(), currentUser(c).ID, c.Param("id"), body.On)
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "not found")
		return
	} else if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, gin.H{"shareToken": token})
}

func (s *Server) serveSharedBoard(c *gin.Context) {
	board, err := s.store.SharedBoard(c.Request.Context(), c.Param("token"))
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "not found")
		return
	} else if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, board)
}

// --- Tags ---

func (s *Server) tagCounts(c *gin.Context) {
	tags, err := s.store.TagCounts(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, tags)
}

// --- Agenda export ---

func (s *Server) agendaMarkdown(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	if days < 1 || days > 60 {
		days = 7
	}
	from := time.Now().Format("2006-01-02")
	to := time.Now().AddDate(0, 0, days).Format("2006-01-02")
	entries, err := s.store.ListEntries(c.Request.Context(), currentUser(c).ID, from, to, "")
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Agenda — %s → %s\n\n", from, to))
	day := ""
	for _, e := range entries {
		if e.Date != day {
			day = e.Date
			b.WriteString(fmt.Sprintf("## %s\n\n", day))
		}
		box := "- "
		if e.Type == "task" {
			if e.Completed {
				box = "- [x] "
			} else {
				box = "- [ ] "
			}
		}
		when := ""
		if e.StartTime != nil && *e.StartTime != "" {
			when = *e.StartTime + " "
		}
		b.WriteString(fmt.Sprintf("%s%s%s\n", box, when, e.Title))
	}
	c.Header("Content-Type", "text/markdown; charset=utf-8")
	c.String(http.StatusOK, b.String())
}

// --- Link enrichment: YouTube oEmbed or generic unfurl, stored on the entry ---

var reYouTube = regexp.MustCompile(`(?:youtube\.com/(?:watch\?[^ ]*v=|shorts/|embed/)|youtu\.be/)([A-Za-z0-9_-]{6,20})`)

type oembedResp struct {
	Title      string `json:"title"`
	AuthorName string `json:"author_name"`
	Thumbnail  string `json:"thumbnail_url"`
}

// enrichLink fills link_desc/link_image/link_favicon/link_video_id on a link
// entry. YouTube URLs go through oEmbed (no API key); everything else unfurls.
func (s *Server) enrichLink(userID, entryID, raw string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if m := reYouTube.FindStringSubmatch(raw); m != nil {
		vid := m[1]
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
			"https://www.youtube.com/oembed?url=https://www.youtube.com/watch?v="+vid+"&format=json", nil)
		if resp, err := unfurlClient.Do(req); err == nil && resp.StatusCode == 200 {
			defer resp.Body.Close()
			var oe oembedResp
			if json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&oe) == nil {
				s.store.SetLinkMeta(ctx, userID, entryID, oe.AuthorName, oe.Thumbnail, "", vid, oe.Title)
				return
			}
		}
		// oEmbed failed — still record the video id so the card can build the thumb.
		s.store.SetLinkMeta(ctx, userID, entryID, "", "https://i.ytimg.com/vi/"+vid+"/hqdefault.jpg", "", vid, "")
		return
	}

	p, err := unfurlURL(ctx, raw)
	if err != nil {
		return
	}
	s.store.SetLinkMeta(ctx, userID, entryID, p.Description, p.Image, p.Favicon, "", p.Title)
}

// globalSearch — one endpoint across entries, files, and boards.
func (s *Server) globalSearch(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		c.JSON(http.StatusOK, gin.H{"entries": []any{}, "files": []any{}})
		return
	}
	ctx := c.Request.Context()
	uid := currentUser(c).ID
	entries, err := s.store.GlobalSearch(ctx, uid, q)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	files, _ := s.store.SearchFiles(ctx, uid, q)
	boards, _ := s.store.SearchBoards(ctx, uid, q)
	c.JSON(http.StatusOK, gin.H{"entries": entries, "files": files, "boards": boards})
}
