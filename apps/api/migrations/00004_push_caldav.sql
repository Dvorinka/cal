-- +goose Up
-- Web push subscriptions, server-wide config (VAPID keys), CalDAV accounts,
-- and sync bookkeeping.

CREATE TABLE server_config (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE push_subscriptions (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  endpoint TEXT NOT NULL UNIQUE,
  p256dh TEXT NOT NULL,
  auth TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE caldav_accounts (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  url TEXT NOT NULL,
  username TEXT NOT NULL,
  password_enc TEXT NOT NULL,
  color TEXT NOT NULL DEFAULT 'sky',
  last_synced TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE entries
  ADD COLUMN external_uid TEXT,
  ADD COLUMN external_href TEXT,
  ADD COLUMN external_etag TEXT,
  ADD COLUMN account_id UUID REFERENCES caldav_accounts(id) ON DELETE CASCADE,
  ADD COLUMN dirty BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN reminded_at TIMESTAMPTZ;

CREATE UNIQUE INDEX entries_external_uid_key ON entries(external_uid) WHERE external_uid IS NOT NULL;

-- Tombstones so a local delete can propagate to the CalDAV server.
CREATE TABLE sync_tombstones (
  account_id UUID NOT NULL REFERENCES caldav_accounts(id) ON DELETE CASCADE,
  href TEXT NOT NULL,
  PRIMARY KEY (account_id, href)
);

-- +goose Down
ALTER TABLE entries DROP COLUMN reminded_at, DROP COLUMN account_id, DROP COLUMN dirty,
  DROP COLUMN external_etag, DROP COLUMN external_href, DROP COLUMN external_uid;
DROP TABLE sync_tombstones;
DROP TABLE caldav_accounts;
DROP TABLE push_subscriptions;
DROP TABLE server_config;
