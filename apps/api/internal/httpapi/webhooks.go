package httpapi

// Outbound webhooks: registered URLs get a POSTed JSON event on entry
// create/update/delete, signed with HMAC-SHA256 in X-Cal-Signature.
// Fire-and-forget; a failure to deliver is logged, not retried.

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

var webhookClient = &http.Client{Timeout: 5 * time.Second}

func (s *Server) listWebhooks(c *gin.Context) {
	hooks, err := s.store.Webhooks(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, hooks)
}

func (s *Server) addWebhook(c *gin.Context) {
	var body struct {
		URL string `json:"url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.String(http.StatusBadRequest, "url required")
		return
	}
	u, err := url.Parse(body.URL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		c.String(http.StatusBadRequest, "http(s) url required")
		return
	}
	secret := make([]byte, 16)
	_, _ = rand.Read(secret)
	hook, err := s.store.CreateWebhook(c.Request.Context(), currentUser(c).ID, u.String(), hex.EncodeToString(secret))
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusCreated, hook)
}

func (s *Server) deleteWebhook(c *gin.Context) {
	if err := s.store.DeleteWebhook(c.Request.Context(), currentUser(c).ID, c.Param("id")); err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.Status(http.StatusNoContent)
}

// fireWebhooks posts {event, entry} to every registered URL. Async: callers
// invoke it in a goroutine; delivery failures are logged only.
func (s *Server) fireWebhooks(ctx context.Context, userID, event string, entry any) {
	hooks, err := s.store.Webhooks(ctx, userID)
	if err != nil || len(hooks) == 0 {
		return
	}
	payload, _ := json.Marshal(gin.H{"event": event, "entry": entry, "at": time.Now().UTC()})
	for _, h := range hooks {
		h := h
		go func() {
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.URL, bytes.NewReader(payload))
			if err != nil {
				return
			}
			mac := hmac.New(sha256.New, []byte(h.Secret))
			mac.Write(payload)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Cal-Event", event)
			req.Header.Set("X-Cal-Signature", hex.EncodeToString(mac.Sum(nil)))
			resp, err := webhookClient.Do(req)
			if err != nil {
				log.Printf("webhook %s: %v", h.URL, err)
				return
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode >= 400 {
				log.Printf("webhook %s: %d", h.URL, resp.StatusCode)
			}
		}()
	}
}
