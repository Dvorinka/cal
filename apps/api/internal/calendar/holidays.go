package calendar

import (
	"fmt"
	"sync"
	"time"
)

type Holiday struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Date    string `json:"date"`
	Country string `json:"country"`
}

type HolidayCache struct {
	mu    sync.RWMutex
	items map[string][]Holiday
}

func NewHolidayCache() *HolidayCache {
	return &HolidayCache{items: map[string][]Holiday{}}
}

func (c *HolidayCache) For(country string, year int) []Holiday {
	key := fmt.Sprintf("%s:%d", country, year)
	c.mu.RLock()
	if holidays, ok := c.items[key]; ok {
		c.mu.RUnlock()
		return holidays
	}
	c.mu.RUnlock()

	holidays := seedHolidays(country, year)
	c.mu.Lock()
	c.items[key] = holidays
	c.mu.Unlock()
	return holidays
}

func seedHolidays(country string, year int) []Holiday {
	common := []struct {
		Name  string
		Month time.Month
		Day   int
	}{
		{"New Year's Day", time.January, 1},
		{"Christmas Day", time.December, 25},
	}
	if country == "US" {
		common = append(common, struct {
			Name  string
			Month time.Month
			Day   int
		}{"Independence Day", time.July, 4})
	}
	if country == "CZ" {
		common = append(common, struct {
			Name  string
			Month time.Month
			Day   int
		}{"Czech Statehood Day", time.September, 28})
	}

	out := make([]Holiday, 0, len(common))
	for _, h := range common {
		date := time.Date(year, h.Month, h.Day, 0, 0, 0, 0, time.UTC).Format(time.DateOnly)
		out = append(out, Holiday{
			ID:      fmt.Sprintf("%s-%s", country, date),
			Name:    h.Name,
			Date:    date,
			Country: country,
		})
	}
	return out
}
