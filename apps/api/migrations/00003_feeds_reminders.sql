-- +goose Up
-- External calendar feeds, entry reminders, widget token, accent color.
CREATE TABLE feeds (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  url TEXT NOT NULL,
  color TEXT NOT NULL DEFAULT 'slate',
  ics_cache TEXT,
  fetched_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE entries
  DROP CONSTRAINT entries_type_check,
  ADD COLUMN remind INTEGER,
  ADD CONSTRAINT entries_type_check CHECK (type IN ('task', 'note', 'link', 'event')),
  ADD CONSTRAINT entries_remind_check CHECK (remind IS NULL OR (remind >= 0 AND remind <= 10080));

ALTER TABLE settings
  ADD COLUMN widget_token TEXT NOT NULL DEFAULT replace(gen_random_uuid()::text, '-', ''),
  ADD COLUMN api_token TEXT NOT NULL DEFAULT replace(gen_random_uuid()::text, '-', ''),
  ADD COLUMN accent TEXT NOT NULL DEFAULT 'green',
  ADD CONSTRAINT settings_accent_check CHECK (accent IN ('green', 'blue', 'violet', 'amber', 'rose'));

-- +goose Down
ALTER TABLE settings DROP COLUMN accent, DROP COLUMN widget_token, DROP COLUMN api_token;
ALTER TABLE entries
  DROP CONSTRAINT entries_remind_check,
  DROP CONSTRAINT entries_type_check,
  DROP COLUMN remind,
  ADD CONSTRAINT entries_type_check CHECK (type IN ('task', 'note', 'link'));
DROP TABLE feeds;
