-- +goose Up
-- Phase 9 — PeopleVault merge: profile depth, tags, relationships, timeline,
-- person attachments, nameday country, date reminders, password resets.
ALTER TABLE people ADD COLUMN IF NOT EXISTS nickname    TEXT NOT NULL DEFAULT '';
ALTER TABLE people ADD COLUMN IF NOT EXISTS avatar      TEXT NOT NULL DEFAULT ''; -- files.name of the avatar image
ALTER TABLE people ADD COLUMN IF NOT EXISTS phone       TEXT NOT NULL DEFAULT '';
ALTER TABLE people ADD COLUMN IF NOT EXISTS email       TEXT NOT NULL DEFAULT '';
ALTER TABLE people ADD COLUMN IF NOT EXISTS address     TEXT NOT NULL DEFAULT '';
ALTER TABLE people ADD COLUMN IF NOT EXISTS gift_ideas  TEXT NOT NULL DEFAULT '';
ALTER TABLE people ADD COLUMN IF NOT EXISTS interests   TEXT NOT NULL DEFAULT '';
ALTER TABLE people ADD COLUMN IF NOT EXISTS is_favorite BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE people ADD COLUMN IF NOT EXISTS fields      JSONB NOT NULL DEFAULT '[]'; -- [{key,value}] custom fields
ALTER TABLE people ADD COLUMN IF NOT EXISTS links       JSONB NOT NULL DEFAULT '[]'; -- [{platform,url}] social links
ALTER TABLE people ADD COLUMN IF NOT EXISTS tags        TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE people ADD COLUMN IF NOT EXISTS birthday_remind INT; -- days ahead to remind about the birthday (NULL = off)

CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS people_name_trgm ON people USING gin (name gin_trgm_ops, nickname gin_trgm_ops);

-- Relationships: directed person→person edges (family-tree foundation).
CREATE TABLE IF NOT EXISTS person_links (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  from_person_id UUID NOT NULL REFERENCES people(id) ON DELETE CASCADE,
  to_person_id   UUID NOT NULL REFERENCES people(id) ON DELETE CASCADE,
  kind           TEXT NOT NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (from_person_id <> to_person_id),
  UNIQUE (from_person_id, to_person_id, kind)
);
CREATE INDEX IF NOT EXISTS person_links_user ON person_links(user_id);
CREATE INDEX IF NOT EXISTS person_links_to   ON person_links(to_person_id);

-- Timeline: a chronological history per person (met, gift, trip, memory…).
CREATE TABLE IF NOT EXISTS person_timeline (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  person_id   UUID NOT NULL REFERENCES people(id) ON DELETE CASCADE,
  type        TEXT NOT NULL DEFAULT 'note',
  title       TEXT NOT NULL,
  body        TEXT NOT NULL DEFAULT '',
  occurred_on DATE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS person_timeline_person ON person_timeline(person_id, occurred_on DESC NULLS LAST, created_at DESC);

-- Person attachments ride the files pipeline (quota, share tokens, icons).
ALTER TABLE files ADD COLUMN IF NOT EXISTS person_id UUID REFERENCES people(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS files_person ON files(person_id) WHERE person_id IS NOT NULL;

-- Nameday lookup preference (two-letter country code; '' = no default).
ALTER TABLE settings ADD COLUMN IF NOT EXISTS nameday_country TEXT NOT NULL DEFAULT '';

-- Date reminders: a sent-log so "Mum's birthday in 7 days" pushes once per
-- year per date, evaluated by the existing push loop.
CREATE TABLE IF NOT EXISTS person_reminder_log (
  person_id UUID NOT NULL REFERENCES people(id) ON DELETE CASCADE,
  date_key  TEXT NOT NULL, -- stable key for the date entry, e.g. "birthday" or the dates[] label+date
  year      INT  NOT NULL,
  sent_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (person_id, date_key, year)
);

-- Password reset: single-use tokens sent via the Mail module's SMTP account.
CREATE TABLE IF NOT EXISTS password_resets (
  user_id    UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS password_resets;
DROP TABLE IF EXISTS person_reminder_log;
ALTER TABLE settings DROP COLUMN IF EXISTS nameday_country;
ALTER TABLE files DROP COLUMN IF EXISTS person_id;
DROP TABLE IF EXISTS person_timeline;
DROP TABLE IF EXISTS person_links;
DROP INDEX IF EXISTS people_name_trgm;
ALTER TABLE people
  DROP COLUMN IF EXISTS nickname, DROP COLUMN IF EXISTS avatar,
  DROP COLUMN IF EXISTS phone, DROP COLUMN IF EXISTS email,
  DROP COLUMN IF EXISTS address, DROP COLUMN IF EXISTS gift_ideas,
  DROP COLUMN IF EXISTS interests, DROP COLUMN IF EXISTS is_favorite,
  DROP COLUMN IF EXISTS fields, DROP COLUMN IF EXISTS links,
  DROP COLUMN IF EXISTS tags, DROP COLUMN IF EXISTS birthday_remind;
