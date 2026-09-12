-- settings.github_token (GitHub PAT) was referenced by the store queries but
-- never created by a migration — fresh installs 500'd on GET /api/settings.
-- +goose Up
ALTER TABLE settings ADD COLUMN IF NOT EXISTS github_token text;
-- +goose Down
ALTER TABLE settings DROP COLUMN IF EXISTS github_token;
