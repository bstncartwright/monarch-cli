---
name: monarch-cli-development
description: Develop, debug, and extend the monarch-cli repository, a read-only JSON-first Go CLI for Monarch Money. Use when working in this repo on command behavior, auth and profile handling, GraphQL queries, output formatting, config or keyring storage, or tests for new CLI features and bug fixes.
---

# Monarch CLI Development

Use this skill to make changes in `monarch-cli` without losing the repo's core constraints: read-only behavior, JSON-first output, and thin command handlers over GraphQL calls.

## Quick Start

1. Read `README.md` and `references/repo-map.md` before editing.
2. Keep new features read-only unless the user explicitly asks for a write path.
3. Preserve JSON output as the default. Only add table output when the command has a stable compact view.
4. Prefer adding coverage with `httptest`-backed command tests over manual-only verification.

## Working Rules

- Route all user-facing commands through `internal/cmd`.
- Keep HTTP and GraphQL details in `internal/monarch`.
- Keep token, profile, and config behavior in `internal/app`, `internal/config`, and `internal/secrets`.
- Preserve auth precedence exactly: `--token`, then `MONARCH_TOKEN`, then stored keyring token.
- Reuse `printMaybeRaw` and `output.Table` patterns so `--raw` and `--format table` continue to work consistently.
- Treat Monarch's private API as unstable. When a GraphQL field breaks, change the narrowest query and normalization path that restores behavior.

## Common Changes

### Add a new first-class read command

1. Add or update the GraphQL query and response types in `internal/monarch/api.go`.
2. Add a client method in `internal/monarch/client.go` or the relevant Monarch package file.
3. Add the CLI command struct and `Run` method in `internal/cmd`.
4. Normalize the output for JSON-first usage; add table output only if it is obviously useful.
5. Add command tests using `httptest.Server` and golden JSON when the output shape should stay stable.

### Fix auth or profile behavior

1. Start in `internal/app/runtime.go` and `internal/cmd/auth.go`.
2. Check whether the bug is in token resolution, config persistence, or keyring interaction.
3. Add or update tests in `internal/app/runtime_test.go` or `internal/cmd/root_test.go` before changing storage semantics.

### Fix a broken GraphQL call

1. Inspect the query text in `internal/monarch/api.go`.
2. Adjust the matching response structs and normalization code together.
3. Keep the public CLI shape as stable as possible even if the upstream response changes.
4. Add a regression test around the changed path.

## Verification

- Run `go test ./...` after changes.
- When output shape changes intentionally, update the golden files in `internal/cmd/testdata`.
- When adding a command, exercise both JSON output and table output if table support exists.

## References

- Read `references/repo-map.md` for the repo layout, change points, and test strategy.
