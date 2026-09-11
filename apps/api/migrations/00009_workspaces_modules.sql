-- +goose Up
-- Workspaces: lightweight scopes over existing content. NULL workspace_id is
-- the implicit "Personal" space — no backfill, no NOT NULL pain.
CREATE TABLE workspaces (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name       TEXT NOT NULL,
  color      TEXT NOT NULL DEFAULT 'green',
  icon       TEXT NOT NULL DEFAULT 'folder',
  position   INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX workspaces_user ON workspaces(user_id, position);

ALTER TABLE entries ADD COLUMN workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL;
ALTER TABLE files   ADD COLUMN workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL;
ALTER TABLE boards  ADD COLUMN workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL;
CREATE INDEX entries_workspace ON entries(user_id, workspace_id) WHERE workspace_id IS NOT NULL;

-- Feature modules: JSONB map name→bool. Empty/absent key = enabled.
ALTER TABLE settings
  ADD COLUMN modules JSONB NOT NULL DEFAULT '{}',
  ADD COLUMN active_workspace UUID REFERENCES workspaces(id) ON DELETE SET NULL;

-- Time entries get their own tags (separate from the linked task's tags).
ALTER TABLE time_entries ADD COLUMN tags TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE time_entries ADD COLUMN workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL;

-- File tagging.
ALTER TABLE files ADD COLUMN tags TEXT[] NOT NULL DEFAULT '{}';

-- Entry dependencies: a card blocked by another can't complete until it does.
ALTER TABLE entries ADD COLUMN blocked_by UUID REFERENCES entries(id) ON DELETE SET NULL;

-- Saved filters / smart views.
CREATE TABLE saved_filters (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name       TEXT NOT NULL,
  filter     JSONB NOT NULL DEFAULT '{}',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX saved_filters_user ON saved_filters(user_id, created_at);

-- Mail accounts (IMAP/SMTP). Password encrypted like CalDAV credentials.
CREATE TABLE mail_accounts (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name         TEXT NOT NULL DEFAULT '',
  email        TEXT NOT NULL,
  imap_host    TEXT NOT NULL,
  imap_port    INT NOT NULL DEFAULT 993,
  smtp_host    TEXT NOT NULL,
  smtp_port    INT NOT NULL DEFAULT 465,
  username     TEXT NOT NULL,
  password_enc TEXT NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX mail_accounts_user ON mail_accounts(user_id, created_at);
-- +goose Down
DROP TABLE IF EXISTS mail_accounts;
DROP TABLE IF EXISTS saved_filters;
ALTER TABLE entries DROP COLUMN IF EXISTS blocked_by, DROP COLUMN IF EXISTS workspace_id;
ALTER TABLE files DROP COLUMN IF EXISTS workspace_id, DROP COLUMN IF EXISTS tags;
ALTER TABLE boards DROP COLUMN IF EXISTS workspace_id;
ALTER TABLE time_entries DROP COLUMN IF EXISTS tags, DROP COLUMN IF EXISTS workspace_id;
ALTER TABLE settings DROP COLUMN IF EXISTS modules, DROP COLUMN IF EXISTS active_workspace;
DROP TABLE IF EXISTS workspaces;
