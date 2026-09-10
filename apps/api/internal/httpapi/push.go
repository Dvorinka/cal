package httpapi

// Web push: VAPID keys are generated once and persisted in server_config so
// subscriptions survive restarts. A 60s scheduler finds due reminders and
// pushes to every subscription of the entry's owner.

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"cal/apps/api/internal/store"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/gin-gonic/gin"
)

// vapidKeys loads or generates the server-wide VAPID keypair.
func vapidKeys(ctx context.Context, s *store.Store) (pub, priv string, err error) {
	pub, err = s.ServerConfig(ctx, "vapid_public")
	if err == nil {
		priv, err = s.ServerConfig(ctx, "vapid_private")
		return pub, priv, err
	}
	priv, pub, err = webpush.GenerateVAPIDKeys()
	if err != nil {
		return "", "", err
	}
	if err := s.SetServerConfig(ctx, "vapid_public", pub); err != nil {
		return "", "", err
	}
	if err := s.SetServerConfig(ctx, "vapid_private", priv); err != nil {
		return "", "", err
	}
	return pub, priv, nil
}

func (s *Server) pushVapid(c *gin.Context) {
	pub, _, err := vapidKeys(c.Request.Context(), s.store)
	if err != nil {
		c.String(http.StatusInternalServerError, "push not available")
		return
	}
	c.JSON(http.StatusOK, gin.H{"publicKey": pub})
}

func (s *Server) subscribePush(c *gin.Context) {
	var body struct {
		Endpoint string `json:"endpoint"`
		Keys     struct {
			P256dh string `json:"p256dh"`
			Auth   string `json:"auth"`
		} `json:"keys"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Endpoint == "" || body.Keys.P256dh == "" {
		c.String(http.StatusBadRequest, "invalid subscription")
		return
	}
	if err := s.store.UpsertPushSub(c.Request.Context(), currentUser(c).ID, body.Endpoint, body.Keys.P256dh, body.Keys.Auth); err != nil {
		c.String(http.StatusInternalServerError, "failed to save subscription")
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) unsubscribePush(c *gin.Context) {
	var body struct {
		Endpoint string `json:"endpoint"`
	}
	_ = c.ShouldBindJSON(&body)
	if body.Endpoint != "" {
		_ = s.store.DeletePushSub(c.Request.Context(), body.Endpoint)
	}
	c.Status(http.StatusNoContent)
}

// PushLoop fires due reminders every minute. Runs for the life of the process.
func PushLoop(ctx context.Context, s *store.Store, every time.Duration) {
	pub, priv, err := vapidKeys(ctx, s)
	if err != nil {
		log.Printf("push: vapid unavailable: %v", err)
		return
	}
	tick := func() {
		due, err := s.DueReminders(ctx)
		if err != nil {
			return
		}
		for _, entry := range due {
			subs, err := s.PushSubs(ctx, entry.OwnerID)
			if err != nil || len(subs) == 0 {
				continue
			}
			payload, _ := json.Marshal(gin.H{
				"title": entry.Title,
				"body":  "Starts at " + deref(entry.StartTime),
				"tag":   entry.ID,
			})
			for _, sub := range subs {
				resp, err := webpush.SendNotification(payload, &webpush.Subscription{
					Endpoint: sub.Endpoint,
					Keys:     webpush.Keys{P256dh: sub.P256dh, Auth: sub.Auth},
				}, &webpush.Options{
					Subscriber:      "cal@localhost",
					VAPIDPublicKey:  pub,
					VAPIDPrivateKey: priv,
					TTL:             300,
				})
				if err != nil {
					continue
				}
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNotFound {
					_ = s.DeletePushSub(ctx, sub.Endpoint)
				}
			}
			_ = s.MarkReminded(ctx, entry.ID)
		}
	}
	tick()
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tick()
		}
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
