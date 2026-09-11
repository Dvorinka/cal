-- +goose Up
-- People: a privacy-first personal relationship manager — names, relations,
-- birthdays and named yearly dates (anniversaries, namedays). NULL
-- workspace_id is the implicit "Personal" space, same as entries/files.
CREATE TABLE IF NOT EXISTS people (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL,
  name         TEXT NOT NULL,
  relation     TEXT NOT NULL DEFAULT '',
  birthday     DATE,
  dates        JSONB NOT NULL DEFAULT '[]',
  notes        TEXT NOT NULL DEFAULT '',
  color        TEXT NOT NULL DEFAULT 'slate',
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS people_user ON people(user_id, created_at);
-- +goose Down
DROP TABLE IF EXISTS people;
