-- YouTube search proxy: base URL of the user's Invidious instance
-- ("" = feature off). User-configured like a CalDAV URL, not a secret.
-- +goose Up
ALTER TABLE settings ADD COLUMN IF NOT EXISTS invidious_url TEXT NOT NULL DEFAULT '';
-- +goose Down
ALTER TABLE settings DROP COLUMN IF EXISTS invidious_url;
