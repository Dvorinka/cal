package httpapi

// People — personal relationship records. Relation is free-ish text
// (family/friend/partner/colleague/acquaintance by convention); delete is
// permanent — people are not entries, no trash.

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"cal/apps/api/internal/ical"
	"cal/apps/api/internal/store"

	"github.com/gin-gonic/gin"
)

func (s *Server) listPeople(c *gin.Context) {
	people, err := s.store.ListPeople(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, people)
}

func (s *Server) createPerson(c *gin.Context) {
	var body store.PersonInput
	if !bind(c, &body) {
		c.String(http.StatusBadRequest, "invalid json")
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if !validPerson(body) {
		c.String(http.StatusBadRequest, "invalid person")
		return
	}
	if !s.checkPersonWorkspace(c, &body) {
		return
	}
	person, err := s.store.CreatePerson(c.Request.Context(), currentUser(c).ID, body)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusCreated, person)
}

func (s *Server) updatePerson(c *gin.Context) {
	var body store.PersonInput
	if !bind(c, &body) {
		c.String(http.StatusBadRequest, "invalid json")
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if !validPerson(body) {
		c.String(http.StatusBadRequest, "invalid person")
		return
	}
	if !s.checkPersonWorkspace(c, &body) {
		return
	}
	person, err := s.store.UpdatePerson(c.Request.Context(), currentUser(c).ID, c.Param("id"), body)
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "not found")
		return
	} else if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, person)
}

func (s *Server) deletePerson(c *gin.Context) {
	err := s.store.DeletePerson(c.Request.Context(), currentUser(c).ID, c.Param("id"))
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "not found")
		return
	} else if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.Status(http.StatusNoContent)
}

// checkPersonWorkspace verifies an assigned workspace belongs to the caller;
// empty/nil means Personal. Mirrors the entry handler's check.
func (s *Server) checkPersonWorkspace(c *gin.Context, body *store.PersonInput) bool {
	if body.WorkspaceID != nil && *body.WorkspaceID != "" {
		if !s.store.WorkspaceOwned(c.Request.Context(), currentUser(c).ID, *body.WorkspaceID) {
			c.String(http.StatusBadRequest, "unknown workspace")
			return false
		}
	} else {
		body.WorkspaceID = nil
	}
	return true
}

func validPerson(p store.PersonInput) bool {
	if p.Name == "" || len(p.Name) > 120 {
		return false
	}
	if len(p.Relation) > 60 || len(p.Color) > 30 || len(p.Notes) > 20000 {
		return false
	}
	if len(p.Nickname) > 120 || len(p.Avatar) > 300 || len(p.Phone) > 60 ||
		len(p.Email) > 200 || len(p.Address) > 500 ||
		len(p.GiftIdeas) > 5000 || len(p.Interests) > 5000 {
		return false
	}
	if p.Email != "" && !strings.Contains(p.Email, "@") {
		return false
	}
	if p.BirthdayRemind != nil && (*p.BirthdayRemind < 0 || *p.BirthdayRemind > 365) {
		return false
	}
	if p.Birthday != "" && !validDate(p.Birthday) {
		return false
	}
	if len(p.Fields) > 50 || len(p.Links) > 20 || len(p.Tags) > 20 {
		return false
	}
	for _, f := range p.Fields {
		if strings.TrimSpace(f.Key) == "" || len(f.Key) > 120 || len(f.Value) > 2000 {
			return false
		}
	}
	for _, l := range p.Links {
		if len(l.Platform) > 60 || len(l.URL) > 500 || l.URL == "" {
			return false
		}
	}
	for _, t := range p.Tags {
		if strings.TrimSpace(t) == "" || len(t) > 40 {
			return false
		}
	}
	for _, d := range p.Dates {
		if strings.TrimSpace(d.Label) == "" || len(d.Label) > 60 || !validDate(d.Date) {
			return false
		}
		if d.RemindDays != nil && (*d.RemindDays < 0 || *d.RemindDays > 365) {
			return false
		}
	}
	return true
}

// --- Public people page: one token per user exposes names + dates ---
//
// Payload is deliberately narrow: name, nickname, relation, birthday and
// custom dates. Notes, contact details, gift ideas and files never leave.

type publicPerson struct {
	Name     string             `json:"name"`
	Nickname string             `json:"nickname,omitempty"`
	Relation string             `json:"relation,omitempty"`
	Color    string             `json:"color,omitempty"`
	Birthday *string            `json:"birthday,omitempty"`
	Dates    []store.PersonDate `json:"dates"`
}

func (s *Server) sharePeople(c *gin.Context) {
	var body struct {
		On bool `json:"on"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.String(http.StatusBadRequest, "invalid")
		return
	}
	token, err := s.store.SetPeopleShare(c.Request.Context(), currentUser(c).ID, body.On)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"shareToken": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"shareToken": token})
}

// sharedPeopleList resolves the token and returns the trimmed person rows.
func (s *Server) sharedPeopleList(c *gin.Context) (string, []store.Person, bool) {
	userID, err := s.store.SharedPeopleOwner(c.Request.Context(), c.Param("token"))
	if err != nil {
		c.String(http.StatusNotFound, "not found")
		return "", nil, false
	}
	people, err := s.store.ListPeople(c.Request.Context(), userID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return "", nil, false
	}
	return userID, people, true
}

func (s *Server) serveSharedPeople(c *gin.Context) {
	_, people, ok := s.sharedPeopleList(c)
	if !ok {
		return
	}
	out := make([]publicPerson, 0, len(people))
	for _, p := range people {
		out = append(out, publicPerson{
			Name: p.Name, Nickname: p.Nickname, Relation: p.Relation,
			Color: p.Color, Birthday: p.Birthday, Dates: p.Dates,
		})
	}
	c.JSON(http.StatusOK, gin.H{"people": out})
}

// serveSharedPeopleICS — the same dates as a subscribable calendar feed.
func (s *Server) serveSharedPeopleICS(c *gin.Context) {
	_, people, ok := s.sharedPeopleList(c)
	if !ok {
		return
	}
	events := make([]ical.Event, 0, len(people))
	yearly := &ical.RRule{Freq: "YEARLY"}
	for _, p := range people {
		if p.Birthday != nil {
			if day, err := time.Parse(time.DateOnly, *p.Birthday); err == nil {
				events = append(events, ical.Event{
					UID: "cal-person-" + p.ID + "-bday", Summary: p.Name + " — birthday",
					AllDay: true, Start: day, Recurrence: yearly,
				})
			}
		}
		for _, d := range p.Dates {
			if day, err := time.Parse(time.DateOnly, d.Date); err == nil {
				events = append(events, ical.Event{
					UID:     "cal-person-" + p.ID + "-" + d.Date + "-" + d.Label,
					Summary: p.Name + " — " + d.Label,
					AllDay:  true, Start: day, Recurrence: yearly,
				})
			}
		}
	}
	c.Data(http.StatusOK, "text/calendar; charset=utf-8", []byte(ical.EncodeCalendar(events)))
}
