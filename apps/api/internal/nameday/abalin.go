package nameday

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// abalinCountries is the full list of country codes supported by the
// nameday.abalin.net V2 API. The API returns all of these in every
// date response.
var abalinCountries = []string{
	"at", "bg", "cz", "de", "dk", "ee", "es", "fi", "fr", "gr",
	"hr", "hu", "it", "lt", "lv", "pl", "se", "sk", "us",
}

// DefaultAbalinBaseURL is the public abalin API root.
const DefaultAbalinBaseURL = "https://nameday.abalin.net/api/V2"

// AbalinClient is a Provider backed by the nameday.abalin.net HTTP API.
type AbalinClient struct {
	baseURL string
	hc      *http.Client
}

// NewAbalinClient builds an AbalinClient with a 10s timeout.
func NewAbalinClient(baseURL string) *AbalinClient {
	if baseURL == "" {
		baseURL = DefaultAbalinBaseURL
	}
	return &AbalinClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		hc:      &http.Client{Timeout: 10 * time.Second},
	}
}

// abalinDateResponse is the shape of the /date response.
type abalinDateResponse struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Data    map[string]string `json:"data"`
}

// GetByDate fetches namedays for the given month/day across all countries.
func (a *AbalinClient) GetByDate(ctx context.Context, month, day int) (map[string]string, error) {
	if month < 1 || month > 12 || day < 1 || day > 31 {
		return nil, fmt.Errorf("abalin: invalid date %d/%d", month, day)
	}
	endpoint, err := url.JoinPath(a.baseURL, "date")
	if err != nil {
		return nil, fmt.Errorf("abalin date url: %w", err)
	}
	q := url.Values{}
	q.Set("day", strconv.Itoa(day))
	q.Set("month", strconv.Itoa(month))
	var resp abalinDateResponse
	if err := a.getJSON(ctx, endpoint, q, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("abalin date: %s", resp.Message)
	}
	return resp.Data, nil
}

// SearchByName searches for a name across all countries via /getname.
func (a *AbalinClient) SearchByName(ctx context.Context, name string) ([]CountryResult, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("abalin: empty name")
	}
	endpoint, err := url.JoinPath(a.baseURL, "getname")
	if err != nil {
		return nil, fmt.Errorf("abalin getname url: %w", err)
	}

	body, err := json.Marshal(map[string]string{"name": name})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	res, err := a.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("abalin getname request: %w", err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("abalin getname read: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("abalin getname: HTTP %d: %s", res.StatusCode, truncate(string(raw), 200))
	}

	// The getname data array elements have a "country" field plus numeric
	// keys mapping to {day, month, name}. Decode generically.
	var top struct {
		Success bool             `json:"success"`
		Data    []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, fmt.Errorf("abalin getname decode: %w", err)
	}
	if !top.Success {
		return nil, fmt.Errorf("abalin getname: unsuccessful response")
	}

	out := make([]CountryResult, 0, len(top.Data))
	for _, item := range top.Data {
		country, _ := item["country"].(string)
		if country == "" {
			continue
		}
		var dates []NameDate
		// Numeric keys are the date entries; "country" is metadata.
		for k, v := range item {
			if k == "country" {
				continue
			}
			// Only numeric keys (e.g. "0", "1") are date entries.
			if _, err := strconv.Atoi(k); err != nil {
				continue
			}
			m, ok := v.(map[string]any)
			if !ok {
				continue
			}
			nd := NameDate{}
			if fv, ok := m["day"].(float64); ok {
				nd.Day = int(fv)
			}
			if fv, ok := m["month"].(float64); ok {
				nd.Month = int(fv)
			}
			if s, ok := m["name"].(string); ok {
				nd.Name = s
			}
			if nd.Day == 0 && nd.Month == 0 && nd.Name == "" {
				continue
			}
			dates = append(dates, nd)
		}
		out = append(out, CountryResult{Country: strings.ToLower(country), Dates: dates})
	}
	return out, nil
}

// SupportedCountries returns the abalin API's supported country codes.
func (a *AbalinClient) SupportedCountries() []string {
	out := make([]string, len(abalinCountries))
	copy(out, abalinCountries)
	return out
}

func (a *AbalinClient) getJSON(ctx context.Context, endpoint string, q url.Values, dst any) error {
	u := endpoint
	if len(q) > 0 {
		u = u + "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	res, err := a.hc.Do(req)
	if err != nil {
		return fmt.Errorf("abalin request: %w", err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("abalin read: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("abalin: HTTP %d: %s", res.StatusCode, truncate(string(raw), 200))
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("abalin decode: %w", err)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
