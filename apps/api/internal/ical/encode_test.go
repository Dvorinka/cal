package ical

import (
	"strings"
	"testing"
	"time"
)

func TestEncodeEventRoundTrip(t *testing.T) {
	ev := Event{
		UID:         "test-1",
		Summary:     "Dinner, maybe?",
		Description: "line one\nline two; with semicolon",
		Start:       time.Date(2026, 9, 12, 18, 30, 0, 0, time.UTC),
		End:         time.Date(2026, 9, 12, 20, 0, 0, 0, time.UTC),
	}
	out := EncodeEvent(ev)
	if !strings.Contains(out, "SUMMARY:Dinner\\, maybe?") {
		t.Errorf("summary not escaped:\n%s", out)
	}
	if !strings.Contains(out, "DTSTART:20260912T183000Z") {
		t.Errorf("dtstart:\n%s", out)
	}
	parsed, err := Parse(out)
	if err != nil || len(parsed) != 1 {
		t.Fatalf("round-trip parse failed: %v", err)
	}
	if parsed[0].UID != "test-1" || parsed[0].Summary != "Dinner, maybe?" {
		t.Errorf("round-trip mismatch: %+v", parsed[0])
	}
}

func TestEncodeAllDay(t *testing.T) {
	ev := Event{UID: "d1", Summary: "Holiday", AllDay: true, Start: time.Date(2026, 12, 25, 0, 0, 0, 0, time.UTC)}
	out := EncodeEvent(ev)
	if !strings.Contains(out, "DTSTART;VALUE=DATE:20261225") {
		t.Errorf("all-day dtstart:\n%s", out)
	}
	if !strings.Contains(out, "DTEND;VALUE=DATE:20261226") {
		t.Errorf("all-day dtend should default to +1d:\n%s", out)
	}
}
