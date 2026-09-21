-- Enrichment status on link entries: NULL = pending, timestamp = attempted
-- (fields may still be empty when unfurl failed — the UI stops waiting either way).
-- +goose Up
ALTER TABLE entries ADD COLUMN IF NOT EXISTS link_meta_at TIMESTAMPTZ;
-- +goose Down
ALTER TABLE entries DROP COLUMN IF EXISTS link_meta_at;
