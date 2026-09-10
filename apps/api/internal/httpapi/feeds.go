package httpapi

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cal/apps/api/internal/ical"
	"cal/apps/api/internal/store"

	"github.com/gin-gonic/gin"
)

// feedEvent is the wire shape for external calendar events: read-only,
// keyed by feed so the client can color them.
type feedEvent struct {
	ID        string  `json:"id"`
	FeedID    string  `json:"feedId"`
	FeedName  string  `json:"feedName"`
	Title     string  `json:"title"`
	Date      string  `json:"date"`
	StartTime *string `json:"startTime,omitempty"`
	EndTime   *string `json:"endTime,omitempty"`
	Color     string  `json:"color"`
	Location  string  `json:"location,omitempty"`
	URL       string  `json:"url,omitempty"`
	Details   string  `json:"details,omitempty"`
}

// fetchICS downloads a feed body with a size cap and scheme check.
func fetchICS(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return "", errors.New("feed URL must be http(s)")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(u.String())
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", errors.New("feed returned " + resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (s *Server) listFeeds(c *gin.Context) {
	feeds, err := s.store.ListFeeds(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to list feeds")
		return
	}
	c.JSON(http.StatusOK, feeds)
}

func (s *Server) createFeed(c *gin.Context) {
	var body struct {
		Name  string `json:"name"`
		URL   string `json:"url"`
		Color string `json:"color"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.String(http.StatusBadRequest, "invalid json")
		return
	}
	body.URL = strings.TrimSpace(body.URL)
	if body.Name = strings.TrimSpace(body.Name); body.Name == "" {
		body.Name = "Calendar"
	}
	ics, err := fetchICS(body.URL)
	if err != nil {
		c.String(http.StatusBadGateway, "could not fetch feed: "+err.Error())
		return
	}
	if _, err := ical.Parse(ics); err != nil {
		c.String(http.StatusBadRequest, "not a valid iCalendar feed")
		return
	}
	if body.Color == "" {
		body.Color = "sky"
	}
	feed, err := s.store.CreateFeed(c.Request.Context(), currentUser(c).ID, body.Name, body.URL, body.Color, ics)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to save feed")
		return
	}
	c.JSON(http.StatusCreated, feed)
}

func (s *Server) deleteFeed(c *gin.Context) {
	if err := s.store.DeleteFeed(c.Request.Context(), currentUser(c).ID, c.Param("id")); errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusNotFound, "feed not found")
		return
	} else if err != nil {
		c.String(http.StatusInternalServerError, "failed to delete feed")
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) refreshFeed(c *gin.Context) {
	user := currentUser(c)
	feedID := c.Param("id")
	feeds, err := s.store.ListFeeds(c.Request.Context(), user.ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to load feeds")
		return
	}
	var target *store.Feed
	for i := range feeds {
		if feeds[i].ID == feedID {
			target = &feeds[i]
		}
	}
	if target == nil {
		c.String(http.StatusNotFound, "feed not found")
		return
	}
	ics, err := fetchICS(target.URL)
	if err != nil {
		c.String(http.StatusBadGateway, "could not fetch feed: "+err.Error())
		return
	}
	if err := s.store.RefreshFeedCache(c.Request.Context(), user.ID, feedID, ics); err != nil {
		c.String(http.StatusInternalServerError, "failed to refresh feed")
		return
	}
	c.Status(http.StatusNoContent)
}

// feedEvents expands every subscribed feed's cache into occurrences inside
// the requested date range.
func (s *Server) feedEvents(c *gin.Context) {
	from, to := c.Query("from"), c.Query("to")
	if from == "" || to == "" {
		c.String(http.StatusBadRequest, "from and to are required")
		return
	}
	fromT, err1 := time.Parse(time.DateOnly, from)
	toT, err2 := time.Parse(time.DateOnly, to)
	if err1 != nil || err2 != nil {
		c.String(http.StatusBadRequest, "invalid dates")
		return
	}
	caches, err := s.store.FeedCaches(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to load feeds")
		return
	}
	out := []feedEvent{}
	for feed, ics := range caches {
		events, err := ical.Parse(ics)
		if err != nil {
			continue
		}
		for _, e := range ical.Expand(events, fromT, toT.AddDate(0, 0, 1)) {
			fe := feedEvent{
				ID:       feed.ID + ":" + e.UID + ":" + e.Start.Format("20060102T150405"),
				FeedID:   feed.ID,
				FeedName: feed.Name,
				Title:    e.Summary,
				Date:     e.Start.Format("2006-01-02"),
				Color:    feed.Color,
				Location: e.Location,
				URL:      e.URL,
				Details:  e.Description,
			}
			if !e.AllDay {
				start := e.Start.Format("15:04")
				end := e.End.Format("15:04")
				fe.StartTime = &start
				fe.EndTime = &end
			}
			out = append(out, fe)
		}
	}
	c.JSON(http.StatusOK, out)
}

// importICS accepts a raw .ics body and converts each VEVENT into an entry.
// Imported events keep their date/times and become non-completable "event"
// entries; recurring rules flatten to their occurrences within ±1 year.
func (s *Server) importICS(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 4<<20))
	if err != nil {
		c.String(http.StatusBadRequest, "empty body")
		return
	}
	events, err := ical.Parse(string(body))
	if err != nil || len(events) == 0 {
		c.String(http.StatusBadRequest, "no events found in file")
		return
	}
	now := time.Now()
	flat := ical.Expand(events, now.AddDate(-1, 0, 0), now.AddDate(1, 0, 0))
	user := currentUser(c)
	created := 0
	for _, e := range flat {
		input := store.EntryInput{
			Title:   e.Summary,
			Type:    "event",
			Content: strings.TrimSpace(e.Description + "\n" + e.Location),
			Date:    e.Start.Format("2006-01-02"),
		}
		if !e.AllDay {
			input.StartTime = e.Start.Format("15:04")
			input.EndTime = e.End.Format("15:04")
		}
		if input.Title = strings.TrimSpace(input.Title); input.Title == "" {
			input.Title = "Untitled event"
		}
		if _, err := s.store.CreateEntry(c.Request.Context(), user.ID, input); err != nil {
			log.Printf("import: %v", err)
			continue
		}
		created++
	}
	c.JSON(http.StatusOK, gin.H{"imported": created})
}

// widgetToday is the token-gated read used by the embeddable Today widget.
func (s *Server) widgetToday(c *gin.Context) {
	user, err := s.store.UserByWidgetToken(c.Request.Context(), c.Query("token"))
	if errors.Is(err, store.ErrNotFound) {
		c.String(http.StatusUnauthorized, "invalid widget token")
		return
	} else if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	today := time.Now().Format(time.DateOnly)
	entries, err := s.store.ListEntries(c.Request.Context(), user.ID, today, today, "")
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, entries)
}

func (s *Server) rotateWidgetToken(c *gin.Context) {
	token, err := s.store.RegenerateWidgetToken(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, gin.H{"widgetToken": token})
}

func (s *Server) rotateApiToken(c *gin.Context) {
	token, err := s.store.RegenerateApiToken(c.Request.Context(), currentUser(c).ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	c.JSON(http.StatusOK, gin.H{"apiToken": token})
}
