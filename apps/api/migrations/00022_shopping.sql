-- +goose Up
CREATE TABLE shopping_lists (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  icon TEXT NOT NULL DEFAULT '',
  position DOUBLE PRECISION NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_shopping_lists_user ON shopping_lists (user_id);

CREATE TABLE shopping_sections (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  list_id UUID NOT NULL REFERENCES shopping_lists(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  position DOUBLE PRECISION NOT NULL DEFAULT 0
);
CREATE INDEX idx_shopping_sections_list ON shopping_sections (list_id);

CREATE TABLE shopping_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  list_id UUID NOT NULL REFERENCES shopping_lists(id) ON DELETE CASCADE,
  section_id UUID REFERENCES shopping_sections(id) ON DELETE SET NULL,
  name TEXT NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  quantity TEXT NOT NULL DEFAULT '',
  checked BOOLEAN NOT NULL DEFAULT false,
  uncertain BOOLEAN NOT NULL DEFAULT false,
  position DOUBLE PRECISION NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  checked_at TIMESTAMPTZ
);
CREATE INDEX idx_shopping_items_list ON shopping_items (list_id);
CREATE INDEX idx_shopping_items_section ON shopping_items (section_id);

-- +goose Down
DROP TABLE shopping_items;
DROP TABLE shopping_sections;
DROP TABLE shopping_lists;
