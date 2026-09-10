-- +goose Up
ALTER TABLE entries
  ADD COLUMN start_time TIME,
  ADD COLUMN end_time TIME,
  ADD COLUMN recur TEXT NOT NULL DEFAULT 'none',
  ADD CONSTRAINT entries_recur_check CHECK (recur IN ('none', 'daily', 'weekly', 'monthly', 'yearly')),
  ADD CONSTRAINT entries_time_order_check CHECK (start_time IS NULL OR end_time IS NULL OR end_time > start_time);

ALTER TABLE settings
  ADD COLUMN week_start TEXT NOT NULL DEFAULT 'monday',
  ADD CONSTRAINT settings_week_start_check CHECK (week_start IN ('monday', 'sunday'));

-- +goose Down
ALTER TABLE settings DROP COLUMN week_start;
ALTER TABLE entries DROP COLUMN recur;
ALTER TABLE entries DROP COLUMN end_time;
ALTER TABLE entries DROP COLUMN start_time;
