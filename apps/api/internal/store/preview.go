package store

// Dry-run support for POST /api/restore?dry=1 — existence lookups only, the
// restore path itself stays untouched.
//
// The table argument is whitelisted inside ExistingIDs; callers never pass
// user input, so the identifier can't be injected.

import (
	"context"
	"fmt"
)

// ExistingIDs returns the subset of ids already present for the user.
// Table is one of entries | people | person_timeline | files.
func (s *Store) ExistingIDs(ctx context.Context, table, userID string, ids []string) (map[string]bool, error) {
	switch table {
	case "entries", "people", "person_timeline", "files":
	default:
		return nil, fmt.Errorf("unknown table %q", table)
	}
	rows, err := s.db.Query(ctx,
		`SELECT id::text FROM `+table+` WHERE user_id = $1 AND id::text = ANY($2::text[])`,
		userID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// ExistingPersonLinks counts edges already stored — restore dedupes on
// (from_person_id, to_person_id, kind).
func (s *Store) ExistingPersonLinks(ctx context.Context, userID string, links []PersonRelation) (int, error) {
	froms := make([]string, 0, len(links))
	tos := make([]string, 0, len(links))
	kinds := make([]string, 0, len(links))
	for _, l := range links {
		froms = append(froms, l.PersonID)
		tos = append(tos, l.OtherID)
		kinds = append(kinds, l.Kind)
	}
	var n int
	err := s.db.QueryRow(ctx, `
		SELECT count(*) FROM person_links pl
		JOIN (SELECT * FROM unnest($2::uuid[], $3::uuid[], $4::text[])) AS t(f, t, k)
		  ON pl.from_person_id = t.f AND pl.to_person_id = t.t AND pl.kind = t.k
		WHERE pl.user_id = $1`, userID, froms, tos, kinds).Scan(&n)
	return n, err
}

// ExistingFileHashes returns the sha256s already stored for this user —
// restore can satisfy those rows by hardlink instead of an archive binary.
func (s *Store) ExistingFileHashes(ctx context.Context, userID string, hashes []string) (map[string]bool, error) {
	rows, err := s.db.Query(ctx,
		`SELECT sha256 FROM files WHERE user_id = $1 AND sha256 = ANY($2::text[])`,
		userID, hashes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, err
		}
		out[h] = true
	}
	return out, rows.Err()
}

// ExistingFileNames returns the stored names already taken by this user.
func (s *Store) ExistingFileNames(ctx context.Context, userID string, names []string) (map[string]bool, error) {
	rows, err := s.db.Query(ctx,
		`SELECT name FROM files WHERE user_id = $1 AND name = ANY($2::text[])`,
		userID, names)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out[n] = true
	}
	return out, rows.Err()
}
