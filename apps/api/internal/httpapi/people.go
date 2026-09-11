package httpapi

// People — personal relationship records. Relation is free-ish text
// (family/friend/partner/colleague/acquaintance by convention); delete is
// permanent — people are not entries, no trash.

import (
	"errors"
	"net/http"
	"strings"

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
	if p.Birthday != "" && !validDate(p.Birthday) {
		return false
	}
	for _, d := range p.Dates {
		if strings.TrimSpace(d.Label) == "" || len(d.Label) > 60 || !validDate(d.Date) {
			return false
		}
	}
	return true
}
