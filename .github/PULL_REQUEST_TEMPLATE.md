## Summary

<!-- What does this change and why? -->

## Test plan

<!-- How was it verified? Commands run, pages checked. -->

- [ ] `npm run typecheck && npm run test && npm run build` passes
- [ ] `cd apps/api && go vet ./... && go test ./...` passes (if Go touched)
- [ ] Checked at desktop and mobile widths (if UI touched)
- [ ] Migrations keep `-- +goose Up/Down` and stay idempotent (if touched)
