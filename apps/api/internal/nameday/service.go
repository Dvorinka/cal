package nameday

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"
)

const cacheTTL = 24 * time.Hour

// Service orchestrates nameday lookups: try the primary provider (abalin
// API), fall back to the embedded CSV loader on error, and cache primary
// results in memory for 24h. Fallback results are not cached — they may be
// incomplete vs. the API.
type Service struct {
	primary  Provider
	fallback Provider

	mu    sync.Mutex
	cache map[string]cachedDate
}

type cachedDate struct {
	names   map[string]string
	expires time.Time
}

// NewService builds a nameday Service. fallback may be nil to disable the
// offline path (not recommended — the embed is free).
func NewService(primary, fallback Provider) *Service {
	return &Service{primary: primary, fallback: fallback, cache: map[string]cachedDate{}}
}

// NewDefault builds the standard service: abalin API primary, embedded CSV
// fallback. Panics only if the embedded data is corrupt — a build-time bug.
func NewDefault() *Service {
	loader, err := NewLoader()
	if err != nil {
		panic(fmt.Sprintf("nameday embedded data: %v", err))
	}
	return NewService(NewAbalinClient(""), loader)
}

// GetByDate returns namedays for the given month/day across all countries.
func (s *Service) GetByDate(ctx context.Context, month, day int) (map[string]string, error) {
	key := fmt.Sprintf("%d:%d", month, day)
	s.mu.Lock()
	if c, ok := s.cache[key]; ok && time.Now().Before(c.expires) {
		s.mu.Unlock()
		return c.names, nil
	}
	s.mu.Unlock()

	res, err := s.primary.GetByDate(ctx, month, day)
	if err == nil {
		s.mu.Lock()
		s.cache[key] = cachedDate{names: res, expires: time.Now().Add(cacheTTL)}
		s.mu.Unlock()
		return res, nil
	}
	log.Printf("nameday: primary failed for %d/%d, falling back to CSV: %v", month, day, err)

	if s.fallback == nil {
		return nil, err
	}
	res, fbErr := s.fallback.GetByDate(ctx, month, day)
	if fbErr != nil {
		return nil, fmt.Errorf("nameday: primary: %v; fallback: %v", err, fbErr)
	}
	return res, nil
}

// SearchByName proxies to the primary provider; on failure it falls back to
// the embedded CSV calendars (fewer countries, no network).
func (s *Service) SearchByName(ctx context.Context, name string) ([]CountryResult, error) {
	res, err := s.primary.SearchByName(ctx, name)
	if err == nil {
		return res, nil
	}
	log.Printf("nameday: search primary failed for %q, falling back to CSV: %v", name, err)
	if s.fallback == nil {
		return nil, err
	}
	res, fbErr := s.fallback.SearchByName(ctx, name)
	if fbErr != nil {
		return nil, fmt.Errorf("nameday search: primary: %v; fallback: %v", err, fbErr)
	}
	return res, nil
}

// SupportedCountries returns the union of primary and fallback country
// codes, sorted and de-duplicated.
func (s *Service) SupportedCountries() []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range []Provider{s.primary, s.fallback} {
		if p == nil {
			continue
		}
		for _, c := range p.SupportedCountries() {
			if !seen[c] {
				seen[c] = true
				out = append(out, c)
			}
		}
	}
	sort.Strings(out)
	return out
}
