package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")
var ErrInvalid = errors.New("invalid entry")

type Store struct {
	db *pgxpool.Pool
}

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// entryCols is the canonical SELECT/RETURNING column list for entries.
// Times are rendered as HH:MM strings to keep the API surface simple.
const entryCols = `id::text, title, content, type, link_url, date::text,
	to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI'),
	completed, color, tags, recur, remind, pinned, created_at,
	account_id::text, external_uid, external_href, external_etag, dirty`

type Entry struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Content      string    `json:"content,omitempty"`
	Type         string    `json:"type"`
	LinkURL      string    `json:"linkUrl,omitempty"`
	Date         string    `json:"date"`
	StartTime    *string   `json:"startTime,omitempty"`
	EndTime      *string   `json:"endTime,omitempty"`
	Completed    bool      `json:"completed"`
	Pinned       bool      `json:"pinned"`
	Color        string    `json:"color"`
	Tags         []string  `json:"tags"`
	Recur        string    `json:"recur"`
	Remind       *int      `json:"remind,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	OwnerID      string    `json:"-"`
	AccountID    *string   `json:"accountId,omitempty"`
	ExternalUID  *string   `json:"-"`
	ExternalHref *string   `json:"-"`
	ExternalETag *string   `json:"-"`
	Dirty        bool      `json:"-"`
}

func (e *Entry) scan(row interface{ Scan(...any) error }) error {
	return row.Scan(&e.ID, &e.Title, &e.Content, &e.Type, &e.LinkURL, &e.Date,
		&e.StartTime, &e.EndTime, &e.Completed, &e.Color, &e.Tags, &e.Recur, &e.Remind, &e.Pinned, &e.CreatedAt,
		&e.AccountID, &e.ExternalUID, &e.ExternalHref, &e.ExternalETag, &e.Dirty)
}

type Settings struct {
	Country      string `json:"country"`
	ShowHolidays bool   `json:"showHolidays"`
	Theme        string `json:"theme"`
	WeekStart    string `json:"weekStart"`
	Accent       string `json:"accent"`
	Timezone     string `json:"timezone"`
	City         string `json:"city"`
	QuotaMB      int    `json:"quotaMb"`
	WidgetToken  string `json:"widgetToken"`
	ApiToken     string `json:"apiToken"`
}

type Feed struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	URL       string     `json:"url"`
	Color     string     `json:"color"`
	FetchedAt *time.Time `json:"fetchedAt,omitempty"`
}

type EntryInput struct {
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	Type      string   `json:"type"`
	LinkURL   string   `json:"linkUrl"`
	Date      string   `json:"date"`
	StartTime string   `json:"startTime"`
	EndTime   string   `json:"endTime"`
	Color     string   `json:"color"`
	Tags      []string `json:"tags"`
	Recur     string   `json:"recur"`
	Remind    *int     `json:"remind"`
	AccountID *string  `json:"accountId"`
}

type EntryPatch struct {
	Title     *string  `json:"title"`
	Content   *string  `json:"content"`
	Type      *string  `json:"type"`
	LinkURL   *string  `json:"linkUrl"`
	Date      *string  `json:"date"`
	StartTime *string  `json:"startTime"`
	EndTime   *string  `json:"endTime"`
	Completed *bool    `json:"completed"`
	Pinned    *bool    `json:"pinned"`
	Color     *string  `json:"color"`
	Recur     *string  `json:"recur"`
	Remind    *int     `json:"remind"`
	Tags      []string `json:"tags"`
	HasTags   bool     `json:"-"`
}

func New(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	return pgxpool.NewWithConfig(ctx, cfg)
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *Store) CreateUser(ctx context.Context, email, passwordHash string) (User, error) {
	var u User
	err := s.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash)
		VALUES ($1, $2)
		RETURNING id::text, email
	`, NormalizeEmail(email), passwordHash).Scan(&u.ID, &u.Email)
	if err != nil {
		return User{}, err
	}
	_, err = s.db.Exec(ctx, `INSERT INTO settings (user_id) VALUES ($1)`, u.ID)
	return u, err
}

func (s *Store) UserByEmail(ctx context.Context, email string) (User, string, error) {
	var u User
	var hash string
	err := s.db.QueryRow(ctx, `
		SELECT id::text, email, password_hash FROM users WHERE email = $1
	`, NormalizeEmail(email)).Scan(&u.ID, &u.Email, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, "", ErrNotFound
	}
	return u, hash, err
}

func (s *Store) UserBySession(ctx context.Context, sessionID string) (User, error) {
	var u User
	err := s.db.QueryRow(ctx, `
		SELECT users.id::text, users.email
		FROM sessions
		JOIN users ON users.id = sessions.user_id
		WHERE sessions.id = $1 AND sessions.expires_at > now()
	`, sessionID).Scan(&u.ID, &u.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (s *Store) CreateSession(ctx context.Context, userID, userAgent string) (string, error) {
	_, _ = s.db.Exec(ctx, `DELETE FROM sessions WHERE expires_at < now()`)
	id := uuid.NewString()
	_, err := s.db.Exec(ctx, `
		INSERT INTO sessions (id, user_id, expires_at, user_agent, last_seen_at)
		VALUES ($1, $2, now() + interval '30 days', nullif($3, ''), now())
	`, id, userID, userAgent)
	return id, err
}

// SessionInfo describes one live session for the security panel.
type SessionInfo struct {
	ID        string     `json:"id"`
	UserAgent *string    `json:"userAgent,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	LastSeen  *time.Time `json:"lastSeen,omitempty"`
	Current   bool       `json:"current"`
}

// ListSessions returns the user's live sessions; currentID marks "this one".
func (s *Store) ListSessions(ctx context.Context, userID, currentID string) ([]SessionInfo, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, user_agent, created_at, last_seen_at, id::text = $2
		FROM sessions
		WHERE user_id = $1 AND expires_at > now()
		ORDER BY last_seen_at DESC NULLS LAST
	`, userID, currentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SessionInfo{}
	for rows.Next() {
		var si SessionInfo
		if err := rows.Scan(&si.ID, &si.UserAgent, &si.CreatedAt, &si.LastSeen, &si.Current); err != nil {
			return nil, err
		}
		out = append(out, si)
	}
	return out, rows.Err()
}

// TouchSession refreshes last_seen_at (cheap activity signal).
func (s *Store) TouchSession(ctx context.Context, sessionID string) {
	_, _ = s.db.Exec(ctx, `UPDATE sessions SET last_seen_at = now() WHERE id = $1`, sessionID)
}

// RevokeSession deletes one session owned by the user.
func (s *Store) RevokeSession(ctx context.Context, userID, sessionID string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM sessions WHERE id = $1 AND user_id = $2`, sessionID, userID)
	return err
}

// ChangePassword verifies the old password and sets a new hash.
func (s *Store) ChangePassword(ctx context.Context, userID, newHash string) error {
	_, err := s.db.Exec(ctx, `UPDATE users SET password_hash = $1 WHERE id = $2`, newHash, userID)
	return err
}

// RevokeOtherSessions deletes every session except the caller's.
func (s *Store) RevokeOtherSessions(ctx context.Context, userID, keepID string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1 AND id <> $2`, userID, keepID)
	return err
}

func (s *Store) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, sessionID)
	return err
}

func (s *Store) ListEntries(ctx context.Context, userID, from, to, q string) ([]Entry, error) {
	query := `
		SELECT ` + entryCols + `
		FROM entries
		WHERE user_id = $1
	`
	args := []any{userID}
	if from != "" {
		query += ` AND date >= $2`
		args = append(args, from)
	}
	if to != "" {
		query += fmt.Sprintf(" AND date <= $%d", len(args)+1)
		args = append(args, to)
	}
	if strings.TrimSpace(q) != "" {
		query += fmt.Sprintf(" AND (title ILIKE $%d OR content ILIKE $%d OR $%d = ANY(tags))",
			len(args)+1, len(args)+2, len(args)+3)
		like := "%" + strings.TrimSpace(q) + "%"
		args = append(args, like, like, strings.TrimSpace(q))
	}
	query += ` ORDER BY date ASC, created_at ASC`
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []Entry{}
	for rows.Next() {
		var e Entry
		if err := e.scan(rows); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *Store) CreateEntry(ctx context.Context, userID string, input EntryInput) (Entry, error) {
	if input.Color == "" {
		input.Color = "slate"
	}
	if input.Tags == nil {
		input.Tags = []string{}
	}
	var e Entry
	err := s.db.QueryRow(ctx, `
		INSERT INTO entries (user_id, title, content, type, link_url, date, start_time, end_time, color, tags, recur, remind, account_id, dirty)
		VALUES ($1, $2, $3, $4, $5, $6, nullif($7, '')::time, nullif($8, '')::time, $9, $10, coalesce(nullif($11, ''), 'none'), $12, $13::uuid, $13 IS NOT NULL)
		RETURNING `+entryCols+`
	`, userID, input.Title, input.Content, input.Type, input.LinkURL, input.Date, input.StartTime, input.EndTime, input.Color, input.Tags, input.Recur, input.Remind, input.AccountID).
		Scan(&e.ID, &e.Title, &e.Content, &e.Type, &e.LinkURL, &e.Date, &e.StartTime, &e.EndTime, &e.Completed, &e.Color, &e.Tags, &e.Recur, &e.Remind, &e.Pinned, &e.CreatedAt,
			&e.AccountID, &e.ExternalUID, &e.ExternalHref, &e.ExternalETag, &e.Dirty)
	return e, err
}

func (s *Store) UpdateEntry(ctx context.Context, userID, id string, patch EntryPatch) (Entry, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Entry{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	current, err := s.entryTx(ctx, tx, userID, id)
	if err != nil {
		return Entry{}, err
	}
	if patch.Title != nil {
		current.Title = *patch.Title
	}
	if patch.Content != nil {
		current.Content = *patch.Content
	}
	if patch.Type != nil {
		current.Type = *patch.Type
	}
	if patch.LinkURL != nil {
		current.LinkURL = *patch.LinkURL
	}
	if patch.Date != nil {
		current.Date = *patch.Date
	}
	if patch.StartTime != nil {
		current.StartTime = patch.StartTime
	}
	if patch.EndTime != nil {
		current.EndTime = patch.EndTime
	}
	if patch.Recur != nil {
		current.Recur = *patch.Recur
	}
	if patch.Completed != nil {
		current.Completed = *patch.Completed
	}
	if patch.Pinned != nil {
		current.Pinned = *patch.Pinned
	}
	if patch.Color != nil {
		current.Color = *patch.Color
	}
	if patch.Remind != nil {
		current.Remind = patch.Remind
	}
	if patch.HasTags {
		current.Tags = patch.Tags
	}

	// Re-validate the merged row so a partial patch cannot violate the
	// time-ordering or recur-on-task rules the DB enforces.
	if strings.TrimSpace(current.Title) == "" || current.Type == "" {
		return Entry{}, ErrInvalid
	}
	if current.Type != "task" && current.Recur != "none" {
		return Entry{}, ErrInvalid
	}
	if current.StartTime == nil && current.EndTime != nil {
		return Entry{}, ErrInvalid
	}
	if current.StartTime != nil && current.EndTime != nil && *current.EndTime <= *current.StartTime {
		return Entry{}, ErrInvalid
	}

	// Snapshot the pre-update row so every change is recoverable.
	if _, err := tx.Exec(ctx, `
		INSERT INTO entry_revisions (entry_id, user_id, title, content, type, link_url, date, start_time, end_time, completed, color, tags, recur, remind)
		SELECT id, user_id, title, content, type, link_url, date, start_time, end_time, completed, color, tags, recur, remind
		FROM entries WHERE id = $1
	`, id); err != nil {
		return Entry{}, err
	}

	// Rescheduling clears the reminder so it can fire again.
	resetRemind := patch.Remind != nil || patch.StartTime != nil || patch.Date != nil

	var e Entry
	err = tx.QueryRow(ctx, `
		UPDATE entries
		SET title = $1, content = $2, type = $3, link_url = $4, date = $5,
		    start_time = nullif($6, '')::time, end_time = nullif($7, '')::time,
		    completed = $8, color = $9, tags = $10, recur = $11, remind = $12,
		    pinned = $16,
		    dirty = CASE WHEN account_id IS NOT NULL THEN true ELSE dirty END,
		    reminded_at = CASE WHEN $15 THEN NULL ELSE reminded_at END
		WHERE id = $13 AND user_id = $14
		RETURNING `+entryCols+`
	`, current.Title, current.Content, current.Type, current.LinkURL, current.Date,
		strOrEmpty(current.StartTime), strOrEmpty(current.EndTime),
		current.Completed, current.Color, current.Tags, current.Recur, current.Remind, id, userID, resetRemind, current.Pinned).
		Scan(&e.ID, &e.Title, &e.Content, &e.Type, &e.LinkURL, &e.Date, &e.StartTime, &e.EndTime, &e.Completed, &e.Color, &e.Tags, &e.Recur, &e.Remind, &e.Pinned, &e.CreatedAt,
			&e.AccountID, &e.ExternalUID, &e.ExternalHref, &e.ExternalETag, &e.Dirty)
	if err != nil {
		return Entry{}, err
	}

	// Completing a recurring task spawns its next occurrence.
	if patch.Completed != nil && *patch.Completed && e.Type == "task" && e.Recur != "none" {
		next, ok := NextRecurDate(e.Date, e.Recur)
		if ok {
			_, err = tx.Exec(ctx, `
				INSERT INTO entries (user_id, title, content, type, link_url, date, start_time, end_time, color, tags, recur)
				VALUES ($1, $2, $3, $4, $5, $6, nullif($7, '')::time, nullif($8, '')::time, $9, $10, $11)
			`, userID, e.Title, e.Content, e.Type, e.LinkURL, next,
				strOrEmpty(e.StartTime), strOrEmpty(e.EndTime), e.Color, e.Tags, e.Recur)
			if err != nil {
				return Entry{}, err
			}
		}
	}

	return e, tx.Commit(ctx)
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// NextRecurDate returns the date of the next occurrence for a recurrence rule.
func NextRecurDate(date, recur string) (string, bool) {
	d, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return "", false
	}
	var next time.Time
	switch recur {
	case "daily":
		next = d.AddDate(0, 0, 1)
	case "weekly":
		next = d.AddDate(0, 0, 7)
	case "monthly":
		next = addMonthsClamped(d, 1)
	case "yearly":
		next = addMonthsClamped(d, 12)
	default:
		return "", false
	}
	return next.Format(time.DateOnly), true
}

// addMonthsClamped adds months while clamping the day to the target month's
// length, so Jan 31 + 1 month lands on Feb 28/29 rather than Mar 3.
func addMonthsClamped(d time.Time, months int) time.Time {
	day := d.Day()
	next := time.Date(d.Year(), d.Month()+time.Month(months), 1, 0, 0, 0, 0, time.UTC)
	last := next.AddDate(0, 1, -1).Day()
	if day > last {
		day = last
	}
	return time.Date(next.Year(), next.Month(), day, 0, 0, 0, 0, time.UTC)
}

func (s *Store) DeleteEntry(ctx context.Context, userID, id string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM entries WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AllUserIDs returns every user id — used by the nightly backup.
func (s *Store) AllUserIDs(ctx context.Context) ([]string, error) {
	rows, err := s.db.Query(ctx, `SELECT id::text FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// RestoreEntries re-inserts exported entries under the user, skipping IDs that
// already exist. Sync metadata is stripped — a restored entry is local.
func (s *Store) RestoreEntries(ctx context.Context, userID string, entries []Entry) (int, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	imported := 0
	for _, e := range entries {
		if e.Title == "" || e.Type == "" || e.Date == "" {
			continue
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO entries (id, user_id, title, content, type, link_url, date, start_time, end_time, completed, color, tags, recur, remind)
			VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, nullif($8, '')::time, nullif($9, '')::time, $10, $11, $12, coalesce(nullif($13, ''), 'none'), $14)
			ON CONFLICT (id) DO NOTHING
		`, e.ID, userID, e.Title, e.Content, e.Type, e.LinkURL, e.Date, strOr(e.StartTime), strOr(e.EndTime), e.Completed, e.Color, e.Tags, e.Recur, e.Remind)
		if err != nil {
			return imported, err
		}
		imported += int(tag.RowsAffected())
	}
	return imported, tx.Commit(ctx)
}

func strOr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// Revision is a point-in-time snapshot of an entry.
type Revision struct {
	ID        string    `json:"id"`
	EntryID   string    `json:"entryId"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Type      string    `json:"type"`
	Date      string    `json:"date"`
	StartTime *string   `json:"startTime,omitempty"`
	EndTime   *string   `json:"endTime,omitempty"`
	Completed bool      `json:"completed"`
	Color     string    `json:"color"`
	Tags      []string  `json:"tags"`
	Recur     string    `json:"recur"`
	Remind    *int      `json:"remind,omitempty"`
	SavedAt   time.Time `json:"savedAt"`
}

// EntryRevisions lists snapshots for an entry, newest first (capped at 50).
func (s *Store) EntryRevisions(ctx context.Context, userID, entryID string) ([]Revision, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, entry_id::text, title, coalesce(content, ''), type, date::text,
		       to_char(start_time, 'HH24:MI'), to_char(end_time, 'HH24:MI'),
		       completed, color, tags, recur, remind, saved_at
		FROM entry_revisions
		WHERE entry_id = $1 AND user_id = $2
		ORDER BY saved_at DESC
		LIMIT 50
	`, entryID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Revision{}
	for rows.Next() {
		var r Revision
		if err := rows.Scan(&r.ID, &r.EntryID, &r.Title, &r.Content, &r.Type, &r.Date, &r.StartTime, &r.EndTime,
			&r.Completed, &r.Color, &r.Tags, &r.Recur, &r.Remind, &r.SavedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// RestoreRevision copies a snapshot's fields back onto the live entry. The
// pre-restore state is snapshotted first so a restore is itself revertible.
func (s *Store) RestoreRevision(ctx context.Context, userID, entryID, revID string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		INSERT INTO entry_revisions (entry_id, user_id, title, content, type, link_url, date, start_time, end_time, completed, color, tags, recur, remind)
		SELECT id, user_id, title, content, type, link_url, date, start_time, end_time, completed, color, tags, recur, remind
		FROM entries WHERE id = $1
	`, entryID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE entries e
		SET title = r.title, content = r.content, type = r.type, link_url = r.link_url,
		    date = r.date, start_time = r.start_time, end_time = r.end_time,
		    completed = r.completed, color = r.color, tags = r.tags, recur = r.recur, remind = r.remind,
		    dirty = CASE WHEN e.account_id IS NOT NULL THEN true ELSE e.dirty END,
		    reminded_at = NULL
		FROM entry_revisions r
		WHERE e.id = r.entry_id AND r.id = $1 AND e.id = $2 AND e.user_id = $3
	`, revID, entryID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

// Webhook is a user-registered outbound endpoint.
type Webhook struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Secret    string    `json:"secret"`
	CreatedAt time.Time `json:"createdAt"`
}

func (s *Store) Webhooks(ctx context.Context, userID string) ([]Webhook, error) {
	rows, err := s.db.Query(ctx, `SELECT id::text, url, secret, created_at FROM webhooks WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Webhook{}
	for rows.Next() {
		var w Webhook
		if err := rows.Scan(&w.ID, &w.URL, &w.Secret, &w.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (s *Store) CreateWebhook(ctx context.Context, userID, url, secret string) (Webhook, error) {
	var w Webhook
	err := s.db.QueryRow(ctx, `
		INSERT INTO webhooks (user_id, url, secret) VALUES ($1, $2, $3)
		RETURNING id::text, url, secret, created_at
	`, userID, url, secret).Scan(&w.ID, &w.URL, &w.Secret, &w.CreatedAt)
	return w, err
}

func (s *Store) DeleteWebhook(ctx context.Context, userID, id string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM webhooks WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

// Entry returns one entry owned by the user.
func (s *Store) Entry(ctx context.Context, userID, id string) (Entry, error) {
	var e Entry
	err := e.scan(s.db.QueryRow(ctx, `
		SELECT `+entryCols+` FROM entries WHERE id = $1 AND user_id = $2
	`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Entry{}, ErrNotFound
	}
	return e, err
}

func (s *Store) entryTx(ctx context.Context, tx pgx.Tx, userID, id string) (Entry, error) {
	var e Entry
	err := e.scan(tx.QueryRow(ctx, `
		SELECT `+entryCols+`
		FROM entries
		WHERE id = $1 AND user_id = $2
	`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Entry{}, ErrNotFound
	}
	return e, err
}

func (s *Store) Settings(ctx context.Context, userID string) (Settings, error) {
	var settings Settings
	err := s.db.QueryRow(ctx, `
		SELECT country, show_holidays, theme, week_start, accent, timezone, coalesce(city, ''), quota_mb, widget_token, api_token FROM settings WHERE user_id = $1
	`, userID).Scan(&settings.Country, &settings.ShowHolidays, &settings.Theme, &settings.WeekStart, &settings.Accent, &settings.Timezone, &settings.City, &settings.QuotaMB, &settings.WidgetToken, &settings.ApiToken)
	return settings, err
}

func (s *Store) UpdateSettings(ctx context.Context, userID string, settings Settings) (Settings, error) {
	var out Settings
	err := s.db.QueryRow(ctx, `
		INSERT INTO settings (user_id, country, show_holidays, theme, week_start, accent, timezone, city)
		VALUES ($1, $2, $3, $4, $5, $6, coalesce(nullif($7, ''), 'UTC'), nullif($8, ''))
		ON CONFLICT (user_id) DO UPDATE
		SET country = EXCLUDED.country,
		    show_holidays = EXCLUDED.show_holidays,
		    theme = EXCLUDED.theme,
		    week_start = EXCLUDED.week_start,
		    accent = EXCLUDED.accent,
		    timezone = EXCLUDED.timezone,
		    city = EXCLUDED.city
		RETURNING country, show_holidays, theme, week_start, accent, timezone, coalesce(city, ''), quota_mb, widget_token, api_token
	`, userID, settings.Country, settings.ShowHolidays, settings.Theme, settings.WeekStart, settings.Accent, settings.Timezone, settings.City).
		Scan(&out.Country, &out.ShowHolidays, &out.Theme, &out.WeekStart, &out.Accent, &out.Timezone, &out.City, &out.QuotaMB, &out.WidgetToken, &out.ApiToken)
	return out, err
}

// RegenerateWidgetToken rotates the read-only widget share token.
func (s *Store) RegenerateWidgetToken(ctx context.Context, userID string) (string, error) {
	var token string
	err := s.db.QueryRow(ctx, `
		UPDATE settings SET widget_token = replace(gen_random_uuid()::text, '-', '')
		WHERE user_id = $1 RETURNING widget_token
	`, userID).Scan(&token)
	return token, err
}

// UserByWidgetToken resolves the owner of a widget share token (read-only use).
func (s *Store) UserByWidgetToken(ctx context.Context, token string) (User, error) {
	var u User
	err := s.db.QueryRow(ctx, `
		SELECT users.id::text, users.email
		FROM settings JOIN users ON users.id = settings.user_id
		WHERE settings.widget_token = $1
	`, token).Scan(&u.ID, &u.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

// RegenerateApiToken rotates the MCP/API access token.
func (s *Store) RegenerateApiToken(ctx context.Context, userID string) (string, error) {
	var token string
	err := s.db.QueryRow(ctx, `
		UPDATE settings SET api_token = replace(gen_random_uuid()::text, '-', '')
		WHERE user_id = $1 RETURNING api_token
	`, userID).Scan(&token)
	return token, err
}

// UserByApiToken resolves the owner of an API token (used by the MCP endpoint).
func (s *Store) UserByApiToken(ctx context.Context, token string) (User, error) {
	var u User
	err := s.db.QueryRow(ctx, `
		SELECT users.id::text, users.email
		FROM settings JOIN users ON users.id = settings.user_id
		WHERE settings.api_token = $1
	`, token).Scan(&u.ID, &u.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

// ---------- Push ----------

type PushSubscription struct {
	ID       string `json:"id"`
	Endpoint string `json:"endpoint"`
	P256dh   string `json:"p256dh"`
	Auth     string `json:"auth"`
	Label    string `json:"label"`
}

func (s *Store) UpsertPushSub(ctx context.Context, userID, endpoint, p256dh, auth, label string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO push_subscriptions (id, user_id, endpoint, p256dh, auth, label)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (endpoint) DO UPDATE SET p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth, user_id = EXCLUDED.user_id, label = EXCLUDED.label
	`, uuid.NewString(), userID, endpoint, p256dh, auth, label)
	return err
}

// DeletePushSubByID removes a subscription owned by the user.
func (s *Store) DeletePushSubByID(ctx context.Context, userID, id string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM push_subscriptions WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (s *Store) DeletePushSub(ctx context.Context, endpoint string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM push_subscriptions WHERE endpoint = $1`, endpoint)
	return err
}

func (s *Store) PushSubs(ctx context.Context, userID string) ([]PushSubscription, error) {
	rows, err := s.db.Query(ctx, `SELECT id::text, endpoint, p256dh, auth, coalesce(label, '') FROM push_subscriptions WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	subs := []PushSubscription{}
	for rows.Next() {
		var sub PushSubscription
		if err := rows.Scan(&sub.ID, &sub.Endpoint, &sub.P256dh, &sub.Auth, &sub.Label); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

// DueReminders returns entries whose reminder moment has arrived but has not
// been pushed yet. Entry.OwnerID carries the owning user.
func (s *Store) DueReminders(ctx context.Context) ([]Entry, error) {
	rows, err := s.db.Query(ctx, `
		SELECT e.id::text, e.title, e.content, e.type, e.link_url, e.date::text,
		       to_char(e.start_time, 'HH24:MI'), to_char(e.end_time, 'HH24:MI'),
		       e.completed, e.color, e.tags, e.recur, e.remind, e.created_at,
		       e.account_id::text, e.external_uid, e.external_href, e.external_etag, e.dirty,
		       e.user_id::text
		FROM entries e JOIN settings st ON st.user_id = e.user_id
		WHERE e.remind IS NOT NULL AND e.reminded_at IS NULL AND e.start_time IS NOT NULL
		  AND ((e.date + e.start_time) AT TIME ZONE st.timezone - (e.remind || ' minutes')::interval) <= now()
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.ID, &e.Title, &e.Content, &e.Type, &e.LinkURL, &e.Date,
			&e.StartTime, &e.EndTime, &e.Completed, &e.Color, &e.Tags, &e.Recur, &e.Remind, &e.Pinned, &e.CreatedAt,
			&e.AccountID, &e.ExternalUID, &e.ExternalHref, &e.ExternalETag, &e.Dirty, &e.OwnerID); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *Store) MarkReminded(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `UPDATE entries SET reminded_at = now() WHERE id = $1`, id)
	return err
}

// ServerConfig reads/writes the singleton config map (VAPID keys, crypto key).
func (s *Store) ServerConfig(ctx context.Context, key string) (string, error) {
	var v string
	err := s.db.QueryRow(ctx, `SELECT value FROM server_config WHERE key = $1`, key).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return v, err
}

func (s *Store) SetServerConfig(ctx context.Context, key, value string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO server_config (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`, key, value)
	return err
}

// ---------- CalDAV ----------

type CaldavAccount struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	URL         string     `json:"url"`
	Username    string     `json:"username"`
	PasswordEnc string     `json:"-"`
	Color       string     `json:"color"`
	LastSynced  *time.Time `json:"lastSynced,omitempty"`
}

func (s *Store) CreateCaldavAccount(ctx context.Context, userID, name, url, username, passwordEnc, color string) (CaldavAccount, error) {
	var a CaldavAccount
	err := s.db.QueryRow(ctx, `
		INSERT INTO caldav_accounts (id, user_id, name, url, username, password_enc, color)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id::text, name, url, username, color
	`, uuid.NewString(), userID, name, url, username, passwordEnc, color).
		Scan(&a.ID, &a.Name, &a.URL, &a.Username, &a.Color)
	return a, err
}

func (s *Store) CaldavAccounts(ctx context.Context, userID string) ([]CaldavAccount, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, name, url, username, color, last_synced FROM caldav_accounts WHERE user_id = $1 ORDER BY created_at
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CaldavAccount{}
	for rows.Next() {
		var a CaldavAccount
		if err := rows.Scan(&a.ID, &a.Name, &a.URL, &a.Username, &a.Color, &a.LastSynced); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// CaldavAccountWithSecret returns the account incl. the encrypted password.
func (s *Store) CaldavAccountWithSecret(ctx context.Context, id string) (CaldavAccount, string, error) {
	var a CaldavAccount
	var userID string
	err := s.db.QueryRow(ctx, `
		SELECT id::text, user_id::text, name, url, username, password_enc, color FROM caldav_accounts WHERE id = $1
	`, id).Scan(&a.ID, &userID, &a.Name, &a.URL, &a.Username, &a.PasswordEnc, &a.Color)
	return a, userID, err
}

// AccountOwner pairs an account with its user for the background sync loop.
type AccountOwner struct {
	Account CaldavAccount
	UserID  string
}

func (s *Store) AllCaldavAccounts(ctx context.Context) ([]AccountOwner, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, user_id::text, name, url, username, password_enc, color FROM caldav_accounts
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AccountOwner
	for rows.Next() {
		var o AccountOwner
		if err := rows.Scan(&o.Account.ID, &o.UserID, &o.Account.Name, &o.Account.URL,
			&o.Account.Username, &o.Account.PasswordEnc, &o.Account.Color); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *Store) DeleteCaldavAccount(ctx context.Context, userID, id string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM caldav_accounts WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (s *Store) TouchCaldavSync(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `UPDATE caldav_accounts SET last_synced = now() WHERE id = $1`, id)
	return err
}

// CarddavAccount is a connected addressbook (birthdays sync).
type CarddavAccount struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	URL         string     `json:"url"`
	Username    string     `json:"username"`
	PasswordEnc string     `json:"-"`
	LastSynced  *time.Time `json:"lastSynced,omitempty"`
}

func (s *Store) CreateCarddavAccount(ctx context.Context, userID, name, url, username, passwordEnc string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO carddav_accounts (id, user_id, name, url, username, password_enc)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, uuid.NewString(), userID, name, url, username, passwordEnc)
	return err
}

func (s *Store) CarddavAccounts(ctx context.Context, userID string) ([]CarddavAccount, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, name, url, username, last_synced FROM carddav_accounts WHERE user_id = $1 ORDER BY created_at
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CarddavAccount{}
	for rows.Next() {
		var a CarddavAccount
		if err := rows.Scan(&a.ID, &a.Name, &a.URL, &a.Username, &a.LastSynced); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) CarddavAccountWithSecret(ctx context.Context, id string) (CarddavAccount, string, error) {
	var a CarddavAccount
	var userID string
	err := s.db.QueryRow(ctx, `
		SELECT id::text, user_id::text, name, url, username, password_enc FROM carddav_accounts WHERE id = $1
	`, id).Scan(&a.ID, &userID, &a.Name, &a.URL, &a.Username, &a.PasswordEnc)
	return a, userID, err
}

func (s *Store) DeleteCarddavAccount(ctx context.Context, userID, id string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM carddav_accounts WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (s *Store) TouchCarddavSync(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, `UPDATE carddav_accounts SET last_synced = now() WHERE id = $1`, id)
	return err
}

// GoogleToken holds one user's OAuth state; the matching events live as a
// feed's cached ICS so the existing feed pipeline renders them.
type GoogleToken struct {
	UserID        string
	FeedID        string
	RefreshEnc    string
	AccessEnc     string
	AccessExpires time.Time
	CalendarID    string
}

func (s *Store) UpsertGoogleToken(ctx context.Context, userID, feedID, refreshEnc, accessEnc string, expires time.Time) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO google_tokens (user_id, feed_id, refresh_enc, access_enc, access_expires)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
		  feed_id = $2, refresh_enc = $3, access_enc = $4, access_expires = $5
	`, userID, feedID, refreshEnc, accessEnc, expires)
	return err
}

func (s *Store) GoogleToken(ctx context.Context, userID string) (GoogleToken, error) {
	var t GoogleToken
	err := s.db.QueryRow(ctx, `
		SELECT user_id::text, coalesce(feed_id::text, ''), refresh_enc, access_enc, access_expires, calendar_id
		FROM google_tokens WHERE user_id = $1
	`, userID).Scan(&t.UserID, &t.FeedID, &t.RefreshEnc, &t.AccessEnc, &t.AccessExpires, &t.CalendarID)
	return t, err
}

func (s *Store) UpdateGoogleAccess(ctx context.Context, userID, accessEnc string, expires time.Time) error {
	_, err := s.db.Exec(ctx, `UPDATE google_tokens SET access_enc = $2, access_expires = $3, last_synced = now() WHERE user_id = $1`,
		userID, accessEnc, expires)
	return err
}

func (s *Store) DeleteGoogleToken(ctx context.Context, userID string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM google_tokens WHERE user_id = $1`, userID)
	return err
}

// AllGoogleTokens lists every connected account for the background sync loop.
func (s *Store) AllGoogleTokens(ctx context.Context) ([]GoogleToken, error) {
	rows, err := s.db.Query(ctx, `
		SELECT user_id::text, coalesce(feed_id::text, ''), refresh_enc, access_enc, access_expires, calendar_id
		FROM google_tokens
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []GoogleToken
	for rows.Next() {
		var t GoogleToken
		if err := rows.Scan(&t.UserID, &t.FeedID, &t.RefreshEnc, &t.AccessEnc, &t.AccessExpires, &t.CalendarID); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// AccountEntries returns all entries belonging to a CalDAV account.
func (s *Store) AccountEntries(ctx context.Context, userID, accountID string) ([]Entry, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+entryCols+` FROM entries WHERE user_id = $1 AND account_id = $2
	`, userID, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Entry{}
	for rows.Next() {
		var e Entry
		if err := e.scan(rows); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// UpsertSyncedEntry inserts or refreshes a pulled remote event. The caller
// supplies the full row; dirty stays false since the remote is authoritative.
func (s *Store) UpsertSyncedEntry(ctx context.Context, userID, accountID string, e Entry) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO entries (id, user_id, title, content, type, date, start_time, end_time, color, external_uid, external_href, external_etag, account_id)
		VALUES ($1, $2, $3, $4, 'event', $5, nullif($6, '')::time, nullif($7, '')::time, $8, $9, $10, $11, $12)
		ON CONFLICT (external_uid) WHERE external_uid IS NOT NULL DO UPDATE SET
			title = EXCLUDED.title, content = EXCLUDED.content, date = EXCLUDED.date,
			start_time = EXCLUDED.start_time, end_time = EXCLUDED.end_time,
			external_href = EXCLUDED.external_href, external_etag = EXCLUDED.external_etag,
			dirty = false
	`, e.ID, userID, e.Title, e.Content, e.Date, strOrEmpty(e.StartTime), strOrEmpty(e.EndTime), e.Color,
		e.ExternalUID, e.ExternalHref, e.ExternalETag, accountID)
	return err
}

// DeleteSyncedMissing removes synced entries whose UIDs disappeared remotely.
func (s *Store) DeleteSyncedMissing(ctx context.Context, userID, accountID string, keepUIDs []string) error {
	// Tombstone only entries that came from the server (not locally-created,
	// not-yet-pushed ones — those have an href but may just be pending).
	_, err := s.db.Exec(ctx, `
		DELETE FROM entries
		WHERE user_id = $1 AND account_id = $2 AND external_uid IS NOT NULL
		  AND external_uid != ALL($3) AND dirty = false
	`, userID, accountID, keepUIDs)
	return err
}

func (s *Store) ClearDirty(ctx context.Context, id, etag string) error {
	_, err := s.db.Exec(ctx, `UPDATE entries SET dirty = false, external_etag = coalesce(nullif($2, ''), external_etag) WHERE id = $1`, id, etag)
	return err
}

func (s *Store) SetExternalRef(ctx context.Context, id, uid, href, etag string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE entries SET external_uid = $2, external_href = $3, external_etag = $4, dirty = false WHERE id = $1
	`, id, uid, href, etag)
	return err
}

// Tombstone records a local delete that must propagate to the server.
func (s *Store) Tombstone(ctx context.Context, accountID, href string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO sync_tombstones (account_id, href) VALUES ($1, $2) ON CONFLICT DO NOTHING
	`, accountID, href)
	return err
}

func (s *Store) Tombstones(ctx context.Context, accountID string) ([]string, error) {
	rows, err := s.db.Query(ctx, `SELECT href FROM sync_tombstones WHERE account_id = $1`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *Store) ClearTombstone(ctx context.Context, accountID, href string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM sync_tombstones WHERE account_id = $1 AND href = $2`, accountID, href)
	return err
}

// EntryExternalRef returns the href needed to tombstone a synced entry.
func (s *Store) EntryExternalRef(ctx context.Context, userID, id string) (accountID, href string, ok bool, err error) {
	err = s.db.QueryRow(ctx, `
		SELECT account_id::text, external_href FROM entries WHERE id = $1 AND user_id = $2 AND external_href IS NOT NULL
	`, id, userID).Scan(&accountID, &href)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, nil
	}
	return accountID, href, err == nil, err
}

// ---------- Feeds ----------

func (s *Store) ListFeeds(ctx context.Context, userID string) ([]Feed, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, name, url, color, fetched_at FROM feeds WHERE user_id = $1 ORDER BY created_at
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	feeds := []Feed{}
	for rows.Next() {
		var f Feed
		if err := rows.Scan(&f.ID, &f.Name, &f.URL, &f.Color, &f.FetchedAt); err != nil {
			return nil, err
		}
		feeds = append(feeds, f)
	}
	return feeds, rows.Err()
}

func (s *Store) CreateFeed(ctx context.Context, userID, name, url, color, ics string) (Feed, error) {
	var f Feed
	err := s.db.QueryRow(ctx, `
		INSERT INTO feeds (id, user_id, name, url, color, ics_cache, fetched_at)
		VALUES ($1, $2, $3, $4, $5, $6, now())
		RETURNING id::text, name, url, color, fetched_at
	`, uuid.NewString(), userID, name, url, color, ics).Scan(&f.ID, &f.Name, &f.URL, &f.Color, &f.FetchedAt)
	return f, err
}

func (s *Store) DeleteFeed(ctx context.Context, userID, id string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM feeds WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// FeedCache returns the stored ICS body for every feed the user owns.
func (s *Store) FeedCaches(ctx context.Context, userID string) (map[Feed]string, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, name, url, color, fetched_at, ics_cache FROM feeds WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[Feed]string{}
	for rows.Next() {
		var f Feed
		var cache *string
		if err := rows.Scan(&f.ID, &f.Name, &f.URL, &f.Color, &f.FetchedAt, &cache); err != nil {
			return nil, err
		}
		if cache != nil {
			out[f] = *cache
		}
	}
	return out, rows.Err()
}

// AllFeeds returns every feed across users — used by the background sync loop.
func (s *Store) AllFeeds(ctx context.Context) (map[string]Feed, error) {
	rows, err := s.db.Query(ctx, `SELECT id::text, user_id::text, name, url, color, fetched_at FROM feeds`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]Feed{}
	var userID string
	for rows.Next() {
		var f Feed
		if err := rows.Scan(&f.ID, &userID, &f.Name, &f.URL, &f.Color, &f.FetchedAt); err != nil {
			return nil, err
		}
		out[userID+"/"+f.ID] = f
	}
	return out, rows.Err()
}

// RefreshFeedCache stores the latest ICS body for a feed.
func (s *Store) RefreshFeedCache(ctx context.Context, userID, feedID, ics string) error {
	_, err := s.db.Exec(ctx, `
		UPDATE feeds SET ics_cache = $3, fetched_at = now() WHERE id = $1 AND user_id = $2
	`, feedID, userID, ics)
	return err
}

// --- Files ---

type File struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`       // random name on disk
	OrigName   string    `json:"origName"`   // client filename for display
	Size       int64     `json:"size"`
	Mime       string    `json:"mime"`
	ShareToken *string   `json:"shareToken,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (s *Store) CreateFile(ctx context.Context, userID, name, origName, mime string, size int64) (File, error) {
	var f File
	err := s.db.QueryRow(ctx, `
		INSERT INTO files (user_id, name, orig_name, size, mime)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, name, orig_name, size, mime, share_token, created_at
	`, userID, name, origName, size, mime).
		Scan(&f.ID, &f.Name, &f.OrigName, &f.Size, &f.Mime, &f.ShareToken, &f.CreatedAt)
	return f, err
}

func (s *Store) ListFiles(ctx context.Context, userID string) ([]File, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, name, orig_name, size, mime, share_token, created_at
		FROM files WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []File{}
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.Name, &f.OrigName, &f.Size, &f.Mime, &f.ShareToken, &f.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *Store) FileByName(ctx context.Context, userID, name string) (File, error) {
	var f File
	err := s.db.QueryRow(ctx, `
		SELECT id::text, name, orig_name, size, mime, share_token, created_at
		FROM files WHERE user_id = $1 AND name = $2`, userID, name).
		Scan(&f.ID, &f.Name, &f.OrigName, &f.Size, &f.Mime, &f.ShareToken, &f.CreatedAt)
	return f, err
}

// FileByShareToken resolves a public share link — no ownership check.
func (s *Store) FileByShareToken(ctx context.Context, token string) (string, File, error) {
	var f File
	var userID string
	err := s.db.QueryRow(ctx, `
		SELECT user_id::text, id::text, name, orig_name, size, mime, share_token, created_at
		FROM files WHERE share_token = $1`, token).
		Scan(&userID, &f.ID, &f.Name, &f.OrigName, &f.Size, &f.Mime, &f.ShareToken, &f.CreatedAt)
	return userID, f, err
}

func (s *Store) SetFileShare(ctx context.Context, userID, id string, token *string) error {
	tag, err := s.db.Exec(ctx, `UPDATE files SET share_token = $3 WHERE id = $1 AND user_id = $2`, id, userID, token)
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// DeleteFile removes the row; the caller unlinks the disk file.
func (s *Store) DeleteFile(ctx context.Context, userID, id string) (string, error) {
	var name string
	err := s.db.QueryRow(ctx, `DELETE FROM files WHERE id = $1 AND user_id = $2 RETURNING name`, id, userID).Scan(&name)
	return name, err
}

// Activity returns date → entry count over the last N days.
func (s *Store) Activity(ctx context.Context, userID string, days int) (map[string]int, error) {
	rows, err := s.db.Query(ctx, `
		SELECT date::text, count(*) FROM entries
		WHERE user_id = $1 AND date >= current_date - $2::int
		GROUP BY date`, userID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var d string
		var n int
		if err := rows.Scan(&d, &n); err != nil {
			return nil, err
		}
		out[d] = n
	}
	return out, rows.Err()
}
