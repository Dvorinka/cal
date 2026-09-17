package httpapi

import (
	"testing"
	"time"

	"cal/apps/api/internal/store"
)

func TestUpcomingPersonDates(t *testing.T) {
	now := time.Date(2026, 9, 17, 10, 0, 0, 0, time.UTC)
	bday := "1990-09-20"
	leap := "2000-02-29"
	old := "1900-03-01" // yearless (carddav import) — no "turns" shown
	people := []store.Person{
		{ID: "1", Name: "Ana", Birthday: &bday},
		{ID: "2", Name: "Ben", Birthday: &leap},
		{
			ID: "3", Name: "Cid",
			Dates: []store.PersonDate{
				{Label: "nameday", Date: "2026-09-17"},
				{Label: "met", Date: old},
				{Label: "far", Date: "2027-08-01"},
			},
		},
		{ID: "4", Name: "Dee"}, // no dates at all
	}
	got := upcomingPersonDates(people, 60, now)
	// Expected: Cid/nameday today, Ana/birthday in 3d, Cid/met in ~165d? no —
	// 1900-03-01 next occurrence is 2027-03-01 (165 days) → outside 60d.
	// Ben 2027-02-28 → outside 60d. "far" → outside.
	if len(got) != 2 {
		t.Fatalf("expected 2 occurrences, got %+v", got)
	}
	if got[0].Label != "nameday" || got[0].DaysUntil != 0 {
		t.Fatalf("first = %+v, want nameday today", got[0])
	}
	if got[1].Name != "Ana" || got[1].DaysUntil != 3 {
		t.Fatalf("second = %+v, want Ana birthday in 3d", got[1])
	}
	if got[1].Turns == nil || *got[1].Turns != 36 {
		t.Fatalf("Ana turns = %v, want 36", got[1].Turns)
	}
}

func TestUpcomingPersonDatesLeap(t *testing.T) {
	// Feb 29 birthdays fire on Feb 28 in a non-leap year.
	now := time.Date(2027, 2, 20, 0, 0, 0, 0, time.UTC)
	leap := "2000-02-29"
	got := upcomingPersonDates([]store.Person{{ID: "1", Name: "Ben", Birthday: &leap}}, 60, now)
	if len(got) != 1 {
		t.Fatalf("expected 1 occurrence, got %+v", got)
	}
	if got[0].Date != "2027-02-28" || got[0].DaysUntil != 8 {
		t.Fatalf("got %+v, want 2027-02-28 in 8d", got[0])
	}
}
