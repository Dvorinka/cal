-- Feeds gain a kind: "calendar" items render on the calendar as before,
-- "links" items become real link entries (YouTube channel feeds, blogs
-- you want as bookmarks) with thumbnails.
ALTER TABLE feeds ADD COLUMN kind TEXT NOT NULL DEFAULT 'calendar';
