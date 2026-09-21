-- Content hash for upload dedup + zero-copy restore (hardlink instead of
-- unpacking an identical binary). NULL for rows predating the column.
-- +goose Up
ALTER TABLE files ADD COLUMN IF NOT EXISTS sha256 TEXT;
CREATE INDEX IF NOT EXISTS files_user_sha256 ON files(user_id, sha256) WHERE sha256 IS NOT NULL;
-- +goose Down
DROP INDEX IF EXISTS files_user_sha256;
ALTER TABLE files DROP COLUMN IF EXISTS sha256;
