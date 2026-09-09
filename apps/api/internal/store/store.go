package store

import (
	"context"
	"errors"
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

type Entry struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content,omitempty"`
	Type      string    `json:"type"`
	LinkURL   string    `json:"linkUrl,omitempty"`
	Date      string    `json:"date"`
	Completed bool      `json:"completed"`
	Color     string    `json:"color"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"createdAt"`
}

type Settings struct {
	Country      string `json:"country"`
	ShowHolidays bool   `json:"showHolidays"`
	Theme        string `json:"theme"`
}

type EntryInput struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Type    string   `json:"type"`
	LinkURL string   `json:"linkUrl"`
	Date    string   `json:"date"`
	Color   string   `json:"color"`
	Tags    []string `json:"tags"`
}

type EntryPatch struct {
	Title     *string  `json:"title"`
	Content   *string  `json:"content"`
	Type      *string  `json:"type"`
	LinkURL   *string  `json:"linkUrl"`
	Date      *string  `json:"date"`
	Completed *bool    `json:"completed"`
	Color     *string  `json:"color"`
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
	if from == "" {
		from = time.Now().AddDate(0, -1, 0).Format(time.DateOnly)
	}
	if to == "" {
		to = time.Now().AddDate(0, 1, 0).Format(time.DateOnly)
	}
	query := `
		SELECT id::text, title, content, type, link_url, date::text, completed, color, tags, created_at
		FROM entries
		WHERE user_id = $1 AND date >= $2 AND date <= $3
	`
	args := []any{userID, from, to}
	if strings.TrimSpace(q) != "" {
		query += ` AND (title ILIKE $4 OR content ILIKE $4 OR $4 = ANY(tags))`
		args = append(args, "%"+strings.TrimSpace(q)+"%")
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
		if err := rows.Scan(&e.ID, &e.Title, &e.Content, &e.Type, &e.LinkURL, &e.Date, &e.Completed, &e.Color, &e.Tags, &e.CreatedAt); err != nil {
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
		INSERT INTO entries (user_id, title, content, type, link_url, date, color, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text, title, content, type, link_url, date::text, completed, color, tags, created_at
	`, userID, input.Title, input.Content, input.Type, input.LinkURL, input.Date, input.Color, input.Tags).
		Scan(&e.ID, &e.Title, &e.Content, &e.Type, &e.LinkURL, &e.Date, &e.Completed, &e.Color, &e.Tags, &e.CreatedAt)
	return e, err
}

func (s *Store) UpdateEntry(ctx context.Context, userID, id string, patch EntryPatch) (Entry, error) {
	current, err := s.entry(ctx, userID, id)
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
	err = s.db.QueryRow(ctx, `
		UPDATE entries
		SET title = $1, content = $2, type = $3, link_url = $4, date = $5,
		    completed = $6, color = $7, tags = $8
		WHERE id = $9 AND user_id = $10
		RETURNING id::text, title, content, type, link_url, date::text, completed, color, tags, created_at
	`, current.Title, current.Content, current.Type, current.LinkURL, current.Date, current.Completed, current.Color, current.Tags, id, userID).
		Scan(&e.ID, &e.Title, &e.Content, &e.Type, &e.LinkURL, &e.Date, &e.Completed, &e.Color, &e.Tags, &e.CreatedAt)
	return e, err
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

func (s *Store) entry(ctx context.Context, userID, id string) (Entry, error) {
	var e Entry
	err := s.db.QueryRow(ctx, `
		SELECT id::text, title, content, type, link_url, date::text, completed, color, tags, created_at
		FROM entries
		WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(&e.ID, &e.Title, &e.Content, &e.Type, &e.LinkURL, &e.Date, &e.Completed, &e.Color, &e.Tags, &e.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Entry{}, ErrNotFound
	}
	return e, err
}

func (s *Store) Settings(ctx context.Context, userID string) (Settings, error) {
	var settings Settings
	err := s.db.QueryRow(ctx, `
		SELECT country, show_holidays, theme FROM settings WHERE user_id = $1
	`, userID).Scan(&settings.Country, &settings.ShowHolidays, &settings.Theme)
	return settings, err
}

func (s *Store) UpdateSettings(ctx context.Context, userID string, settings Settings) (Settings, error) {
	var out Settings
	err := s.db.QueryRow(ctx, `
		INSERT INTO settings (user_id, country, show_holidays, theme)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE
		SET country = EXCLUDED.country,
		    show_holidays = EXCLUDED.show_holidays,
		    theme = EXCLUDED.theme
		RETURNING country, show_holidays, theme
	`, userID, settings.Country, settings.ShowHolidays, settings.Theme).Scan(&out.Country, &out.ShowHolidays, &out.Theme)
	return out, err
}
