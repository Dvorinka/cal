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
	completed, color, tags, recur, created_at`

type Entry struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content,omitempty"`
	Type      string    `json:"type"`
	LinkURL   string    `json:"linkUrl,omitempty"`
	Date      string    `json:"date"`
	StartTime *string   `json:"startTime,omitempty"`
	EndTime   *string   `json:"endTime,omitempty"`
	Completed bool      `json:"completed"`
	Color     string    `json:"color"`
	Tags      []string  `json:"tags"`
	Recur     string    `json:"recur"`
	CreatedAt time.Time `json:"createdAt"`
}

func (e *Entry) scan(row interface{ Scan(...any) error }) error {
	return row.Scan(&e.ID, &e.Title, &e.Content, &e.Type, &e.LinkURL, &e.Date,
		&e.StartTime, &e.EndTime, &e.Completed, &e.Color, &e.Tags, &e.Recur, &e.CreatedAt)
}

type Settings struct {
	Country      string `json:"country"`
	ShowHolidays bool   `json:"showHolidays"`
	Theme        string `json:"theme"`
	WeekStart    string `json:"weekStart"`
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
	Color     *string  `json:"color"`
	Recur     *string  `json:"recur"`
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

func (s *Store) CreateSession(ctx context.Context, userID string) (string, error) {
	id := uuid.NewString()
	_, err := s.db.Exec(ctx, `
		INSERT INTO sessions (id, user_id, expires_at)
		VALUES ($1, $2, now() + interval '30 days')
	`, id, userID)
	return id, err
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
		INSERT INTO entries (user_id, title, content, type, link_url, date, start_time, end_time, color, tags, recur)
		VALUES ($1, $2, $3, $4, $5, $6, nullif($7, '')::time, nullif($8, '')::time, $9, $10, coalesce(nullif($11, ''), 'none'))
		RETURNING `+entryCols+`
	`, userID, input.Title, input.Content, input.Type, input.LinkURL, input.Date, input.StartTime, input.EndTime, input.Color, input.Tags, input.Recur).
		Scan(&e.ID, &e.Title, &e.Content, &e.Type, &e.LinkURL, &e.Date, &e.StartTime, &e.EndTime, &e.Completed, &e.Color, &e.Tags, &e.Recur, &e.CreatedAt)
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
	if patch.Color != nil {
		current.Color = *patch.Color
	}
	if patch.HasTags {
		current.Tags = patch.Tags
	}

	var e Entry
	err = tx.QueryRow(ctx, `
		UPDATE entries
		SET title = $1, content = $2, type = $3, link_url = $4, date = $5,
		    start_time = nullif($6, '')::time, end_time = nullif($7, '')::time,
		    completed = $8, color = $9, tags = $10, recur = $11
		WHERE id = $12 AND user_id = $13
		RETURNING `+entryCols+`
	`, current.Title, current.Content, current.Type, current.LinkURL, current.Date,
		strOrEmpty(current.StartTime), strOrEmpty(current.EndTime),
		current.Completed, current.Color, current.Tags, current.Recur, id, userID).
		Scan(&e.ID, &e.Title, &e.Content, &e.Type, &e.LinkURL, &e.Date, &e.StartTime, &e.EndTime, &e.Completed, &e.Color, &e.Tags, &e.Recur, &e.CreatedAt)
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
		SELECT country, show_holidays, theme, week_start FROM settings WHERE user_id = $1
	`, userID).Scan(&settings.Country, &settings.ShowHolidays, &settings.Theme, &settings.WeekStart)
	return settings, err
}

func (s *Store) UpdateSettings(ctx context.Context, userID string, settings Settings) (Settings, error) {
	var out Settings
	err := s.db.QueryRow(ctx, `
		INSERT INTO settings (user_id, country, show_holidays, theme, week_start)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE
		SET country = EXCLUDED.country,
		    show_holidays = EXCLUDED.show_holidays,
		    theme = EXCLUDED.theme,
		    week_start = EXCLUDED.week_start
		RETURNING country, show_holidays, theme, week_start
	`, userID, settings.Country, settings.ShowHolidays, settings.Theme, settings.WeekStart).
		Scan(&out.Country, &out.ShowHolidays, &out.Theme, &out.WeekStart)
	return out, err
}
