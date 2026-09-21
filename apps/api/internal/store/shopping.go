package store

// Shopping — shared-style shopping lists inspired by Koffan. A list owns
// ordered sections (Dairy, Produce, …) and items (name, quantity, note,
// checked / "can't find it" uncertain flag). Sections and items carry no
// user_id — ownership flows through the list.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type ShoppingList struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon"`
	Position  float64   `json:"position"`
	Items     int       `json:"items"`
	Open      int       `json:"open"` // unchecked items — what still needs buying
	CreatedAt time.Time `json:"createdAt"`
}

type ShoppingSection struct {
	ID       string  `json:"id"`
	ListID   string  `json:"listId"`
	Name     string  `json:"name"`
	Position float64 `json:"position"`
}

type ShoppingItem struct {
	ID        string     `json:"id"`
	ListID    string     `json:"listId"`
	SectionID *string    `json:"sectionId,omitempty"`
	Name      string     `json:"name"`
	Note      string     `json:"note"`
	Quantity  string     `json:"quantity"`
	Checked   bool       `json:"checked"`
	Uncertain bool       `json:"uncertain"`
	Position  float64    `json:"position"`
	CreatedAt time.Time  `json:"createdAt"`
	CheckedAt *time.Time `json:"checkedAt,omitempty"`
}

// ShoppingItemPatch — nil leaves the field alone; SectionID "" clears the
// section (item floats to the unsectioned block).
type ShoppingItemPatch struct {
	Name      *string  `json:"name"`
	Note      *string  `json:"note"`
	Quantity  *string  `json:"quantity"`
	SectionID *string  `json:"sectionId"`
	Checked   *bool    `json:"checked"`
	Uncertain *bool    `json:"uncertain"`
	Position  *float64 `json:"position"`
}

const listCols = `id::text, name, icon, position, created_at`
const sectionCols = `id::text, list_id::text, name, position`
const itemCols = `id::text, list_id::text, section_id::text, name, note,
	quantity, checked, uncertain, position, created_at, checked_at`

func (l *ShoppingList) scan(row interface{ Scan(...any) error }, items, open *int) error {
	return row.Scan(&l.ID, &l.Name, &l.Icon, &l.Position, &l.CreatedAt, items, open)
}

func (sc *ShoppingSection) scan(row interface{ Scan(...any) error }) error {
	return row.Scan(&sc.ID, &sc.ListID, &sc.Name, &sc.Position)
}

func (it *ShoppingItem) scan(row interface{ Scan(...any) error }) error {
	return row.Scan(&it.ID, &it.ListID, &it.SectionID, &it.Name, &it.Note,
		&it.Quantity, &it.Checked, &it.Uncertain, &it.Position, &it.CreatedAt, &it.CheckedAt)
}

// ---------- lists ----------

func (s *Store) ListShoppingLists(ctx context.Context, userID string) ([]ShoppingList, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+listCols+`,
			COALESCE(c.items, 0), COALESCE(c.open, 0)
		FROM shopping_lists l
		LEFT JOIN (
			SELECT list_id, count(*) AS items,
				count(*) FILTER (WHERE NOT checked) AS open
			FROM shopping_items GROUP BY list_id
		) c ON c.list_id = l.id
		WHERE l.user_id = $1
		ORDER BY l.position, l.created_at
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lists := []ShoppingList{}
	for rows.Next() {
		var l ShoppingList
		if err := l.scan(rows, &l.Items, &l.Open); err != nil {
			return nil, err
		}
		lists = append(lists, l)
	}
	return lists, rows.Err()
}

func (s *Store) ShoppingListOwned(ctx context.Context, userID, listID string) bool {
	var ok bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM shopping_lists WHERE id = $1 AND user_id = $2)`,
		listID, userID).Scan(&ok)
	return err == nil && ok
}

func (s *Store) CreateShoppingList(ctx context.Context, userID, name, icon string) (ShoppingList, error) {
	var l ShoppingList
	err := s.db.QueryRow(ctx, `
		INSERT INTO shopping_lists (user_id, name, icon, position)
		VALUES ($1, $2, $3,
			COALESCE((SELECT max(position) + 1024 FROM shopping_lists WHERE user_id = $1), 0))
		RETURNING `+listCols+`, 0, 0
	`, userID, name, icon).Scan(&l.ID, &l.Name, &l.Icon, &l.Position, &l.CreatedAt, &l.Items, &l.Open)
	return l, err
}

func (s *Store) UpdateShoppingList(ctx context.Context, userID, id, name, icon string) (ShoppingList, error) {
	var l ShoppingList
	err := s.db.QueryRow(ctx, `
		UPDATE shopping_lists SET name = $3, icon = $4
		WHERE id = $1 AND user_id = $2
		RETURNING `+listCols+`,
			(SELECT count(*) FROM shopping_items WHERE list_id = $1),
			(SELECT count(*) FROM shopping_items WHERE list_id = $1 AND NOT checked)
	`, id, userID, name, icon).Scan(&l.ID, &l.Name, &l.Icon, &l.Position, &l.CreatedAt, &l.Items, &l.Open)
	if errors.Is(err, pgx.ErrNoRows) {
		return ShoppingList{}, ErrNotFound
	}
	return l, err
}

func (s *Store) DeleteShoppingList(ctx context.Context, userID, id string) error {
	res, err := s.db.Exec(ctx, `DELETE FROM shopping_lists WHERE id = $1 AND user_id = $2`, id, userID)
	if err == nil && res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// ---------- sections ----------

func (s *Store) ListShoppingSections(ctx context.Context, listID string) ([]ShoppingSection, error) {
	rows, err := s.db.Query(ctx,
		`SELECT `+sectionCols+` FROM shopping_sections WHERE list_id = $1 ORDER BY position`, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sections := []ShoppingSection{}
	for rows.Next() {
		var sc ShoppingSection
		if err := sc.scan(rows); err != nil {
			return nil, err
		}
		sections = append(sections, sc)
	}
	return sections, rows.Err()
}

func (s *Store) CreateShoppingSection(ctx context.Context, listID, name string) (ShoppingSection, error) {
	var sc ShoppingSection
	err := s.db.QueryRow(ctx, `
		INSERT INTO shopping_sections (list_id, name, position)
		VALUES ($1, $2,
			COALESCE((SELECT max(position) + 1024 FROM shopping_sections WHERE list_id = $1), 0))
		RETURNING `+sectionCols, listID, name).Scan(&sc.ID, &sc.ListID, &sc.Name, &sc.Position)
	return sc, err
}

// ShoppingSectionList resolves which list a section belongs to — ownership
// checks then go through the list.
func (s *Store) ShoppingSectionList(ctx context.Context, sectionID string) (string, error) {
	var listID string
	err := s.db.QueryRow(ctx,
		`SELECT list_id::text FROM shopping_sections WHERE id = $1`, sectionID).Scan(&listID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return listID, err
}

func (s *Store) UpdateShoppingSection(ctx context.Context, listID, id, name string) error {
	res, err := s.db.Exec(ctx,
		`UPDATE shopping_sections SET name = $3 WHERE id = $1 AND list_id = $2`, id, listID, name)
	if err == nil && res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (s *Store) DeleteShoppingSection(ctx context.Context, listID, id string) error {
	res, err := s.db.Exec(ctx,
		`DELETE FROM shopping_sections WHERE id = $1 AND list_id = $2`, id, listID)
	if err == nil && res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// SetSectionChecked checks/unchecks every item in a section ("" sectionID =
// the unsectioned block).
func (s *Store) SetSectionChecked(ctx context.Context, listID string, sectionID *string, checked bool) error {
	_, err := s.db.Exec(ctx, `
		UPDATE shopping_items
		SET checked = $3, checked_at = CASE WHEN $3 THEN now() ELSE NULL END
		WHERE list_id = $1 AND section_id IS NOT DISTINCT FROM $2::uuid
	`, listID, sectionID, checked)
	return err
}

// ---------- items ----------

func (s *Store) ListShoppingItems(ctx context.Context, listID string) ([]ShoppingItem, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+itemCols+` FROM shopping_items
		WHERE list_id = $1
		ORDER BY checked, position, created_at
	`, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ShoppingItem{}
	for rows.Next() {
		var it ShoppingItem
		if err := it.scan(rows); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (s *Store) CreateShoppingItem(ctx context.Context, listID string, sectionID *string, name, note, quantity string) (ShoppingItem, error) {
	var it ShoppingItem
	err := s.db.QueryRow(ctx, `
		INSERT INTO shopping_items (list_id, section_id, name, note, quantity, position)
		VALUES ($1, $2, $3, $4, $5,
			COALESCE((SELECT max(position) + 1024 FROM shopping_items WHERE list_id = $1), 0))
		RETURNING `+itemCols,
		listID, sectionID, name, note, quantity).Scan(&it.ID, &it.ListID, &it.SectionID, &it.Name,
		&it.Note, &it.Quantity, &it.Checked, &it.Uncertain, &it.Position, &it.CreatedAt, &it.CheckedAt)
	return it, err
}

// ShoppingItemList resolves the owning list for an item.
func (s *Store) ShoppingItemList(ctx context.Context, itemID string) (string, error) {
	var listID string
	err := s.db.QueryRow(ctx,
		`SELECT list_id::text FROM shopping_items WHERE id = $1`, itemID).Scan(&listID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return listID, err
}

func (s *Store) UpdateShoppingItem(ctx context.Context, listID, id string, p ShoppingItemPatch) (ShoppingItem, error) {
	var it ShoppingItem
	// SectionID tri-state: nil = leave, "" = clear, uuid = move.
	err := s.db.QueryRow(ctx, `
		UPDATE shopping_items SET
			name      = COALESCE($3, name),
			note      = COALESCE($4, note),
			quantity  = COALESCE($5, quantity),
			section_id = CASE WHEN $6::text IS NULL THEN section_id
			                  WHEN $6 = '' THEN NULL ELSE $6::uuid END,
			checked   = COALESCE($7, checked),
			checked_at = CASE
				WHEN $7 IS NULL THEN checked_at
				WHEN $7 THEN now()
				ELSE NULL END,
			uncertain = COALESCE($8, uncertain),
			position  = COALESCE($9, position)
		WHERE id = $1 AND list_id = $2
		RETURNING `+itemCols,
		id, listID, p.Name, p.Note, p.Quantity, p.SectionID, p.Checked, p.Uncertain, p.Position,
	).Scan(&it.ID, &it.ListID, &it.SectionID, &it.Name, &it.Note,
		&it.Quantity, &it.Checked, &it.Uncertain, &it.Position, &it.CreatedAt, &it.CheckedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ShoppingItem{}, ErrNotFound
	}
	return it, err
}

func (s *Store) DeleteShoppingItem(ctx context.Context, listID, id string) error {
	res, err := s.db.Exec(ctx,
		`DELETE FROM shopping_items WHERE id = $1 AND list_id = $2`, id, listID)
	if err == nil && res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// ClearPurchased deletes checked items from a list; returns rows removed.
func (s *Store) ClearPurchased(ctx context.Context, listID string) (int, error) {
	res, err := s.db.Exec(ctx,
		`DELETE FROM shopping_items WHERE list_id = $1 AND checked`, listID)
	return int(res.RowsAffected()), err
}

// ---------- suggestions ----------

// ShoppingSuggestion is a previously-used item name plus the section it last
// lived in — powers the add-row autocomplete.
type ShoppingSuggestion struct {
	Name    string `json:"name"`
	Section string `json:"section,omitempty"`
}

func (s *Store) ShoppingSuggestions(ctx context.Context, userID, q string) ([]ShoppingSuggestion, error) {
	rows, err := s.db.Query(ctx, `
		SELECT name, section FROM (
			SELECT DISTINCT ON (lower(i.name)) i.name, s.name AS section
			FROM shopping_items i
			JOIN shopping_lists l ON l.id = i.list_id
			LEFT JOIN shopping_sections s ON s.id = i.section_id
			WHERE l.user_id = $1 AND ($2 = '' OR i.name ILIKE '%' || $2 || '%')
			ORDER BY lower(i.name), i.created_at DESC
		) t
		ORDER BY name
		LIMIT 10
	`, userID, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ShoppingSuggestion{}
	for rows.Next() {
		var sg ShoppingSuggestion
		var sec *string
		if err := rows.Scan(&sg.Name, &sec); err != nil {
			return nil, err
		}
		if sec != nil {
			sg.Section = *sec
		}
		out = append(out, sg)
	}
	return out, rows.Err()
}

// ---------- restore / preview ----------

// RestoreShoppingLists/RestoreShoppingSections/RestoreShoppingItems are
// additive — ON CONFLICT DO NOTHING; children only attach to lists the user
// owns (including ones this same restore just inserted).
func (s *Store) RestoreShoppingLists(ctx context.Context, userID string, lists []ShoppingList) (int, error) {
	n := 0
	for _, l := range lists {
		res, err := s.db.Exec(ctx, `
			INSERT INTO shopping_lists (id, user_id, name, icon, position, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO NOTHING
		`, l.ID, userID, l.Name, l.Icon, l.Position, l.CreatedAt)
		if err != nil {
			return n, err
		}
		n += int(res.RowsAffected())
	}
	return n, nil
}

func (s *Store) RestoreShoppingSections(ctx context.Context, userID string, sections []ShoppingSection) (int, error) {
	n := 0
	for _, sc := range sections {
		res, err := s.db.Exec(ctx, `
			INSERT INTO shopping_sections (id, list_id, name, position)
			SELECT $1, $2, $3, $4
			WHERE EXISTS (SELECT 1 FROM shopping_lists WHERE id = $2 AND user_id = $5)
			ON CONFLICT (id) DO NOTHING
		`, sc.ID, sc.ListID, sc.Name, sc.Position, userID)
		if err != nil {
			return n, err
		}
		n += int(res.RowsAffected())
	}
	return n, nil
}

func (s *Store) RestoreShoppingItems(ctx context.Context, userID string, items []ShoppingItem) (int, error) {
	n := 0
	for _, it := range items {
		res, err := s.db.Exec(ctx, `
			INSERT INTO shopping_items (id, list_id, section_id, name, note, quantity,
				checked, uncertain, position, created_at, checked_at)
			SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
			WHERE EXISTS (SELECT 1 FROM shopping_lists WHERE id = $2 AND user_id = $12)
			ON CONFLICT (id) DO NOTHING
		`, it.ID, it.ListID, it.SectionID, it.Name, it.Note, it.Quantity,
			it.Checked, it.Uncertain, it.Position, it.CreatedAt, it.CheckedAt, userID)
		if err != nil {
			return n, err
		}
		n += int(res.RowsAffected())
	}
	return n, nil
}

// ExistingShoppingIDs — preview existence checks for the three tables.
// Sections and items have no user_id; ownership goes through list_id.
func (s *Store) ExistingShoppingIDs(ctx context.Context, table, userID string, ids []string) (map[string]bool, error) {
	var query string
	switch table {
	case "shopping_lists":
		query = `SELECT id::text FROM shopping_lists WHERE user_id = $1 AND id::text = ANY($2::text[])`
	case "shopping_sections", "shopping_items":
		query = `SELECT id::text FROM ` + table + `
			WHERE list_id IN (SELECT id FROM shopping_lists WHERE user_id = $1)
			  AND id::text = ANY($2::text[])`
	default:
		return nil, fmt.Errorf("unknown table %q", table)
	}
	rows, err := s.db.Query(ctx, query, userID, ids)
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

// AllShoppingSections / AllShoppingItems — export fetches every child row
// for the user in one query (per-list listing is for the page).
func (s *Store) AllShoppingSections(ctx context.Context, userID string) ([]ShoppingSection, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+sectionCols+` FROM shopping_sections
		WHERE list_id IN (SELECT id FROM shopping_lists WHERE user_id = $1)
		ORDER BY position
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ShoppingSection{}
	for rows.Next() {
		var sc ShoppingSection
		if err := sc.scan(rows); err != nil {
			return nil, err
		}
		out = append(out, sc)
	}
	return out, rows.Err()
}

func (s *Store) AllShoppingItems(ctx context.Context, userID string) ([]ShoppingItem, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+itemCols+` FROM shopping_items
		WHERE list_id IN (SELECT id FROM shopping_lists WHERE user_id = $1)
		ORDER BY position, created_at
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ShoppingItem{}
	for rows.Next() {
		var it ShoppingItem
		if err := it.scan(rows); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// OwnedShoppingListIDs returns the subset of list IDs the user owns —
// preview counts child rows whose list isn't in the export as orphaned.
func (s *Store) OwnedShoppingListIDs(ctx context.Context, userID string, listIDs []string) (map[string]bool, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text FROM shopping_lists
		WHERE user_id = $1 AND id::text = ANY($2::text[])
	`, userID, listIDs)
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
