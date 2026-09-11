-- +goose Up
-- Feeds gain a kind: "calendar" items render on the calendar as before,
-- "links" items become real link entries (YouTube channel feeds, blogs
-- you want as bookmarks) with thumbnails.
ALTER TABLE feeds ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'calendar';

-- +goose Down
ALTER TABLE feeds DROP COLUMN IF EXISTS kind;
