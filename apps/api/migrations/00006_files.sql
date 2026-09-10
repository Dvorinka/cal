-- +goose Up
CREATE TABLE files (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name        TEXT NOT NULL,          -- server-side random name on disk
  orig_name   TEXT NOT NULL,
  size        BIGINT NOT NULL,
  mime        TEXT NOT NULL DEFAULT 'application/octet-stream',
  share_token TEXT UNIQUE,            -- NULL until shared
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE entries ADD COLUMN pinned BOOLEAN NOT NULL DEFAULT false;
CREATE INDEX files_user ON files(user_id, created_at DESC);
-- +goose Down
ALTER TABLE entries DROP COLUMN IF EXISTS pinned;
DROP TABLE IF EXISTS files;
