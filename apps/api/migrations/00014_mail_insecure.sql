-- Per-account TLS trust override for self-hosted mail with self-signed certs.
-- +goose Up
ALTER TABLE mail_accounts ADD COLUMN insecure_tls BOOLEAN NOT NULL DEFAULT false;
-- +goose Down
ALTER TABLE mail_accounts DROP COLUMN IF EXISTS insecure_tls;
