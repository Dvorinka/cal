package httpapi

// Weekly review: a computed digest of the last 7 days — done count, slipped
// tasks, notes written, busiest day, and the completion streak.

import (
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) weeklyReview(c *gin.Context) {
	userID := currentUser(c).ID
	now := time.Now()
	from := now.AddDate(0, 0, -6).Format(time.DateOnly)
	to := now.Format(time.DateOnly)
	entries, err := s.store.ListEntries(c.Request.Context(), userID, from, to, "")
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	// Slipped tasks may predate the window — look back further for open tasks.
	older, err := s.store.ListEntries(c.Request.Context(), userID, "", from, "")
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}

	done := 0
	slipped := 0
	notes := 0
	perDay := map[string]int{}
	// Streak: consecutive days (ending today or yesterday) with ≥1 completion.
	completedDays := map[string]bool{}
	for _, e := range entries {
		if e.Type == "note" {
			notes++
		}
		if e.Type == "task" && e.Completed {
			done++
			perDay[e.Date]++
			completedDays[e.Date] = true
		}
	}
	for _, e := range older {
		if e.Type == "task" && !e.Completed && e.Date < to {
			slipped++
		}
	}
	streak := 0
	for d := now; ; d = d.AddDate(0, 0, -1) {
		key := d.Format(time.DateOnly)
		if completedDays[key] {
			streak++
			continue
		}
		// Today may legitimately have none yet — don't break on it.
		if key == to {
			continue
		}
		break
	}
	busiest, busiestCount := "", 0
	for day, n := range perDay {
		if n > busiestCount {
			busiest, busiestCount = day, n
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"from":         from,
		"to":           to,
		"tasksDone":    done,
		"tasksSlipped": slipped,
		"notesWritten": notes,
		"streak":       streak,
		"busiestDay":   busiest,
		"busiestCount": busiestCount,
		"perDay":       perDay,
	})
}

// habitStreaks returns current streaks for recurring tasks tagged "habit".
// A streak counts consecutive periods (day/week/month/year) where at least
// one occurrence was completed.
func (s *Server) habitStreaks(c *gin.Context) {
	userID := currentUser(c).ID
	entries, err := s.store.ListEntries(c.Request.Context(), userID, "", "", "")
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	// Group completed task occurrences by (title, recur).
	type key struct{ title, recur string }
	done := map[key]map[string]bool{}
	for _, e := range entries {
		if e.Type != "task" || !e.Completed {
			continue
		}
		k := key{e.Title, e.Recur}
		if done[k] == nil {
			done[k] = map[string]bool{}
		}
		done[k][e.Date] = true
	}
	type habit struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Recur   string `json:"recur"`
		Streak  int    `json:"streak"`
		LastDone string `json:"lastDone,omitempty"`
	}
	out := []habit{}
	seen := map[string]bool{}
	for _, e := range entries {
		if e.Type != "task" || e.Recur == "none" || !hasTag(e.Tags, "habit") || seen[e.Title] {
			continue
		}
		seen[e.Title] = true
		k := key{e.Title, e.Recur}
		dates := done[k]
		streak := 0
		lastDone := ""
		d := time.Now()
		step := recurStep(e.Recur)
		for i := 0; i < 400; i++ {
			day := d.Format(time.DateOnly)
			if dates[day] {
				streak++
				if lastDone == "" {
					lastDone = day
				}
			} else if streak == 0 && day == time.Now().Format(time.DateOnly) {
				// Today unfinished doesn't break the streak.
			} else {
				break
			}
			d = d.AddDate(step.years, step.months, step.days)
		}
		out = append(out, habit{ID: e.ID, Title: e.Title, Recur: e.Recur, Streak: streak, LastDone: lastDone})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Streak > out[j].Streak })
	c.JSON(http.StatusOK, out)
}

type step struct{ years, months, days int }

func recurStep(recur string) step {
	switch recur {
	case "daily":
		return step{days: -1}
	case "weekly":
		return step{days: -7}
	case "monthly":
		return step{months: -1}
	case "yearly":
		return step{years: -1}
	}
	return step{days: -1}
}

func hasTag(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}
