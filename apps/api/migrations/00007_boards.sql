-- +goose Up
CREATE TABLE boards (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name       TEXT NOT NULL,
  color      TEXT NOT NULL DEFAULT 'slate',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE board_columns (
  id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  board_id UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
  name     TEXT NOT NULL,
  position INT  NOT NULL
);
CREATE INDEX board_columns_board ON board_columns(board_id, position);
ALTER TABLE entries
  ADD COLUMN board_id  UUID REFERENCES boards(id) ON DELETE SET NULL,
  ADD COLUMN column_id UUID REFERENCES board_columns(id) ON DELETE SET NULL,
  ADD COLUMN position  FLOAT;
CREATE INDEX entries_board ON entries(board_id, column_id, position) WHERE board_id IS NOT NULL;
-- +goose Down
ALTER TABLE entries DROP COLUMN IF EXISTS board_id, DROP COLUMN IF EXISTS column_id, DROP COLUMN IF EXISTS position;
DROP TABLE IF EXISTS board_columns;
DROP TABLE IF EXISTS boards;
