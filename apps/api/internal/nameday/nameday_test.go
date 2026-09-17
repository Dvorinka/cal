package nameday

import (
	"context"
	"testing"
)

func TestEmbeddedCalendarsLoad(t *testing.T) {
	l, err := NewLoader()
	if err != nil {
		t.Fatalf("NewLoader: %v", err)
	}
	want := []string{"at", "cz", "de", "hu", "pl", "sk"}
	got := l.Countries()
	if len(got) != len(want) {
		t.Fatalf("countries = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("countries = %v, want %v", got, want)
		}
	}
	// Every calendar should cover the full year.
	for _, code := range want {
		for month := 1; month <= 12; month++ {
			// Spot-check a day in each month exists.
			if _, ok := l.Lookup(code, month, 15); !ok {
				t.Fatalf("%s: no entry for %d/15", code, month)
			}
		}
	}
}

func TestCSVSearchByName(t *testing.T) {
	l, _ := NewLoader()
	res, err := l.SearchByName(context.Background(), "jan")
	if err != nil {
		t.Fatalf("SearchByName: %v", err)
	}
	if len(res) == 0 {
		t.Fatal("expected matches for 'jan'")
	}
	for _, r := range res {
		if len(r.Dates) == 0 {
			t.Fatalf("%s: empty dates", r.Country)
		}
	}
}

func TestCSVGetByDate(t *testing.T) {
	l, _ := NewLoader()
	m, err := l.GetByDate(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("GetByDate: %v", err)
	}
	if m["cz"] == "" {
		t.Fatal("expected a CZ nameday on Jan 1")
	}
	if _, err := l.GetByDate(context.Background(), 13, 1); err == nil {
		t.Fatal("expected invalid-month error")
	}
}
