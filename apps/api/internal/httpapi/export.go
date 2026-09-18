package httpapi

// Export/restore plus the subscribable ICS feed of the user's own entries -
// the inverse of feed import. The ICS feed is authenticated by the read-only
// widget token, so the URL can live in a calendar app without exposing full
// API access.

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cal/apps/api/internal/ical"
	"cal/apps/api/internal/store"

	"github.com/gin-gonic/gin"
)

const (
	exportManifestName = "cal-export.json"
	// Uploads cap at 20 MB each; a zip export of a heavy account is still
	// bounded well under this.
	maxRestoreBytes = 512 << 20
)

// exportPayload is the restore-side view of the /api/export manifest.
type exportPayload struct {
	Entries        []store.Entry          `json:"entries"`
	People         []store.Person         `json:"people"`
	PersonLinks    []store.PersonRelation `json:"personLinks"`
	PersonTimeline []store.TimelineItem   `json:"personTimeline"`
	Files          []store.File           `json:"files"`
}

// restoreJSON accepts the /export payload - plain JSON, or the zip archive
// (cal-export.json manifest + files/<name> binaries) - and re-inserts
// entries, people, person links, timeline items and file rows. Additive:
// existing IDs are skipped, so a restore never clobbers current data.
func (s *Server) restoreJSON(c *gin.Context) {
	raw, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, maxRestoreBytes))
	if err != nil {
		c.String(http.StatusRequestEntityTooLarge, "file too large")
		return
	}
	var body exportPayload
	// binaries maps stored file name → zip member, present only for zip restores.
	var binaries map[string]*zip.File
	if bytes.HasPrefix(raw, []byte("PK\x03\x04")) {
		zr, zerr := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
		if zerr != nil {
			c.String(http.StatusBadRequest, "invalid zip archive")
			return
		}
		binaries = map[string]*zip.File{}
		var manifest []byte
		for _, zf := range zr.File {
			switch {
			case zf.Name == exportManifestName:
				if rc, oerr := zf.Open(); oerr == nil {
					manifest, _ = io.ReadAll(io.LimitReader(rc, 64<<20))
					_ = rc.Close()
				}
			case strings.HasPrefix(zf.Name, "files/"):
				binaries[strings.TrimPrefix(zf.Name, "files/")] = zf
			}
		}
		if manifest == nil || json.Unmarshal(manifest, &body) != nil {
			c.String(http.StatusBadRequest, "expected a Cal export archive")
			return
		}
	} else if json.Unmarshal(raw, &body) != nil {
		c.String(http.StatusBadRequest, "expected a Cal export file")
		return
	}
	if len(body.Entries) == 0 && len(body.People) == 0 && len(body.PersonLinks) == 0 &&
		len(body.PersonTimeline) == 0 && len(body.Files) == 0 {
		c.String(http.StatusBadRequest, "expected a Cal export file")
		return
	}
	if len(body.Entries) > 50000 || len(body.People) > 20000 {
		c.String(http.StatusBadRequest, "file too large")
		return
	}
	userID := currentUser(c).ID
	ctx := c.Request.Context()
	imported, err := s.store.RestoreEntries(ctx, userID, body.Entries)
	if err != nil {
		c.String(http.StatusInternalServerError, "restore failed")
		return
	}
	// People first — links, timeline and attachments reference person IDs.
	people, err := s.store.RestorePeople(ctx, userID, body.People)
	if err != nil {
		c.String(http.StatusInternalServerError, "restore failed")
		return
	}
	links, err := s.store.RestorePersonLinks(ctx, userID, body.PersonLinks)
	if err != nil {
		c.String(http.StatusInternalServerError, "restore failed")
		return
	}
	timeline, err := s.store.RestorePersonTimeline(ctx, userID, body.PersonTimeline)
	if err != nil {
		c.String(http.StatusInternalServerError, "restore failed")
		return
	}
	// Files need their binary in the archive and a free disk name; manifest
	// rows without a binary are skipped, never restored dangling.
	kept := s.writeRestoredFiles(ctx, userID, body.Files, binaries)
	files, err := s.store.RestoreFiles(ctx, userID, kept)
	if err != nil {
		c.String(http.StatusInternalServerError, "restore failed")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"restored": imported, "people": people, "links": links, "timeline": timeline,
		"files": files, "filesSkipped": len(body.Files) - files,
	})
}

// writeRestoredFiles unpacks archive binaries to the uploads dir and returns
// the rows worth inserting (binary written, name free).
func (s *Server) writeRestoredFiles(ctx context.Context, userID string, files []store.File, binaries map[string]*zip.File) []store.File {
	if binaries == nil {
		return nil
	}
	dir := filepath.Join(s.dataDir, "uploads", userID)
	kept := make([]store.File, 0, len(files))
	for _, f := range files {
		zf, ok := binaries[f.Name]
		if !ok || !reSafeName.MatchString(f.Name) {
			continue
		}
		// A row or binary already occupying the name wins — additive restore.
		if _, err := s.store.FileByName(ctx, userID, f.Name); err == nil {
			continue
		}
		// The row insert skips on global id-conflict (e.g. restoring one
		// account's export into another account on the same instance) — don't
		// write an unreachable binary in that case.
		if taken, err := s.store.FileIDExists(ctx, f.ID); err != nil || taken {
			continue
		}
		dst := filepath.Join(dir, f.Name)
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			continue
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			_ = rc.Close()
			continue
		}
		out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_, err = io.Copy(out, io.LimitReader(rc, maxUploadBytes+1))
			_ = out.Close()
		}
		_ = rc.Close()
		if err != nil {
			_ = os.Remove(dst)
			continue
		}
		kept = append(kept, f)
	}
	return kept
}

func (s *Server) exportICS(c *gin.Context) {
	user, err := s.store.UserByWidgetToken(c.Request.Context(), c.Query("token"))
	if err != nil {
		c.String(http.StatusUnauthorized, "invalid token")
		return
	}
	settings, _ := s.store.Settings(c.Request.Context(), user.ID)
	loc := time.Local
	if l, err := time.LoadLocation(settings.Timezone); err == nil {
		loc = l
	}
	entries, err := s.store.ListEntries(c.Request.Context(), user.ID, "", "", "")
	if err != nil {
		c.String(http.StatusInternalServerError, "failed")
		return
	}
	events := make([]ical.Event, 0, len(entries))
	for _, e := range entries {
		if e.Type == "note" || e.Type == "link" {
			continue
		}
		events = append(events, entryToEvent(e, loc))
	}
	c.Data(http.StatusOK, "text/calendar; charset=utf-8", []byte(ical.EncodeCalendar(events)))
}

func entryToEvent(e store.Entry, loc *time.Location) ical.Event {
	ev := ical.Event{
		UID:         "cal-" + e.ID,
		Summary:     e.Title,
		Description: e.Content,
		URL:         e.LinkURL,
		Completed:   e.Type == "task" && e.Completed,
		TZ:          loc,
	}
	day, err := time.Parse(time.DateOnly, e.Date)
	if err != nil {
		return ev
	}
	if e.StartTime == nil || *e.StartTime == "" {
		ev.AllDay = true
		ev.Start = day
	} else {
		if t, err := time.ParseInLocation("2006-01-02 15:04", e.Date+" "+*e.StartTime, loc); err == nil {
			ev.Start = t
		}
		if e.EndTime != nil {
			if t, err := time.ParseInLocation("2006-01-02 15:04", e.Date+" "+*e.EndTime, loc); err == nil {
				ev.End = t
			}
		}
	}
	switch e.Recur {
	case "daily":
		ev.Recurrence = &ical.RRule{Freq: "DAILY"}
	case "weekly":
		ev.Recurrence = &ical.RRule{Freq: "WEEKLY"}
	case "monthly":
		ev.Recurrence = &ical.RRule{Freq: "MONTHLY"}
	case "yearly":
		ev.Recurrence = &ical.RRule{Freq: "YEARLY"}
	}
	return ev
}
