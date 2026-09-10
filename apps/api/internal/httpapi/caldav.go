package httpapi

// CalDAV account handlers: connect a calendar collection, test credentials,
// delete the account, force a sync. Passwords are AES-encrypted at rest.

import (
	"context"
	"net/http"

	"cal/apps/api/internal/caldav"

	"github.com/gin-gonic/gin"
)

func (s *Server) listCaldav(c *gin.Context) {
	accounts, err := s.store.CaldavAccounts(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, accounts)
}

func (s *Server) testCaldav(c *gin.Context) {
	var body struct {
		URL      string `json:"url"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.URL == "" {
		c.String(http.StatusBadRequest, "url required")
		return
	}
	if err := caldav.New(body.URL, body.Username, body.Password).TestConnection(c.Request.Context()); err != nil {
		c.String(http.StatusBadGateway, "connection failed: "+err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) createCaldav(c *gin.Context) {
	var body struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		Username string `json:"username"`
		Password string `json:"password"`
		Color    string `json:"color"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.URL == "" {
		c.String(http.StatusBadRequest, "url required")
		return
	}
	// Verify before persisting — a saved-but-broken account is worse than none.
	if err := caldav.New(body.URL, body.Username, body.Password).TestConnection(c.Request.Context()); err != nil {
		c.String(http.StatusBadGateway, "connection failed: "+err.Error())
		return
	}
	enc, err := s.store.Encrypt(c.Request.Context(), body.Password)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	if body.Name == "" {
		body.Name = body.URL
	}
	if body.Color == "" {
		body.Color = "sky"
	}
	account, err := s.store.CreateCaldavAccount(c.Request.Context(), currentUser(c).ID, body.Name, body.URL, body.Username, enc, body.Color)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to save account")
		return
	}
	// Kick an initial sync so the calendar fills immediately.
	go func() {
		_ = caldav.NewSyncer(s.store).SyncAccount(context.Background(), currentUser(c).ID, account)
	}()
	c.JSON(http.StatusCreated, account)
}

func (s *Server) deleteCaldav(c *gin.Context) {
	if err := s.store.DeleteCaldavAccount(c.Request.Context(), currentUser(c).ID, c.Param("id")); err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) syncCaldav(c *gin.Context) {
	account, userID, err := s.store.CaldavAccountWithSecret(c.Request.Context(), c.Param("id"))
	if err != nil || userID != currentUser(c).ID {
		c.String(http.StatusNotFound, "account not found")
		return
	}
	if err := caldav.NewSyncer(s.store).SyncAccount(c.Request.Context(), userID, account); err != nil {
		c.String(http.StatusBadGateway, "sync failed: "+err.Error())
		return
	}
	c.Status(http.StatusNoContent)
}
