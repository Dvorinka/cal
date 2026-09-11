package store

// People — the personal relationship manager: names, relations, birthdays
// and named yearly dates (anniversaries, namedays). workspace_id follows the
// entries/files model: NULL is the implicit "Personal" space.

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// PersonDate is one named yearly date on a person — an anniversary, a
// nameday, "first met". The year is informational (lets the client show
// "turns N"); recurrence is month+day.
type PersonDate struct {
	Label string `json:"label"`
	Date  string `json:"date"` // YYYY-MM-DD
}

// personCols is the canonical people column list.
const personCols = `id::text, name, relation, birthday::text, dates, notes, color, workspace_id::text, created_at`

type Person struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Relation    string       `json:"relation"`
	Birthday    *string      `json:"birthday,omitempty"`
	Dates       []PersonDate `json:"dates"`
	Notes       string       `json:"notes"`
	Color       string       `json:"color"`
	WorkspaceID *string      `json:"workspaceId,omitempty"`
	CreatedAt   time.Time    `json:"createdAt"`
}

func (p *Person) scan(row interface{ Scan(...any) error }) error {
	if err := row.Scan(&p.ID, &p.Name, &p.Relation, &p.Birthday, &p.Dates, &p.Notes, &p.Color, &p.WorkspaceID, &p.CreatedAt); err != nil {
		return err
	}
	if p.Dates == nil {
		p.Dates = []PersonDate{}
	}
	return nil
}

// PersonInput is the whole editable record — the dialog always sends every
// field, so update is a plain replace (birthday "" / workspaceId ""|null
// clear their columns).
type PersonInput struct {
	Name        string       `json:"name"`
	Relation    string       `json:"relation"`
	Birthday    string       `json:"birthday"`
	Dates       []PersonDate `json:"dates"`
	Notes       string       `json:"notes"`
	Color       string       `json:"color"`
	WorkspaceID *string      `json:"workspaceId"`
}

func (s *Store) ListPeople(ctx context.Context, userID string) ([]Person, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+personCols+`
		FROM people WHERE user_id = $1 ORDER BY lower(name), created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Person{}
	for rows.Next() {
		var p Person
		if err := p.scan(rows); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) CreatePerson(ctx context.Context, userID string, in PersonInput) (Person, error) {
	if in.Color == "" {
		in.Color = "slate"
	}
	var p Person
	err := p.scan(s.db.QueryRow(ctx, `
		INSERT INTO people (user_id, workspace_id, name, relation, birthday, dates, notes, color)
		VALUES ($1, nullif($2, '')::uuid, $3, $4, nullif($5, '')::date, coalesce($6::jsonb, '[]'::jsonb), $7, $8)
		RETURNING `+personCols,
		userID, strOr(in.WorkspaceID), in.Name, in.Relation, in.Birthday, in.Dates, in.Notes, in.Color))
	return p, err
}

func (s *Store) UpdatePerson(ctx context.Context, userID, id string, in PersonInput) (Person, error) {
	if in.Color == "" {
		in.Color = "slate"
	}
	var p Person
	err := p.scan(s.db.QueryRow(ctx, `
		UPDATE people SET
			name = $3, relation = $4, birthday = nullif($5, '')::date,
			dates = coalesce($6::jsonb, '[]'::jsonb), notes = $7, color = $8,
			workspace_id = nullif($9, '')::uuid
		WHERE id = $1 AND user_id = $2
		RETURNING `+personCols,
		id, userID, in.Name, in.Relation, in.Birthday, in.Dates, in.Notes, in.Color, strOr(in.WorkspaceID)))
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	return p, err
}

func (s *Store) DeletePerson(ctx context.Context, userID, id string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM people WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
