package ical

import (
	"testing"
	"time"
)

const sample = `BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//Test//EN
BEGIN:VEVENT
UID:one@test
SUMMARY:All-hands
DTSTART:20260914T150000Z
DTEND:20260914T160000Z
END:VEVENT
BEGIN:VEVENT
UID:two@test
SUMMARY:Long\nfolded
 line
DTSTART;VALUE=DATE:20260920
DTEND;VALUE=DATE:20260922
END:VEVENT
BEGIN:VEVENT
UID:three@test
SUMMARY:Standup
DTSTART;TZID=Europe/Prague:20260914T093000
RRULE:FREQ=DAILY;COUNT=5
END:VEVENT
BEGIN:VEVENT
UID:four@test
SUMMARY:Weekly review
DTSTART:20260911T160000Z
RRULE:FREQ=WEEKLY;BYDAY=FR;UNTIL=20261031T000000Z
END:VEVENT
BEGIN:VEVENT
UID:five@test
SUMMARY:Cancelled thing
DTSTART:20260915T100000Z
STATUS:CANCELLED
END:VEVENT
END:VCALENDAR`

func TestParse(t *testing.T) {
	events, err := Parse(sample)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
	}
	if events[0].Summary != "All-hands" {
		t.Errorf("summary: %q", events[0].Summary)
	}
	if events[0].Start.UTC().Hour() != 15 {
		t.Errorf("start: %v", events[0].Start)
	}
	// Folded line + \n unescape: fold point joins "folded" + "line".
	if events[1].Summary != "Long\nfoldedline" {
		t.Errorf("folded summary: %q", events[1].Summary)
	}
	if !events[1].AllDay || events[1].End.Sub(events[1].Start) != 48*time.Hour {
		t.Errorf("all-day event: %v %v", events[1].Start, events[1].End)
	}
	if events[2].Recurrence == nil || events[2].Recurrence.Freq != "DAILY" || events[2].Recurrence.Count != 5 {
		t.Errorf("rrule: %+v", events[2].Recurrence)
	}
	// TZID parsing — Prague is UTC+2 in September.
	if events[2].Start.UTC().Hour() != 7 || events[2].Start.UTC().Minute() != 30 {
		t.Errorf("tzid start: %v", events[2].Start)
	}
}

func TestExpand(t *testing.T) {
	events, _ := Parse(sample)
	from := time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	out := Expand(events, from, to)

	var standups, reviews, cancelled, allhands, allday int
	for _, e := range out {
		switch e.UID {
		case "three@test":
			standups++
		case "four@test":
			reviews++
		case "five@test":
			cancelled++
		case "one@test":
			allhands++
		case "two@test":
			allday++
		}
	}
	if standups != 5 {
		t.Errorf("expected 5 standup occurrences, got %d", standups)
	}
	if reviews != 1 { // Sep 18 falls in range
		t.Errorf("expected 1 weekly review, got %d", reviews)
	}
	if cancelled != 0 {
		t.Error("cancelled event leaked into expansion")
	}
	if allhands != 1 || allday != 0 { // allday is Sep 20-22 → overlaps range end? Sep 20 < Sep 21 → overlaps!
		if allday != 1 {
			t.Errorf("all-day event in range: %d", allday)
		}
		if allhands != 1 {
			t.Errorf("all-hands: %d", allhands)
		}
	}
}

func TestExpandMonthlyClamp(t *testing.T) {
	ics := `BEGIN:VCALENDAR
BEGIN:VEVENT
UID:m@test
SUMMARY:Rent
DTSTART:20260131T090000Z
RRULE:FREQ=MONTHLY;COUNT=4
END:VEVENT
END:VCALENDAR`
	events, _ := Parse(ics)
	out := Expand(events, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	days := []int{}
	for _, e := range out {
		days = append(days, e.Start.Day())
	}
	// Jan 31, Feb 28 (clamped), Mar 31, Apr 30 (clamped)
	want := []int{31, 28, 31, 30}
	if len(days) != 4 {
		t.Fatalf("got %v", days)
	}
	for i := range want {
		if days[i] != want[i] {
			t.Fatalf("days %v, want %v", days, want)
		}
	}
}
