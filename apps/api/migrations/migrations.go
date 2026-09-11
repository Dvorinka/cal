// Package migrations embeds the SQL migrations so binaries can apply them
// in-process (desktop app, tests) without the goose CLI.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
