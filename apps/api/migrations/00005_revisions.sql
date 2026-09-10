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
ALTER TABLE settings ADD COLUMN city TEXT;
ALTER TABLE settings ADD COLUMN quota_mb INTEGER NOT NULL DEFAULT 500;
ALTER TABLE sessions
  ADD COLUMN user_agent TEXT,
  ADD COLUMN last_seen_at TIMESTAMPTZ;
CREATE TABLE webhooks (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  url        TEXT NOT NULL,
  secret     TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE carddav_accounts (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name         TEXT NOT NULL,
  url          TEXT NOT NULL,
  username     TEXT NOT NULL,
  password_enc TEXT NOT NULL,
  last_synced  TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE google_tokens (
  user_id          UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  feed_id          UUID,
  refresh_enc      TEXT NOT NULL,
  access_enc       TEXT NOT NULL,
  access_expires   TIMESTAMPTZ NOT NULL,
  calendar_id      TEXT NOT NULL DEFAULT 'primary',
  last_synced      TIMESTAMPTZ
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS google_tokens;
DROP TABLE IF EXISTS carddav_accounts;
DROP TABLE IF EXISTS webhooks;
DROP TABLE IF EXISTS entry_revisions;
ALTER TABLE sessions DROP COLUMN IF EXISTS user_agent, DROP COLUMN IF EXISTS last_seen_at;
ALTER TABLE settings DROP COLUMN IF EXISTS city, DROP COLUMN IF EXISTS timezone, DROP COLUMN IF EXISTS quota_mb;
-- +goose StatementEnd
