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
	"time"
)

// --- Focus timer (one running per user; solidtime model) ---

func (s *Server) startTimer(c *gin.Context) {
	var body struct {
		EntryID *string `json:"entryId"`
		Note    string  `json:"note"`
	}
	_ = c.ShouldBindJSON(&body)
	if body.EntryID != nil && *body.EntryID == "" {
		body.EntryID = nil
	}
	t, err := s.store.StartTimer(c.Request.Context(), currentUser(c).ID, body.EntryID, body.Note)
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
