-- +goose Up
ALTER TABLE settings ADD COLUMN IF NOT EXISTS default_view text NOT NULL DEFAULT 'month';

-- +goose Down
ALTER TABLE settings DROP COLUMN IF EXISTS default_view;
