package httpapi

import (
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
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func New(st *store.Store, holidays *calendar.HolidayCache) *gin.Engine {
	secure := os.Getenv("SESSION_SECURE") == "true"
	server := &Server{store: st, holiday: holidays, secure: secure}

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
	authed.GET("/settings", server.settings)
	authed.PUT("/settings", server.updateSettings)

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
	entry, err := s.store.CreateEntry(c.Request.Context(), currentUser(c).ID, input)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to create entry")
		return
	}
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
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to update entry")
		return
	}
	c.JSON(http.StatusOK, entry)
}

func (s *Server) deleteEntry(c *gin.Context) {
	err := s.store.DeleteEntry(c.Request.Context(), currentUser(c).ID, c.Param("id"))
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
	c.Next()
}

func (s *Server) setSession(c *gin.Context, userID string) bool {
	sessionID, err := s.store.CreateSession(c.Request.Context(), userID)
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
	return typ == "task" || typ == "note" || typ == "link"
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
	return len(settings.Country) == 2 && validTheme &&
		(settings.WeekStart == "monday" || settings.WeekStart == "sunday")
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
