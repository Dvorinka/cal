package store

// People — the personal relationship manager: names, relations, birthdays
// and named yearly dates (anniversaries, namedays), plus profile depth
// (contact fields, tags, links, custom fields), person→person relationships,
// a per-person timeline, and file attachments. workspace_id follows the
// entries/files model: NULL is the implicit "Personal" space.

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// PersonDate is one named yearly date on a person — an anniversary, a
// nameday, "first met". The year is informational (lets the client show
// "turns N"); recurrence is month+day. RemindDays optionally fires a push
// reminder that many days before each occurrence.
type PersonDate struct {
	Label      string `json:"label"`
	Date       string `json:"date"` // YYYY-MM-DD
	RemindDays *int   `json:"remindDays,omitempty"`
}

// PersonField is one custom key/value pair (PV's custom_fields table).
type PersonField struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// PersonLink is one social/web link (PV's social_links table).
type PersonLink struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
}

// personCols is the canonical people column list.
const personCols = `id::text, name, relation, birthday::text, dates, notes, color, workspace_id::text, created_at,
	nickname, avatar, phone, email, address, gift_ideas, interests, is_favorite, fields, links, tags, birthday_remind`

type Person struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Relation       string        `json:"relation"`
	Birthday       *string       `json:"birthday,omitempty"`
	Dates          []PersonDate  `json:"dates"`
	Notes          string        `json:"notes"`
	Color          string        `json:"color"`
	WorkspaceID    *string       `json:"workspaceId,omitempty"`
	CreatedAt      time.Time     `json:"createdAt"`
	Nickname       string        `json:"nickname,omitempty"`
	Avatar         string        `json:"avatar,omitempty"` // files.name of the avatar image
	Phone          string        `json:"phone,omitempty"`
	Email          string        `json:"email,omitempty"`
	Address        string        `json:"address,omitempty"`
	GiftIdeas      string        `json:"giftIdeas,omitempty"`
	Interests      string        `json:"interests,omitempty"`
	IsFavorite     bool          `json:"isFavorite"`
	Fields         []PersonField `json:"fields"`
	Links          []PersonLink  `json:"links"`
	Tags           []string      `json:"tags"`
	BirthdayRemind *int          `json:"birthdayRemind,omitempty"` // days ahead to push a reminder
}

func (p *Person) scan(row interface{ Scan(...any) error }) error {
	if err := row.Scan(&p.ID, &p.Name, &p.Relation, &p.Birthday, &p.Dates, &p.Notes, &p.Color, &p.WorkspaceID, &p.CreatedAt,
		&p.Nickname, &p.Avatar, &p.Phone, &p.Email, &p.Address, &p.GiftIdeas, &p.Interests, &p.IsFavorite, &p.Fields, &p.Links, &p.Tags, &p.BirthdayRemind); err != nil {
		return err
	}
	if p.Dates == nil {
		p.Dates = []PersonDate{}
	}
	if p.Fields == nil {
		p.Fields = []PersonField{}
	}
	if p.Links == nil {
		p.Links = []PersonLink{}
	}
	if p.Tags == nil {
		p.Tags = []string{}
	}
	return nil
}

// PersonInput is the whole editable record — the dialog always sends every
// field, so update is a plain replace (birthday "" / workspaceId ""|null
// clear their columns).
type PersonInput struct {
	Name           string        `json:"name"`
	Relation       string        `json:"relation"`
	Birthday       string        `json:"birthday"`
	Dates          []PersonDate  `json:"dates"`
	Notes          string        `json:"notes"`
	Color          string        `json:"color"`
	WorkspaceID    *string       `json:"workspaceId"`
	Nickname       string        `json:"nickname"`
	Avatar         string        `json:"avatar"`
	Phone          string        `json:"phone"`
	Email          string        `json:"email"`
	Address        string        `json:"address"`
	GiftIdeas      string        `json:"giftIdeas"`
	Interests      string        `json:"interests"`
	IsFavorite     bool          `json:"isFavorite"`
	Fields         []PersonField `json:"fields"`
	Links          []PersonLink  `json:"links"`
	Tags           []string      `json:"tags"`
	BirthdayRemind *int          `json:"birthdayRemind"`
}

func (s *Store) ListPeople(ctx context.Context, userID string) ([]Person, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+personCols+`
		FROM people WHERE user_id = $1
		ORDER BY is_favorite DESC, lower(name), created_at`, userID)
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

// GetPerson returns one owned person.
func (s *Store) GetPerson(ctx context.Context, userID, id string) (Person, error) {
	var p Person
	err := p.scan(s.db.QueryRow(ctx, `
		SELECT `+personCols+` FROM people WHERE id = $1 AND user_id = $2`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	return p, err
}

func (s *Store) CreatePerson(ctx context.Context, userID string, in PersonInput) (Person, error) {
	if in.Color == "" {
		in.Color = "slate"
	}
	var p Person
	err := p.scan(s.db.QueryRow(ctx, `
		INSERT INTO people (user_id, workspace_id, name, relation, birthday, dates, notes, color,
			nickname, avatar, phone, email, address, gift_ideas, interests, is_favorite, fields, links, tags, birthday_remind)
		VALUES ($1, nullif($2, '')::uuid, $3, $4, nullif($5, '')::date, coalesce($6::jsonb, '[]'::jsonb), $7, $8,
			$9, $10, $11, $12, $13, $14, $15, $16, coalesce($17::jsonb, '[]'::jsonb), coalesce($18::jsonb, '[]'::jsonb), coalesce($19::text[], '{}'::text[]), $20)
		RETURNING `+personCols,
		userID, strOr(in.WorkspaceID), in.Name, in.Relation, in.Birthday, in.Dates, in.Notes, in.Color,
		in.Nickname, in.Avatar, in.Phone, in.Email, in.Address, in.GiftIdeas, in.Interests, in.IsFavorite, in.Fields, in.Links, in.Tags, in.BirthdayRemind))
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
			workspace_id = nullif($9, '')::uuid,
			nickname = $10, avatar = $11, phone = $12, email = $13, address = $14,
			gift_ideas = $15, interests = $16, is_favorite = $17,
			fields = coalesce($18::jsonb, '[]'::jsonb), links = coalesce($19::jsonb, '[]'::jsonb),
			tags = coalesce($20::text[], '{}'::text[]), birthday_remind = $21
		WHERE id = $1 AND user_id = $2
		RETURNING `+personCols,
		id, userID, in.Name, in.Relation, in.Birthday, in.Dates, in.Notes, in.Color, strOr(in.WorkspaceID),
		in.Nickname, in.Avatar, in.Phone, in.Email, in.Address,
		in.GiftIdeas, in.Interests, in.IsFavorite, in.Fields, in.Links, in.Tags, in.BirthdayRemind))
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

// SearchPeople matches name, nickname, relation, notes, tags and custom-field
// values for the global search (pg_trgm index covers name+nickname).
func (s *Store) SearchPeople(ctx context.Context, userID, q string) ([]Person, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+personCols+` FROM people
		WHERE user_id = $1
		  AND (name ILIKE $2 OR nickname ILIKE $2 OR relation ILIKE $2 OR notes ILIKE $2
		       OR gift_ideas ILIKE $2 OR interests ILIKE $2 OR $3 = ANY(tags))
		ORDER BY is_favorite DESC, lower(name) LIMIT 20`, userID, "%"+q+"%", q)
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

// ---------- Relationships (person_links) ----------

// PersonRelation is one directed edge from→to with the other person's name
// resolved for display. Outgoing marks the direction relative to PersonID.
type PersonRelation struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	PersonID  string    `json:"personId"` // the person this row was fetched for
	OtherID   string    `json:"otherId"`  // the person on the far end
	OtherName string    `json:"otherName"`
	Outgoing  bool      `json:"outgoing"` // true: PersonID is from_person; false: it's to_person
	CreatedAt time.Time `json:"createdAt"`
}

// PersonRelations returns every link touching a person, both directions.
func (s *Store) PersonRelations(ctx context.Context, userID, personID string) ([]PersonRelation, error) {
	rows, err := s.db.Query(ctx, `
		SELECT l.id::text, l.kind, l.from_person_id::text, l.to_person_id::text,
		       fp.name, tp.name, l.created_at
		FROM person_links l
		JOIN people fp ON fp.id = l.from_person_id
		JOIN people tp ON tp.id = l.to_person_id
		WHERE l.user_id = $1 AND (l.from_person_id = $2 OR l.to_person_id = $2)
		ORDER BY l.created_at`, userID, personID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PersonRelation{}
	for rows.Next() {
		var r PersonRelation
		var fromID, toID, fromName, toName string
		if err := rows.Scan(&r.ID, &r.Kind, &fromID, &toID, &fromName, &toName, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.PersonID = personID
		if fromID == personID {
			r.OtherID, r.OtherName, r.Outgoing = toID, toName, true
		} else {
			r.OtherID, r.OtherName, r.Outgoing = fromID, fromName, false
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// AllPersonRelations — every link in the account (family tree render).
func (s *Store) AllPersonRelations(ctx context.Context, userID string) ([]PersonRelation, error) {
	rows, err := s.db.Query(ctx, `
		SELECT l.id::text, l.kind, l.from_person_id::text, l.to_person_id::text, l.created_at
		FROM person_links l WHERE l.user_id = $1 ORDER BY l.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PersonRelation{}
	for rows.Next() {
		var r PersonRelation
		if err := rows.Scan(&r.ID, &r.Kind, &r.PersonID, &r.OtherID, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Outgoing = true // raw edge direction; PersonID/OtherID carry from/to
		out = append(out, r)
	}
	return out, rows.Err()
}

// LinkPersons inserts a directed edge. kind is caller-validated.
func (s *Store) LinkPersons(ctx context.Context, userID, fromID, toID, kind string) (PersonRelation, error) {
	var r PersonRelation
	err := s.db.QueryRow(ctx, `
		INSERT INTO person_links (user_id, from_person_id, to_person_id, kind)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, kind, from_person_id::text, to_person_id::text, created_at`,
		userID, fromID, toID, kind).
		Scan(&r.ID, &r.Kind, &r.PersonID, &r.OtherID, &r.CreatedAt)
	if err != nil {
		return r, err
	}
	r.PersonID, r.Outgoing = fromID, true
	r.OtherID = toID
	return r, nil
}

func (s *Store) UnlinkPersons(ctx context.Context, userID, linkID string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM person_links WHERE id = $1 AND user_id = $2`, linkID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------- Timeline ----------

type TimelineItem struct {
	ID         string    `json:"id"`
	PersonID   string    `json:"personId"`
	Type       string    `json:"type"` // met | gift | trip | achievement | memory | note
	Title      string    `json:"title"`
	Body       string    `json:"body,omitempty"`
	OccurredOn *string   `json:"occurredOn,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (s *Store) PersonTimeline(ctx context.Context, userID, personID string) ([]TimelineItem, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, person_id::text, type, title, body, occurred_on::text, created_at
		FROM person_timeline
		WHERE user_id = $1 AND person_id = $2
		ORDER BY occurred_on DESC NULLS LAST, created_at DESC`, userID, personID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TimelineItem{}
	for rows.Next() {
		var t TimelineItem
		if err := rows.Scan(&t.ID, &t.PersonID, &t.Type, &t.Title, &t.Body, &t.OccurredOn, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) CreateTimelineItem(ctx context.Context, userID, personID, typ, title, body, occurredOn string) (TimelineItem, error) {
	var t TimelineItem
	err := s.db.QueryRow(ctx, `
		INSERT INTO person_timeline (user_id, person_id, type, title, body, occurred_on)
		VALUES ($1, $2, $3, $4, $5, nullif($6, '')::date)
		RETURNING id::text, person_id::text, type, title, body, occurred_on::text, created_at`,
		userID, personID, typ, title, body, occurredOn).
		Scan(&t.ID, &t.PersonID, &t.Type, &t.Title, &t.Body, &t.OccurredOn, &t.CreatedAt)
	return t, err
}

func (s *Store) UpdateTimelineItem(ctx context.Context, userID, id, typ, title, body, occurredOn string) (TimelineItem, error) {
	var t TimelineItem
	err := s.db.QueryRow(ctx, `
		UPDATE person_timeline SET type = $3, title = $4, body = $5, occurred_on = nullif($6, '')::date
		WHERE id = $1 AND user_id = $2
		RETURNING id::text, person_id::text, type, title, body, occurred_on::text, created_at`,
		id, userID, typ, title, body, occurredOn).
		Scan(&t.ID, &t.PersonID, &t.Type, &t.Title, &t.Body, &t.OccurredOn, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrNotFound
	}
	return t, err
}

func (s *Store) DeleteTimelineItem(ctx context.Context, userID, id string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM person_timeline WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------- Export / restore ----------

// orNow returns nil for a zero time so coalesce falls through to now().
func orNow(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// AllTimeline — every timeline item in the account (export/backup).
func (s *Store) AllTimeline(ctx context.Context, userID string) ([]TimelineItem, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id::text, person_id::text, type, title, body, occurred_on::text, created_at
		FROM person_timeline WHERE user_id = $1
		ORDER BY person_id, occurred_on DESC NULLS LAST, created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TimelineItem{}
	for rows.Next() {
		var t TimelineItem
		if err := rows.Scan(&t.ID, &t.PersonID, &t.Type, &t.Title, &t.Body, &t.OccurredOn, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// RestorePeople inserts exported people additively — existing IDs are
// skipped, same contract as RestoreEntries. Returns the count inserted.
func (s *Store) RestorePeople(ctx context.Context, userID string, people []Person) (int, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	imported := 0
	for _, p := range people {
		if p.ID == "" || p.Name == "" {
			continue
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO people (id, user_id, workspace_id, name, relation, birthday, dates, notes, color, created_at,
				nickname, avatar, phone, email, address, gift_ideas, interests, is_favorite, fields, links, tags, birthday_remind)
			VALUES ($1::uuid, $2, nullif($3, '')::uuid, $4, $5, nullif($6, '')::date,
				coalesce($7::jsonb, '[]'::jsonb), $8, $9, coalesce($10, now()),
				$11, $12, $13, $14, $15, $16, $17, $18,
				coalesce($19::jsonb, '[]'::jsonb), coalesce($20::jsonb, '[]'::jsonb), coalesce($21::text[], '{}'::text[]), $22)
			ON CONFLICT (id) DO NOTHING`,
			p.ID, userID, strOr(p.WorkspaceID), p.Name, p.Relation, strOr(p.Birthday),
			p.Dates, p.Notes, p.Color, orNow(p.CreatedAt),
			p.Nickname, p.Avatar, p.Phone, p.Email, p.Address, p.GiftIdeas, p.Interests, p.IsFavorite,
			p.Fields, p.Links, p.Tags, p.BirthdayRemind)
		if err != nil {
			return imported, err
		}
		imported += int(tag.RowsAffected())
	}
	return imported, tx.Commit(ctx)
}

// RestorePersonLinks re-inserts edges — both endpoints must exist locally
// (restored or already present), otherwise the edge is skipped.
func (s *Store) RestorePersonLinks(ctx context.Context, userID string, links []PersonRelation) (int, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	imported := 0
	for _, l := range links {
		if l.Kind == "" || l.PersonID == "" || l.OtherID == "" || l.PersonID == l.OtherID {
			continue
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO person_links (user_id, from_person_id, to_person_id, kind, created_at)
			SELECT $1, $2::uuid, $3::uuid, $4, coalesce($5, now())
			WHERE EXISTS (SELECT 1 FROM people WHERE id = $2::uuid AND user_id = $1)
			  AND EXISTS (SELECT 1 FROM people WHERE id = $3::uuid AND user_id = $1)
			ON CONFLICT (from_person_id, to_person_id, kind) DO NOTHING`,
			userID, l.PersonID, l.OtherID, l.Kind, orNow(l.CreatedAt))
		if err != nil {
			return imported, err
		}
		imported += int(tag.RowsAffected())
	}
	return imported, tx.Commit(ctx)
}

// RestorePersonTimeline re-inserts items whose person exists locally.
func (s *Store) RestorePersonTimeline(ctx context.Context, userID string, items []TimelineItem) (int, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	imported := 0
	for _, t := range items {
		if t.ID == "" || t.PersonID == "" || t.Title == "" {
			continue
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO person_timeline (id, user_id, person_id, type, title, body, occurred_on, created_at)
			SELECT $1::uuid, $2, $3::uuid, $4, $5, $6, nullif($7, '')::date, coalesce($8, now())
			WHERE EXISTS (SELECT 1 FROM people WHERE id = $3::uuid AND user_id = $2)
			ON CONFLICT (id) DO NOTHING`,
			t.ID, userID, t.PersonID, t.Type, t.Title, t.Body, strOr(t.OccurredOn), orNow(t.CreatedAt))
		if err != nil {
			return imported, err
		}
		imported += int(tag.RowsAffected())
	}
	return imported, tx.Commit(ctx)
}

// ---------- Person date reminders ----------

// PersonReminder is one due person-date reminder resolved against the
// user's local "today" — the push loop turns it into a notification.
type PersonReminder struct {
	UserID    string
	PersonID  string
	Name      string
	Label     string // "birthday" or the date's label
	DateKey   string // dedupe key per date entry
	DaysUntil int
	Year      int // occurrence year (dedupe key part)
}

// PersonRemindersDue finds every birthday/dates[] occurrence whose remind
// window covers the user's local today (0 <= daysUntil <= remindDays) and
// hasn't been sent for this occurrence year. Occurrence expansion is the
// same month/day rule the client uses (Feb 29 → Feb 28 off-leap).
func (s *Store) PersonRemindersDue(ctx context.Context) ([]PersonReminder, error) {
	rows, err := s.db.Query(ctx, `
		SELECT p.user_id::text, `+personCols+`, coalesce(st.timezone, 'UTC')
		FROM people p JOIN settings st ON st.user_id = p.user_id
		WHERE p.birthday_remind IS NOT NULL
		   OR jsonb_path_exists(p.dates, '$[*].remindDays')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type cand struct {
		uid string
		p   Person
		tz  string
	}
	var people []cand
	for rows.Next() {
		var c cand
		var p Person
		if err := rows.Scan(&c.uid, &p.ID, &p.Name, &p.Relation, &p.Birthday, &p.Dates, &p.Notes, &p.Color, &p.WorkspaceID, &p.CreatedAt,
			&p.Nickname, &p.Avatar, &p.Phone, &p.Email, &p.Address, &p.GiftIdeas, &p.Interests, &p.IsFavorite, &p.Fields, &p.Links, &p.Tags, &p.BirthdayRemind, &c.tz); err != nil {
			return nil, err
		}
		c.p = p
		people = append(people, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	locs := map[string]*time.Location{}
	var out []PersonReminder
	for _, c := range people {
		loc, ok := locs[c.tz]
		if !ok {
			l, err := time.LoadLocation(c.tz)
			if err != nil {
				l = time.UTC
			}
			loc = l
			locs[c.tz] = l
		}
		now := time.Now().In(loc)
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

		type md struct {
			label, key string
			date       string
			remind     int
		}
		var dates []md
		if c.p.Birthday != nil && c.p.BirthdayRemind != nil && *c.p.BirthdayRemind > 0 {
			dates = append(dates, md{"birthday", "birthday", *c.p.Birthday, *c.p.BirthdayRemind})
		}
		for _, dt := range c.p.Dates {
			if dt.RemindDays != nil && *dt.RemindDays > 0 {
				dates = append(dates, md{dt.Label, dt.Label + "|" + dt.Date, dt.Date, *dt.RemindDays})
			}
		}
		for _, d := range dates {
			month, day, ok := parseMonthDay(d.date)
			if !ok {
				continue
			}
			// Next occurrence: this year's date, else next year's.
			occ := occurrenceDate(today.Year(), month, day, loc)
			if occ.Before(today) {
				occ = occurrenceDate(today.Year()+1, month, day, loc)
			}
			daysUntil := int(occ.Sub(today).Hours() / 24)
			if daysUntil < 0 || daysUntil > d.remind {
				continue
			}
			sent, err := s.PersonReminderSent(ctx, c.p.ID, d.key, occ.Year())
			if err != nil || sent {
				continue
			}
			out = append(out, PersonReminder{
				UserID: c.uid, PersonID: c.p.ID, Name: c.p.Name, Label: d.label,
				DateKey: d.key, DaysUntil: daysUntil, Year: occ.Year(),
			})
		}
	}
	return out, nil
}

// parseMonthDay extracts month+day from a YYYY-MM-DD string.
func parseMonthDay(date string) (time.Month, int, bool) {
	t, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return 0, 0, false
	}
	return t.Month(), t.Day(), true
}

// occurrenceDate resolves a yearly (month, day) onto year — Feb 29 falls
// back to Feb 28 on non-leap years, matching the client rule.
func occurrenceDate(year int, month time.Month, day int, loc *time.Location) time.Time {
	d := time.Date(year, month, day, 0, 0, 0, 0, loc)
	if d.Month() != month {
		return time.Date(year, month, day-1, 0, 0, 0, 0, loc)
	}
	return d
}

func (s *Store) PersonReminderSent(ctx context.Context, personID, dateKey string, year int) (bool, error) {
	var one int
	err := s.db.QueryRow(ctx, `
		SELECT 1 FROM person_reminder_log WHERE person_id = $1 AND date_key = $2 AND year = $3`,
		personID, dateKey, year).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) MarkPersonReminderSent(ctx context.Context, personID, dateKey string, year int) {
	_, _ = s.db.Exec(ctx, `
		INSERT INTO person_reminder_log (person_id, date_key, year) VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING`, personID, dateKey, year)
}

// SetPeopleShare turns the public people list on/off; returns the token.
func (s *Store) SetPeopleShare(ctx context.Context, userID string, on bool) (string, error) {
	token := ""
	if on {
		token = newToken(20)
	}
	tag, err := s.db.Exec(ctx, `UPDATE settings SET people_share_token = nullif($2, '') WHERE user_id = $1`, userID, token)
	if tag.RowsAffected() == 0 {
		return "", ErrNotFound
	}
	return token, err
}

// SharedPeopleOwner resolves a people share token → owner user id.
func (s *Store) SharedPeopleOwner(ctx context.Context, token string) (string, error) {
	var userID string
	err := s.db.QueryRow(ctx,
		`SELECT user_id::text FROM settings WHERE people_share_token = $1`, token).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return userID, err
}
