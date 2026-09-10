package calendar

import (
	"testing"
	"time"
)

func dates(country string, year int) map[string]string {
	out := map[string]string{}
	for _, h := range NewHolidayCache().For(country, year) {
		out[h.Name] = h.Date
	}
	return out
}

func TestHolidayCacheReturnsStableCountryYear(t *testing.T) {
	cache := NewHolidayCache()
	holidays := cache.For("US", 2026)
	if len(holidays) == 0 {
		t.Fatal("expected seeded holidays")
	}
	if holidays[0].Country != "US" {
		t.Fatalf("country = %q, want US", holidays[0].Country)
	}
	if holidays[0].Date[:4] != "2026" {
		t.Fatalf("date = %q, want year 2026", holidays[0].Date)
	}
}

func TestKnownDates(t *testing.T) {
	cases := []struct {
		country string
		year    int
		name    string
		want    string
	}{
		{"US", 2026, "Thanksgiving", "2026-11-26"},
		{"US", 2026, "Martin Luther King Jr. Day", "2026-01-19"},
		{"US", 2025, "Memorial Day", "2025-05-26"},
		{"CA", 2026, "Victoria Day", "2026-05-18"},
		{"GB", 2026, "Spring Bank Holiday", "2026-05-25"},
		{"CZ", 2026, "Liberation Day", "2026-05-08"},
		{"DE", 2026, "Ascension Day", "2026-05-14"},
		{"PL", 2026, "Whit Sunday", "2026-05-24"},
		{"FR", 2026, "Bastille Day", "2026-07-14"},
		{"JP", 2026, "Coming of Age Day", "2026-01-12"},
		{"NZ", 2026, "Waitangi Day", "2026-02-06"},
	}
	for _, tc := range cases {
		got, ok := dates(tc.country, tc.year)[tc.name]
		if !ok {
			t.Fatalf("%s %d: holiday %q missing", tc.country, tc.year, tc.name)
		}
		if got != tc.want {
			t.Fatalf("%s %d %s = %s, want %s", tc.country, tc.year, tc.name, got, tc.want)
		}
	}
}

func TestEasterDates(t *testing.T) {
	// Known Easters: Western 2026-04-05, 2027-03-28; Orthodox 2026-04-12.
	if got := gregorianEaster(2026).Format(time.DateOnly); got != "2026-04-05" {
		t.Fatalf("gregorian easter 2026 = %s", got)
	}
	if got := gregorianEaster(2027).Format(time.DateOnly); got != "2027-03-28" {
		t.Fatalf("gregorian easter 2027 = %s", got)
	}
	if got := orthodoxEaster(2026).Format(time.DateOnly); got != "2026-04-12" {
		t.Fatalf("orthodox easter 2026 = %s", got)
	}
}

func TestUnsupportedCountryReturnsEmpty(t *testing.T) {
	if got := NewHolidayCache().For("XX", 2026); len(got) != 0 {
		t.Fatalf("expected no holidays, got %d", len(got))
	}
	if Supported("XX") {
		t.Fatal("XX should not be supported")
	}
}

func TestCountriesSortedAndPresent(t *testing.T) {
	list := Countries()
	if len(list) < 30 {
		t.Fatalf("expected at least 30 countries, got %d", len(list))
	}
	for i := 1; i < len(list); i++ {
		if list[i-1].Name > list[i].Name {
			t.Fatalf("countries not sorted at %d", i)
		}
	}
}

func TestAllRulesProduceValidDates(t *testing.T) {
	for code := range countries {
		for _, h := range NewHolidayCache().For(code, 2026) {
			if _, err := time.Parse(time.DateOnly, h.Date); err != nil {
				t.Fatalf("%s produced invalid date %q", code, h.Date)
			}
		}
	}
}
