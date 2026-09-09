package calendar

import "testing"

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
