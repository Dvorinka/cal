-- +goose Up
-- Write-tier board sharing: share_edit lets the public link move cards
-- between columns. View-only remains the default.
ALTER TABLE boards ADD COLUMN IF NOT EXISTS share_edit BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE boards DROP COLUMN IF EXISTS share_edit;
