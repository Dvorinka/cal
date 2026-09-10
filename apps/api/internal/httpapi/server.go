package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"cal/apps/api/internal/auth"
	"cal/apps/api/internal/calendar"
	"cal/apps/api/internal/store"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type Server struct {
	store   *store.Store
	holiday *calendar.HolidayCache
	secure  bool
	dataDir string
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func New(st *store.Store, holidays *calendar.HolidayCache) *gin.Engine {
	secure := os.Getenv("SESSION_SECURE") == "true"
	server := &Server{store: st, holiday: holidays, secure: secure, dataDir: env("DATA_DIR", "./data")}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(securityHeaders())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{env("WEB_ORIGIN", "http://localhost:5173")},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	api := router.Group("/api")
	api.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	authLimited := api.Group("/auth", newRateLimiter(8, time.Minute))
	authLimited.POST("/register", server.register)
	authLimited.POST("/login", server.login)
	api.POST("/auth/logout", server.logout)
	api.GET("/holidays", server.holidays)
	api.GET("/holidays/countries", server.holidayCountries)

	authed := api.Group("")
	authed.Use(server.requireUser)
	authed.GET("/me", server.me)
	authed.GET("/entries", server.entries)
	authed.POST("/entries", server.createEntry)
	authed.PATCH("/entries/:id", server.updateEntry)
	authed.DELETE("/entries/:id", server.deleteEntry)
	authed.GET("/entries/:id/revisions", server.entryRevisions)
	authed.POST("/entries/:id/restore/:rev", server.restoreRevision)
	authed.GET("/settings", server.settings)
	authed.PUT("/settings", server.updateSettings)
	authed.GET("/export", server.export)
	authed.GET("/feeds", server.listFeeds)
	authed.POST("/feeds", newRateLimiter(10, time.Minute), server.createFeed)
	authed.DELETE("/feeds/:id", server.deleteFeed)
	authed.POST("/feeds/:id/refresh", server.refreshFeed)
	authed.GET("/feed-events", server.feedEvents)
	authed.POST("/import", server.importICS)
	authed.POST("/settings/widget-token", server.rotateWidgetToken)
	authed.POST("/settings/api-token", server.rotateApiToken)
	authed.POST("/restore", server.restoreJSON)
	authed.GET("/sessions", server.listSessions)
	authed.DELETE("/sessions/:id", server.revokeSession)
	authed.POST("/password", server.changePassword)
	authed.GET("/webhooks", server.listWebhooks)
	authed.POST("/webhooks", server.addWebhook)
	authed.DELETE("/webhooks/:id", server.deleteWebhook)
	authed.GET("/review/week", server.weeklyReview)
	authed.GET("/habits", server.habitStreaks)
	authed.GET("/unfurl", server.unfurl)
	authed.POST("/files", newRateLimiter(20, time.Minute), server.uploadFile)
	authed.GET("/files/:name", server.serveFile)
	authed.GET("/storage", server.storageUsage)
	authed.GET("/push/vapid", server.pushVapid)
	authed.POST("/push/subscribe", server.subscribePush)
	authed.POST("/push/unsubscribe", server.unsubscribePush)
	authed.GET("/caldav", server.listCaldav)
	authed.POST("/caldav", server.createCaldav)
	authed.POST("/caldav/test", server.testCaldav)
	authed.POST("/caldav/discover", server.discoverCaldav)
	authed.POST("/carddav", server.connectCarddav)
	authed.POST("/carddav/:id/sync", server.syncCarddav)
	authed.DELETE("/carddav/:id", server.deleteCarddav)
	authed.GET("/google/connect", server.googleConnect)
	authed.GET("/google/status", server.googleStatus)
	authed.POST("/google/sync", server.googleSyncNow)
	authed.DELETE("/google", server.googleDisconnect)
	authed.DELETE("/caldav/:id", server.deleteCaldav)
	authed.POST("/caldav/:id/sync", server.syncCaldav)

	router.GET("/api/widget/today", server.widgetToday)
	router.GET("/api/feed.ics", server.exportICS)
	router.POST("/api/mcp", newRateLimiter(60, time.Minute), server.mcp)
	router.POST("/api/intake", newRateLimiter(10, time.Minute), server.intake)
	router.GET("/api/google/callback", server.googleCallback)

	return router
}

func (s *Server) register(c *gin.Context) {
	var input authRequest
	if !bind(c, &input) || !validAuth(input) {
		c.String(http.StatusBadRequest, "invalid email or password")
		return
	}
	hash, err := auth.HashPassword(input.Password)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to create user")
		return
	}
	user, err := s.store.CreateUser(c.Request.Context(), input.Email, hash)
	if err != nil {
		c.String(http.StatusConflict, "account already exists")
		return
	}
	if !s.setSession(c, user.ID) {
		return
	}
	c.JSON(http.StatusOK, user)
}

func (s *Server) login(c *gin.Context) {
	var input authRequest
	if !bind(c, &input) || !validAuth(input) {
		c.String(http.StatusUnauthorized, "invalid credentials")
		return
	}
	user, hash, err := s.store.UserByEmail(c.Request.Context(), input.Email)
	if err != nil || !auth.VerifyPassword(input.Password, hash) {
		c.String(http.StatusUnauthorized, "invalid credentials")
		return
	}
	if !s.setSession(c, user.ID) {
		return
	}
	c.JSON(http.StatusOK, user)
}

func (s *Server) logout(c *gin.Context) {
	if cookie, err := c.Cookie("cal_session"); err == nil {
		_ = s.store.DeleteSession(c.Request.Context(), cookie)
	}
	s.writeSessionCookie(c, "", -1)
	c.Status(http.StatusNoContent)
}

func (s *Server) me(c *gin.Context) {
	c.JSON(http.StatusOK, currentUser(c))
}

func (s *Server) entries(c *gin.Context) {
	user := currentUser(c)
	entries, err := s.store.ListEntries(
		c.Request.Context(),
		user.ID,
		c.Query("from"),
		c.Query("to"),
		c.Query("q"),
	)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to list entries")
		return
	}
	c.JSON(http.StatusOK, entries)
}

func (s *Server) createEntry(c *gin.Context) {
	var input store.EntryInput
	if !bind(c, &input) || !validEntry(input) {
		c.String(http.StatusBadRequest, "invalid entry")
		return
	}
	// accountId must reference one of the caller's CalDAV accounts.
	if input.AccountID != nil && *input.AccountID != "" {
		_, owner, err := s.store.CaldavAccountWithSecret(c.Request.Context(), *input.AccountID)
		if err != nil || owner != currentUser(c).ID {
			c.String(http.StatusBadRequest, "unknown calendar account")
			return
		}
	} else {
		input.AccountID = nil
	}
	entry, err := s.store.CreateEntry(c.Request.Context(), currentUser(c).ID, input)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to create entry")
		return
	}
	go s.fireWebhooks(context.Background(), currentUser(c).ID, "entry.created", entry)
	c.JSON(http.StatusCreated, entry)
}

func (s *Server) updateEntry(c *gin.Context) {
	var raw map[string]json.RawMessage
	if !bind(c, &raw) {
		c.String(http.StatusBadRequest, "invalid entry")
		return
	}
	patch, ok := parsePatch(raw)
	if !ok {
		c.String(http.StatusBadRequest, "invalid entry")
		return
	}
	entry, err := s.store.UpdateEntry(c.Request.Context(), currentUser(c).ID, c.Param("id"), patch)
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "entry not found")
		return
	}
	if errors.Is(err, store.ErrInvalid) {
		c.String(http.StatusBadRequest, "invalid entry")
		return
	}
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to update entry")
		return
	}
	go s.fireWebhooks(context.Background(), currentUser(c).ID, "entry.updated", entry)
	c.JSON(http.StatusOK, entry)
}

func (s *Server) deleteEntry(c *gin.Context) {
	userID, id := currentUser(c).ID, c.Param("id")
	// A synced delete writes a tombstone so the next sync can DELETE remotely.
	if accountID, href, ok, err := s.store.EntryExternalRef(c.Request.Context(), userID, id); err == nil && ok {
		_ = s.store.Tombstone(c.Request.Context(), accountID, href)
	}
	err := s.store.DeleteEntry(c.Request.Context(), userID, id)
	if err == nil {
		go s.fireWebhooks(context.Background(), userID, "entry.deleted", gin.H{"id": id})
	}
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "entry not found")
		return
	}
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to delete entry")
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) entryRevisions(c *gin.Context) {
	revs, err := s.store.EntryRevisions(c.Request.Context(), currentUser(c).ID, c.Param("id"))
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, revs)
}

func (s *Server) restoreRevision(c *gin.Context) {
	err := s.store.RestoreRevision(c.Request.Context(), currentUser(c).ID, c.Param("id"), c.Param("rev"))
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "revision not found")
		return
	}
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) settings(c *gin.Context) {
	settings, err := s.store.Settings(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to load settings")
		return
	}
	c.JSON(http.StatusOK, settings)
}

func (s *Server) updateSettings(c *gin.Context) {
	var input store.Settings
	if !bind(c, &input) || !validSettings(input) {
		c.String(http.StatusBadRequest, "invalid settings")
		return
	}
	settings, err := s.store.UpdateSettings(c.Request.Context(), currentUser(c).ID, input)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to update settings")
		return
	}
	c.JSON(http.StatusOK, settings)
}

func (s *Server) export(c *gin.Context) {
	user := currentUser(c)
	entries, err := s.store.ListEntries(c.Request.Context(), user.ID, "", "", "")
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to export entries")
		return
	}
	settings, err := s.store.Settings(c.Request.Context(), user.ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to export settings")
		return
	}
	c.Header("Content-Disposition", `attachment; filename="cal-export.json"`)
	c.JSON(http.StatusOK, gin.H{
		"exportedAt": time.Now().UTC().Format(time.RFC3339),
		"user":       user,
		"settings":   settings,
		"entries":    entries,
	})
}

func (s *Server) holidays(c *gin.Context) {
	year, err := strconv.Atoi(c.Query("year"))
	if err != nil || year < 1970 || year > 2100 {
		c.String(http.StatusBadRequest, "invalid year")
		return
	}
	country := strings.ToUpper(strings.TrimSpace(c.Query("country")))
	if len(country) != 2 || !calendar.Supported(country) {
		c.String(http.StatusBadRequest, "unsupported country")
		return
	}
	c.JSON(http.StatusOK, s.holiday.For(country, year))
}

func (s *Server) holidayCountries(c *gin.Context) {
	c.JSON(http.StatusOK, calendar.Countries())
}

func (s *Server) requireUser(c *gin.Context) {
	sessionID, err := c.Cookie("cal_session")
	if err != nil || sessionID == "" {
		c.String(http.StatusUnauthorized, "authentication required")
		c.Abort()
		return
	}
	user, err := s.store.UserBySession(c.Request.Context(), sessionID)
	if err != nil {
		c.String(http.StatusUnauthorized, "authentication required")
		c.Abort()
		return
	}
	c.Set("user", user)
	c.Set("sessionID", sessionID)
	s.store.TouchSession(c.Request.Context(), sessionID)
	c.Next()
}

func (s *Server) setSession(c *gin.Context, userID string) bool {
	sessionID, err := s.store.CreateSession(c.Request.Context(), userID, c.Request.UserAgent())
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to create session")
		c.Abort()
		return false
	}
	s.writeSessionCookie(c, sessionID, int((30 * 24 * time.Hour).Seconds()))
	return true
}

func (s *Server) writeSessionCookie(c *gin.Context, value string, maxAge int) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     "cal_session",
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func currentUser(c *gin.Context) store.User {
	user, _ := c.Get("user")
	return user.(store.User)
}

func bind(c *gin.Context, target any) bool {
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target) == nil
}

func parsePatch(raw map[string]json.RawMessage) (store.EntryPatch, bool) {
	var patch store.EntryPatch
	for key, value := range raw {
		switch key {
		case "title":
			var v string
			if json.Unmarshal(value, &v) != nil || strings.TrimSpace(v) == "" {
				return patch, false
			}
			patch.Title = &v
		case "content":
			var v string
			if json.Unmarshal(value, &v) != nil {
				return patch, false
			}
			patch.Content = &v
		case "type":
			var v string
			if json.Unmarshal(value, &v) != nil || !validType(v) {
				return patch, false
			}
			patch.Type = &v
		case "linkUrl":
			var v string
			if json.Unmarshal(value, &v) != nil {
				return patch, false
			}
			patch.LinkURL = &v
		case "date":
			var v string
			if json.Unmarshal(value, &v) != nil || !validDate(v) {
				return patch, false
			}
			patch.Date = &v
		case "startTime":
			var v string
			if json.Unmarshal(value, &v) != nil || !validTime(v) {
				return patch, false
			}
			patch.StartTime = &v
		case "endTime":
			var v string
			if json.Unmarshal(value, &v) != nil || !validTime(v) {
				return patch, false
			}
			patch.EndTime = &v
		case "recur":
			var v string
			if json.Unmarshal(value, &v) != nil || !validRecur(v) {
				return patch, false
			}
			patch.Recur = &v
		case "completed":
			var v bool
			if json.Unmarshal(value, &v) != nil {
				return patch, false
			}
			patch.Completed = &v
		case "color":
			var v string
			if json.Unmarshal(value, &v) != nil {
				return patch, false
			}
			patch.Color = &v
		case "tags":
			var v []string
			if json.Unmarshal(value, &v) != nil {
				return patch, false
			}
			patch.Tags = v
			patch.HasTags = true
		default:
			return patch, false
		}
	}
	return patch, true
}

func validAuth(input authRequest) bool {
	return strings.Contains(input.Email, "@") && len(input.Email) <= 254 && len(input.Password) >= 8 && len(input.Password) <= 128
}

func validEntry(input store.EntryInput) bool {
	if strings.TrimSpace(input.Title) == "" || !validType(input.Type) || !validDate(input.Date) {
		return false
	}
	if !validTime(input.StartTime) || !validTime(input.EndTime) {
		return false
	}
	if input.StartTime == "" && input.EndTime != "" {
		return false
	}
	if input.StartTime != "" && input.EndTime != "" && input.EndTime <= input.StartTime {
		return false
	}
	if input.Recur != "" && (!validRecur(input.Recur) || input.Type != "task") {
		return false
	}
	return true
}

func validType(typ string) bool {
	return typ == "task" || typ == "note" || typ == "link" || typ == "event"
}

func validDate(date string) bool {
	_, err := time.Parse(time.DateOnly, date)
	return err == nil
}

// validTime accepts "" (unset) or an HH:MM 24h clock time.
func validTime(v string) bool {
	if v == "" {
		return true
	}
	_, err := time.Parse("15:04", v)
	return err == nil
}

func validRecur(v string) bool {
	switch v {
	case "none", "daily", "weekly", "monthly", "yearly":
		return true
	}
	return false
}

func validSettings(settings store.Settings) bool {
	theme := settings.Theme
	validTheme := theme == "light" || theme == "dark" || theme == "system"
	if settings.Timezone != "" {
		if _, err := time.LoadLocation(settings.Timezone); err != nil {
			return false
		}
	}
	return len(settings.Country) == 2 && validTheme &&
		(settings.WeekStart == "monday" || settings.WeekStart == "sunday")
}

func currentSessionID(c *gin.Context) string {
	if v, ok := c.Get("sessionID"); ok {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

func (s *Server) listSessions(c *gin.Context) {
	sessions, err := s.store.ListSessions(c.Request.Context(), currentUser(c).ID, currentSessionID(c))
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, sessions)
}

func (s *Server) revokeSession(c *gin.Context) {
	if err := s.store.RevokeSession(c.Request.Context(), currentUser(c).ID, c.Param("id")); err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) changePassword(c *gin.Context) {
	var body struct {
		Current  string `json:"current"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.Password) < 8 || len(body.Password) > 128 {
		c.String(http.StatusBadRequest, "password must be 8-128 characters")
		return
	}
	user, hash, err := s.store.UserByEmail(c.Request.Context(), currentUser(c).Email)
	if err != nil || !auth.VerifyPassword(body.Current, hash) {
		c.String(http.StatusForbidden, "current password is wrong")
		return
	}
	newHash, err := auth.HashPassword(body.Password)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	if err := s.store.ChangePassword(c.Request.Context(), user.ID, newHash); err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	// Sign out every other session — a password change invalidates them.
	_ = s.store.RevokeOtherSessions(c.Request.Context(), user.ID, currentSessionID(c))
	c.Status(http.StatusNoContent)
}

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Next()
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
