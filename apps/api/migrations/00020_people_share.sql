-- Public "birthday list" page: one share token per user exposes names,
-- birthdays and recurring dates at /api/shared/people/<token> (+ .ics).
-- +goose Up
ALTER TABLE settings ADD COLUMN IF NOT EXISTS people_share_token TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS settings_people_share ON settings(people_share_token) WHERE people_share_token IS NOT NULL;
-- +goose Down
DROP INDEX IF EXISTS settings_people_share;
ALTER TABLE settings DROP COLUMN IF EXISTS people_share_token;
