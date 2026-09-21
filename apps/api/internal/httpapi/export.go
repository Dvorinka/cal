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
// ?dry=1 parses the same payload and reports what a restore would merge —
// counts per collection, existing-ID conflicts, orphaned rows — without
// writing anything.
func (s *Server) restoreJSON(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRestoreBytes)
	// Spool to a temp file — zip needs io.ReaderAt anyway, and big archives
	// shouldn't sit in RAM. Lives only for this request.
	tmp, err := os.CreateTemp("", "cal-restore-*")
	if err != nil {
		c.String(http.StatusInternalServerError, "restore failed")
		return
	}
	defer func() { _ = tmp.Close(); _ = os.Remove(tmp.Name()) }()
	size, err := io.Copy(tmp, c.Request.Body)
	if err != nil {
		c.String(http.StatusRequestEntityTooLarge, "file too large")
		return
	}
	body, binaries, ok := parseExportPayload(tmp, size)
	if !ok {
		c.String(http.StatusBadRequest, "expected a Cal export file")
		return
	}
	userID := currentUser(c).ID
	ctx := c.Request.Context()
	if c.Query("dry") == "1" {
		s.previewRestore(c, userID, body, binaries)
		return
	}
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

// parseExportPayload decodes a restore body — plain JSON or the zip archive —
// into the manifest plus the binaries map (zip restores only). Reads the
// spooled file; zip binaries stay lazily attached until the caller closes it.
func parseExportPayload(f *os.File, size int64) (exportPayload, map[string]*zip.File, bool) {
	var body exportPayload
	var binaries map[string]*zip.File
	head := make([]byte, 4)
	_, _ = f.ReadAt(head, 0)
	if bytes.Equal(head, []byte("PK\x03\x04")) {
		zr, zerr := zip.NewReader(f, size)
		if zerr != nil {
			return body, nil, false
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
			return body, nil, false
		}
	} else {
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return body, nil, false
		}
		if json.NewDecoder(f).Decode(&body) != nil {
			return body, nil, false
		}
	}
	if len(body.Entries) == 0 && len(body.People) == 0 && len(body.PersonLinks) == 0 &&
		len(body.PersonTimeline) == 0 && len(body.Files) == 0 {
		return body, nil, false
	}
	if len(body.Entries) > 50000 || len(body.People) > 20000 {
		return body, nil, false
	}
	return body, binaries, true
}

// previewRestore reports what a restore would merge — without writing.
// "new" = rows that would be inserted; "existing" = skipped by id-conflict;
// "orphaned" = links/timeline items whose person isn't local or in the
// payload; files also report rows missing their binary in the archive.
func (s *Server) previewRestore(c *gin.Context, userID string, body exportPayload, binaries map[string]*zip.File) {
	ctx := c.Request.Context()

	entryIDs := make([]string, 0, len(body.Entries))
	for _, e := range body.Entries {
		if e.ID != "" && e.Title != "" && e.Type != "" && e.Date != "" {
			entryIDs = append(entryIDs, e.ID)
		}
	}
	entryExisting, _ := s.store.ExistingIDs(ctx, "entries", userID, entryIDs)

	personIDs := make([]string, 0, len(body.People))
	payloadPersons := map[string]bool{}
	for _, p := range body.People {
		if p.ID != "" && p.Name != "" {
			personIDs = append(personIDs, p.ID)
			payloadPersons[p.ID] = true
		}
	}
	personExisting, _ := s.store.ExistingIDs(ctx, "people", userID, personIDs)
	localPersons, _ := s.store.ExistingIDs(ctx, "people", userID, func() []string {
		ids := make([]string, 0, len(body.PersonLinks)+len(body.PersonTimeline))
		for _, l := range body.PersonLinks {
			ids = append(ids, l.PersonID, l.OtherID)
		}
		for _, t := range body.PersonTimeline {
			ids = append(ids, t.PersonID)
		}
		return ids
	}())
	personKnown := func(id string) bool { return payloadPersons[id] || localPersons[id] }

	linkExisting, _ := s.store.ExistingPersonLinks(ctx, userID, body.PersonLinks)
	linkOrphans, linkValid := 0, 0
	for _, l := range body.PersonLinks {
		if l.PersonID == "" || l.OtherID == "" || l.Kind == "" || l.PersonID == l.OtherID ||
			!personKnown(l.PersonID) || !personKnown(l.OtherID) {
			linkOrphans++
			continue
		}
		linkValid++
	}

	timelineIDs := make([]string, 0, len(body.PersonTimeline))
	tlOrphans := 0
	for _, t := range body.PersonTimeline {
		if t.ID == "" || t.PersonID == "" || t.Title == "" || !personKnown(t.PersonID) {
			tlOrphans++
			continue
		}
		timelineIDs = append(timelineIDs, t.ID)
	}
	tlExisting, _ := s.store.ExistingIDs(ctx, "person_timeline", userID, timelineIDs)

	fileIDs := make([]string, 0, len(body.Files))
	fileNames := make([]string, 0, len(body.Files))
	fileHashes := make([]string, 0, len(body.Files))
	for _, f := range body.Files {
		if f.ID != "" && f.Name != "" && reSafeName.MatchString(f.Name) {
			fileIDs = append(fileIDs, f.ID)
			fileNames = append(fileNames, f.Name)
			if f.Sha256 != "" {
				fileHashes = append(fileHashes, f.Sha256)
			}
		}
	}
	fileExistingIDs, _ := s.store.ExistingIDs(ctx, "files", userID, fileIDs)
	fileTakenNames, _ := s.store.ExistingFileNames(ctx, userID, fileNames)
	fileKnownHashes, _ := s.store.ExistingFileHashes(ctx, userID, fileHashes)
	fileNew, fileNoBinary, fileExisting, fileInvalid := 0, 0, 0, 0
	for _, f := range body.Files {
		if f.ID == "" || f.Name == "" || !reSafeName.MatchString(f.Name) {
			fileInvalid++
			continue
		}
		if fileExistingIDs[f.ID] || fileTakenNames[f.Name] {
			fileExisting++
			continue
		}
		// Restorable with an archive binary or an on-disk twin (hardlink).
		if _, ok := binaries[f.Name]; !ok && !fileKnownHashes[f.Sha256] {
			fileNoBinary++
			continue
		}
		fileNew++
	}

	c.JSON(http.StatusOK, gin.H{
		"dryRun":   true,
		"entries":  gin.H{"total": len(body.Entries), "new": len(entryIDs) - len(entryExisting), "existing": len(entryExisting), "invalid": len(body.Entries) - len(entryIDs)},
		"people":   gin.H{"total": len(body.People), "new": len(personIDs) - len(personExisting), "existing": len(personExisting), "invalid": len(body.People) - len(personIDs)},
		"links":    gin.H{"total": len(body.PersonLinks), "new": linkValid - linkExisting, "existing": linkExisting, "orphaned": linkOrphans},
		"timeline": gin.H{"total": len(body.PersonTimeline), "new": len(timelineIDs) - len(tlExisting), "existing": len(tlExisting), "orphaned": tlOrphans},
		"files":    gin.H{"total": len(body.Files), "new": fileNew, "existing": fileExisting, "noBinary": fileNoBinary, "invalid": fileInvalid},
	})
}

// writeRestoredFiles unpacks archive binaries to the uploads dir and returns
// the rows worth inserting (binary written, name free).
func (s *Server) writeRestoredFiles(ctx context.Context, userID string, files []store.File, binaries map[string]*zip.File) []store.File {
	dir := filepath.Join(s.dataDir, "uploads", userID)
	kept := make([]store.File, 0, len(files))
	for _, f := range files {
		if !reSafeName.MatchString(f.Name) {
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
		// Content dedup: an identical binary already on disk gets a hardlink —
		// zero extra bytes, works even for JSON exports that carry no binaries.
		if f.Sha256 != "" {
			if dup, err := s.store.FileByHash(ctx, userID, f.Sha256); err == nil {
				if os.Link(filepath.Join(dir, dup.Name), dst) == nil {
					kept = append(kept, f)
				}
				continue
			}
		}
		zf, ok := binaries[f.Name]
		if !ok {
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
