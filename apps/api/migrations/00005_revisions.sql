-- +goose Up
-- +goose StatementBegin
CREATE TABLE entry_revisions (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  entry_id   UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title      TEXT NOT NULL,
  content    TEXT,
  type       TEXT NOT NULL,
  link_url   TEXT,
  date       DATE NOT NULL,
  start_time TIME,
  end_time   TIME,
  completed  BOOLEAN NOT NULL,
  color      TEXT,
  tags       TEXT[],
  recur      TEXT,
  remind     INTEGER,
  saved_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX entry_revisions_entry ON entry_revisions(entry_id, saved_at DESC);
ALTER TABLE settings ADD COLUMN timezone TEXT NOT NULL DEFAULT 'UTC';
ALTER TABLE sessions
  ADD COLUMN user_agent TEXT,
  ADD COLUMN last_seen_at TIMESTAMPTZ;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS entry_revisions;
-- +goose StatementEnd
