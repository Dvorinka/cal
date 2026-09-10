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
-- +goose Down
DROP TABLE IF EXISTS time_entries;
DROP TABLE IF EXISTS card_activity;
ALTER TABLE entries DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE boards DROP COLUMN IF EXISTS description, DROP COLUMN IF EXISTS target_date, DROP COLUMN IF EXISTS share_token;
