package store

import "testing"

func TestNextRecurDate(t *testing.T) {
	cases := []struct {
		date  string
		recur string
		want  string
		ok    bool
	}{
		{"2026-09-10", "daily", "2026-09-11", true},
		{"2026-09-10", "weekly", "2026-09-17", true},
		{"2026-09-10", "monthly", "2026-10-10", true},
		{"2026-01-31", "monthly", "2026-02-28", true},
		{"2024-01-31", "monthly", "2024-02-29", true},
		{"2026-09-10", "yearly", "2027-09-10", true},
		{"2026-09-10", "none", "", false},
		{"2026-09-10", "bogus", "", false},
		{"not-a-date", "daily", "", false},
	}
	for _, tc := range cases {
		got, ok := NextRecurDate(tc.date, tc.recur)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("NextRecurDate(%q, %q) = %q, %v; want %q, %v", tc.date, tc.recur, got, ok, tc.want, tc.ok)
		}
	}
}
