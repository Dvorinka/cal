-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE settings (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  country TEXT NOT NULL DEFAULT 'US',
  show_holidays BOOLEAN NOT NULL DEFAULT true,
  theme TEXT NOT NULL DEFAULT 'system',
  CONSTRAINT settings_theme_check CHECK (theme IN ('light', 'dark', 'system'))
);

CREATE TABLE entries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  content TEXT NOT NULL DEFAULT '',
  type TEXT NOT NULL,
  link_url TEXT NOT NULL DEFAULT '',
  date DATE NOT NULL,
  completed BOOLEAN NOT NULL DEFAULT false,
  color TEXT NOT NULL DEFAULT 'slate',
  tags TEXT[] NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT entries_type_check CHECK (type IN ('task', 'note', 'link'))
);

CREATE INDEX entries_user_date_idx ON entries (user_id, date);
CREATE INDEX entries_user_title_idx ON entries USING GIN (to_tsvector('simple', title || ' ' || content));
CREATE INDEX sessions_user_expires_idx ON sessions (user_id, expires_at);

-- +goose Down
DROP TABLE IF EXISTS entries;
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
