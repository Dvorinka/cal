package calendar

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type Holiday struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Date    string `json:"date"`
	Country string `json:"country"`
}

// HolidayCache memoizes generated holiday lists per country/year.
type HolidayCache struct {
	mu    sync.RWMutex
	items map[string][]Holiday
}

func NewHolidayCache() *HolidayCache {
	return &HolidayCache{items: map[string][]Holiday{}}
}

// Supported reports whether a country code has holiday definitions.
func Supported(country string) bool {
	_, ok := countries[country]
	return ok
}

// Countries returns the supported country list, sorted by name.
func Countries() []Country {
	out := make([]Country, 0, len(countries))
	for code, c := range countries {
		out = append(out, Country{Code: code, Name: c.Name})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (c *HolidayCache) For(country string, year int) []Holiday {
	key := fmt.Sprintf("%s:%d", country, year)
	c.mu.RLock()
	if holidays, ok := c.items[key]; ok {
		c.mu.RUnlock()
		return holidays
	}
	c.mu.RUnlock()

	holidays := buildHolidays(country, year)
	c.mu.Lock()
	c.items[key] = holidays
	c.mu.Unlock()
	return holidays
}

func buildHolidays(country string, year int) []Holiday {
	defs, ok := countries[country]
	if !ok {
		return []Holiday{}
	}
	seen := map[string]bool{}
	out := make([]Holiday, 0, len(defs.Holidays))
	for _, h := range defs.Holidays {
		date := h.Rule(year).Format(time.DateOnly)
		if seen[date+h.Name] {
			continue
		}
		seen[date+h.Name] = true
		out = append(out, Holiday{
			ID:      fmt.Sprintf("%s-%s-%d", country, date, len(out)),
			Name:    h.Name,
			Date:    date,
			Country: country,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return out
}
