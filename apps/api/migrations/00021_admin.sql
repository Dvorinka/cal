-- +goose Up
ALTER TABLE users ADD COLUMN is_admin BOOLEAN NOT NULL DEFAULT false;
-- Bootstrap: the earliest account administers the instance.
UPDATE users SET is_admin = true
  WHERE id = (SELECT id FROM users ORDER BY created_at, id LIMIT 1);

-- +goose Down
ALTER TABLE users DROP COLUMN is_admin;
