// Package holiday fetches public holidays from date.nager.at (v4) for
// countries the offline rule engine doesn't cover. The embedded rules in
// internal/calendar stay primary; this is a browse-and-import source.
package holiday

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// NagerHoliday is a single public holiday entry as returned by date.nager.at.
type NagerHoliday struct {
	Date             string   `json:"date"`
	Name             string   `json:"name"`
	CountryCode      string   `json:"countryCode"`
	NationalHoliday  bool     `json:"nationalHoliday"`
	SubdivisionCodes []string `json:"subdivisionCodes"`
	HolidayTypes     []string `json:"holidayTypes"`
}

// NagerCountry is one entry of the AvailableCountries list.
type NagerCountry struct {
	Code string `json:"countryCode"`
	Name string `json:"name"`
}

const defaultBaseURL = "https://date.nager.at/api/v4"

// Client fetches holidays from date.nager.at with a 7-day in-memory cache.
type Client struct {
	baseURL string
	hc      *http.Client

	mu    sync.Mutex
	lists map[string]cachedList
}

type cachedList struct {
	items   []NagerHoliday
	expires time.Time
}

// NewClient builds a holiday Client with a 10s timeout.
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		hc:      &http.Client{Timeout: 10 * time.Second},
		lists:   map[string]cachedList{},
	}
}

// List returns the public holidays for the given ISO country code and year.
func (c *Client) List(ctx context.Context, countryCode string, year int) ([]NagerHoliday, error) {
	code := strings.ToUpper(strings.TrimSpace(countryCode))
	if code == "" {
		return nil, fmt.Errorf("holiday: empty country code")
	}
	if year < 1900 || year > 2200 {
		return nil, fmt.Errorf("holiday: invalid year %d", year)
	}
	key := fmt.Sprintf("%s:%d", code, year)
	c.mu.Lock()
	if cl, ok := c.lists[key]; ok && time.Now().Before(cl.expires) {
		c.mu.Unlock()
		return cl.items, nil
	}
	c.mu.Unlock()

	endpoint, err := url.JoinPath(c.baseURL, "Holidays", code, fmt.Sprintf("%d", year))
	if err != nil {
		return nil, fmt.Errorf("holiday url: %w", err)
	}
	var out []NagerHoliday
	if err := c.getJSON(ctx, endpoint, &out); err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.lists[key] = cachedList{items: out, expires: time.Now().Add(7 * 24 * time.Hour)}
	c.mu.Unlock()
	return out, nil
}

// Countries returns every country the API covers (~150).
func (c *Client) Countries(ctx context.Context) ([]NagerCountry, error) {
	endpoint, err := url.JoinPath(c.baseURL, "AvailableCountries")
	if err != nil {
		return nil, fmt.Errorf("holiday url: %w", err)
	}
	var out []NagerCountry
	if err := c.getJSON(ctx, endpoint, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) getJSON(ctx context.Context, endpoint string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	res, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("holiday request: %w", err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("holiday read: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("holiday: HTTP %d: %s", res.StatusCode, truncBody(string(raw)))
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("holiday decode: %w", err)
	}
	return nil
}

func truncBody(s string) string {
	if len(s) <= 200 {
		return s
	}
	return s[:200] + "..."
}
