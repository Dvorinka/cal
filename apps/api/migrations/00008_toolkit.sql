-- +goose Up
CREATE TABLE time_entries (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  entry_id   UUID REFERENCES entries(id) ON DELETE CASCADE,
  start_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  end_at     TIMESTAMPTZ,
  note       TEXT NOT NULL DEFAULT ''
);
CREATE INDEX time_entries_user ON time_entries(user_id, start_at DESC);
CREATE UNIQUE INDEX time_entries_open ON time_entries(user_id) WHERE end_at IS NULL;

CREATE TABLE card_activity (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  entry_id   UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action     TEXT NOT NULL,
  detail     TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX card_activity_entry ON card_activity(entry_id, created_at DESC);

ALTER TABLE entries ADD COLUMN deleted_at TIMESTAMPTZ;
CREATE INDEX entries_active ON entries(user_id) WHERE deleted_at IS NULL;

ALTER TABLE boards
  ADD COLUMN description TEXT NOT NULL DEFAULT '',
  ADD COLUMN target_date DATE,
  ADD COLUMN share_token TEXT UNIQUE;
ALTER TABLE time_entries ADD COLUMN planned_minutes INT;
ALTER TABLE settings ADD COLUMN digest_time TIME, ADD COLUMN digest_last DATE;
ALTER TABLE time_entries
  ADD COLUMN billable BOOLEAN DEFAULT false,
  ADD COLUMN hourly_rate NUMERIC(10,2),
  ADD COLUMN project_id UUID REFERENCES boards(id) ON DELETE SET NULL;
ALTER TABLE settings ADD COLUMN default_rate NUMERIC(10,2);
CREATE INDEX time_entries_project ON time_entries (project_id) WHERE end_at IS NOT NULL;
ALTER TABLE entries
  ADD COLUMN link_image TEXT,
  ADD COLUMN link_desc TEXT,
  ADD COLUMN link_favicon TEXT,
  ADD COLUMN watched BOOLEAN DEFAULT false,
  ADD COLUMN link_video_id TEXT;
-- +goose Down
ALTER TABLE time_entries DROP COLUMN IF EXISTS planned_minutes,
  DROP COLUMN IF EXISTS billable, DROP COLUMN IF EXISTS hourly_rate,
  DROP COLUMN IF EXISTS project_id;
ALTER TABLE settings DROP COLUMN IF EXISTS digest_time, DROP COLUMN IF EXISTS digest_last,
  DROP COLUMN IF EXISTS default_rate;
DROP TABLE IF EXISTS time_entries;
DROP TABLE IF EXISTS card_activity;
ALTER TABLE entries DROP COLUMN IF EXISTS deleted_at,
  DROP COLUMN IF EXISTS link_image, DROP COLUMN IF EXISTS link_desc,
  DROP COLUMN IF EXISTS link_favicon, DROP COLUMN IF EXISTS watched,
  DROP COLUMN IF EXISTS link_video_id;
ALTER TABLE boards DROP COLUMN IF EXISTS description, DROP COLUMN IF EXISTS target_date, DROP COLUMN IF EXISTS share_token;
