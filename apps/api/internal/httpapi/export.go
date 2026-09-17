package httpapi

// Subscribable ICS feed of the user's own entries — the inverse of feed
// import. Authenticated by the read-only widget token, so the URL can live in
// a calendar app without exposing full API access.

import (
	"net/http"
	"time"

	"cal/apps/api/internal/ical"
	"cal/apps/api/internal/store"

	"github.com/gin-gonic/gin"
)

// restoreJSON accepts the /export payload and re-inserts entries, people,
// person links and timeline items. Additive: existing IDs are skipped, so a
// restore never clobbers current data.
func (s *Server) restoreJSON(c *gin.Context) {
	var body struct {
		Entries        []store.Entry          `json:"entries"`
		People         []store.Person         `json:"people"`
		PersonLinks    []store.PersonRelation `json:"personLinks"`
		PersonTimeline []store.TimelineItem   `json:"personTimeline"`
	}
	if err := c.ShouldBindJSON(&body); err != nil ||
		(len(body.Entries) == 0 && len(body.People) == 0 && len(body.PersonLinks) == 0 && len(body.PersonTimeline) == 0) {
		c.String(http.StatusBadRequest, "expected a Cal export file")
		return
	}
	if len(body.Entries) > 50000 || len(body.People) > 20000 {
		c.String(http.StatusBadRequest, "file too large")
		return
	}
	userID := currentUser(c).ID
	imported, err := s.store.RestoreEntries(c.Request.Context(), userID, body.Entries)
	if err != nil {
		c.String(http.StatusInternalServerError, "restore failed")
		return
	}
	// People first — links and timeline reference person IDs.
	people, err := s.store.RestorePeople(c.Request.Context(), userID, body.People)
	if err != nil {
		c.String(http.StatusInternalServerError, "restore failed")
		return
	}
	links, err := s.store.RestorePersonLinks(c.Request.Context(), userID, body.PersonLinks)
	if err != nil {
		c.String(http.StatusInternalServerError, "restore failed")
		return
	}
	timeline, err := s.store.RestorePersonTimeline(c.Request.Context(), userID, body.PersonTimeline)
	if err != nil {
		c.String(http.StatusInternalServerError, "restore failed")
		return
	}
	c.JSON(http.StatusOK, gin.H{"restored": imported, "people": people, "links": links, "timeline": timeline})
}

func (s *Server) exportICS(c *gin.Context) {
	user, err := s.store.UserByWidgetToken(c.Request.Context(), c.Query("token"))
	if err != nil {
		c.String(http.StatusUnauthorized, "invalid token")
		return
	}
	settings, _ := s.store.Settings(c.Request.Context(), user.ID)
	loc := time.Local
	if l, err := time.LoadLocation(settings.Timezone); err == nil {
		loc = l
	}
	entries, err := s.store.ListEntries(c.Request.Context(), user.ID, "", "", "")
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	events := make([]ical.Event, 0, len(entries))
	for _, e := range entries {
		if e.Type == "note" || e.Type == "link" {
			continue
		}
		events = append(events, entryToEvent(e, loc))
	}
	c.Data(http.StatusOK, "text/calendar; charset=utf-8", []byte(ical.EncodeCalendar(events)))
}

func entryToEvent(e store.Entry, loc *time.Location) ical.Event {
	ev := ical.Event{
		UID:         "cal-" + e.ID,
		Summary:     e.Title,
		Description: e.Content,
		URL:         e.LinkURL,
		Completed:   e.Type == "task" && e.Completed,
		TZ:          loc,
	}
	day, err := time.Parse(time.DateOnly, e.Date)
	if err != nil {
		return ev
	}
	if e.StartTime == nil || *e.StartTime == "" {
		ev.AllDay = true
		ev.Start = day
	} else {
		if t, err := time.ParseInLocation("2006-01-02 15:04", e.Date+" "+*e.StartTime, loc); err == nil {
			ev.Start = t
		}
		if e.EndTime != nil {
			if t, err := time.ParseInLocation("2006-01-02 15:04", e.Date+" "+*e.EndTime, loc); err == nil {
				ev.End = t
			}
		}
	}
	switch e.Recur {
	case "daily":
		ev.Recurrence = &ical.RRule{Freq: "DAILY"}
	case "weekly":
		ev.Recurrence = &ical.RRule{Freq: "WEEKLY"}
	case "monthly":
		ev.Recurrence = &ical.RRule{Freq: "MONTHLY"}
	case "yearly":
		ev.Recurrence = &ical.RRule{Freq: "YEARLY"}
	}
	return ev
}
