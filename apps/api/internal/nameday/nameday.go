// Package nameday serves country-aware nameday lookups: the abalin.net V2
// API as primary source and embedded per-country CSV calendars (CZ, SK, PL,
// HU, AT, DE) as offline fallback, behind a 24-hour in-memory cache.
package nameday

import (
	"context"
	"embed"
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

//go:embed data/*.csv
var csvFS embed.FS

// Entry is a single day's nameday list for a country.
type Entry struct {
	Month int
	Day   int
	Names []string
}

// Calendar is the full year of namedays for one country, keyed by month/day.
type Calendar struct {
	Country string
	Entries map[[2]int]Entry
}

// Loader holds nameday calendars for all embedded countries and doubles as
// the offline fallback Provider.
type Loader struct {
	calendars map[string]*Calendar
}

// NewLoader loads every embedded *.csv file. Filenames (without extension)
// are lowercased country codes.
func NewLoader() (*Loader, error) {
	l := &Loader{calendars: make(map[string]*Calendar)}
	err := fs.WalkDir(csvFS, "data", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".csv") {
			return nil
		}
		base := filepath.Base(path)
		code := strings.ToLower(strings.TrimSuffix(base, ".csv"))
		f, err := csvFS.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		cal, err := parseCSV(f, code)
		if err != nil {
			return fmt.Errorf("load %s: %w", base, err)
		}
		l.calendars[code] = cal
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(l.calendars) == 0 {
		return nil, fmt.Errorf("no embedded nameday calendars")
	}
	return l, nil
}

func parseCSV(r io.Reader, code string) (*Calendar, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	rows, err := cr.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("empty calendar")
	}
	// Skip header row.
	cal := &Calendar{Country: code, Entries: make(map[[2]int]Entry, 366)}
	for i, row := range rows[1:] {
		if len(row) < 3 {
			return nil, fmt.Errorf("row %d: expected 3 fields, got %d", i+2, len(row))
		}
		month, err := strconv.Atoi(strings.TrimSpace(row[0]))
		if err != nil {
			return nil, fmt.Errorf("row %d: invalid month %q", i+2, row[0])
		}
		day, err := strconv.Atoi(strings.TrimSpace(row[1]))
		if err != nil {
			return nil, fmt.Errorf("row %d: invalid day %q", i+2, row[1])
		}
		if month < 1 || month > 12 || day < 1 || day > 31 {
			return nil, fmt.Errorf("row %d: out-of-range date %d/%d", i+2, month, day)
		}
		names := splitNames(row[2])
		if len(names) == 0 {
			return nil, fmt.Errorf("row %d: empty names", i+2)
		}
		cal.Entries[[2]int{month, day}] = Entry{Month: month, Day: day, Names: names}
	}
	return cal, nil
}

func splitNames(s string) []string {
	parts := strings.Split(s, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// Countries returns the sorted list of loaded country codes.
func (l *Loader) Countries() []string {
	codes := make([]string, 0, len(l.calendars))
	for c := range l.calendars {
		codes = append(codes, c)
	}
	sort.Strings(codes)
	return codes
}

// Lookup returns the entry for the given country and date.
func (l *Loader) Lookup(country string, month, day int) (Entry, bool) {
	cal, ok := l.calendars[strings.ToLower(country)]
	if !ok {
		return Entry{}, false
	}
	e, ok := cal.Entries[[2]int{month, day}]
	return e, ok
}

// GetByDate returns namedays for the given month/day across all loaded CSV
// calendars, comma-joined per country (matching the abalin API shape).
func (l *Loader) GetByDate(ctx context.Context, month, day int) (map[string]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if month < 1 || month > 12 || day < 1 || day > 31 {
		return nil, fmt.Errorf("csv: invalid date %d/%d", month, day)
	}
	out := make(map[string]string, len(l.calendars))
	for code, cal := range l.calendars {
		if e, ok := cal.Entries[[2]int{month, day}]; ok {
			out[code] = strings.Join(e.Names, ", ")
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("csv: no namedays for %d/%d", month, day)
	}
	return out, nil
}

// SearchByName searches every loaded CSV calendar for entries whose names
// contain the given substring (case-insensitive).
func (l *Loader) SearchByName(ctx context.Context, name string) ([]CountryResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	needle := strings.ToLower(strings.TrimSpace(name))
	if needle == "" {
		return nil, fmt.Errorf("csv: empty name")
	}
	out := make([]CountryResult, 0, len(l.calendars))
	for code, cal := range l.calendars {
		var dates []NameDate
		for _, e := range cal.Entries {
			for _, n := range e.Names {
				if strings.Contains(strings.ToLower(n), needle) {
					dates = append(dates, NameDate{Day: e.Day, Month: e.Month, Name: n})
				}
			}
		}
		if len(dates) > 0 {
			sort.Slice(dates, func(i, j int) bool {
				if dates[i].Month != dates[j].Month {
					return dates[i].Month < dates[j].Month
				}
				return dates[i].Day < dates[j].Day
			})
			out = append(out, CountryResult{Country: code, Dates: dates})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Country < out[j].Country })
	return out, nil
}

// SupportedCountries returns the sorted list of loaded CSV country codes.
func (l *Loader) SupportedCountries() []string {
	return l.Countries()
}
