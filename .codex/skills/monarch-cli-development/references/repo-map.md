# Repo Map

## Purpose

`monarch-cli` is a read-only, JSON-first Go CLI for Monarch Money's private web API. The project favors machine-readable output, shell pipelines, and agent workflows over an interactive terminal experience.

## Architecture

- `cmd/monarch/main.go`: binary entrypoint.
- `internal/cmd`: Kong command definitions and command handlers.
- `internal/app`: runtime wiring for config, token resolution, HTTP client setup, and printing.
- `internal/monarch`: GraphQL queries, response types, login flow, and API client methods.
- `internal/config`: JSON config file loading and saving.
- `internal/secrets`: keyring-backed token storage with a memory store used by tests.
- `internal/output`: pretty-printed JSON and simple table rendering.

## Command Design

- Root flags live in `internal/cmd/root.go`.
- First-class data commands live in `internal/cmd/data.go`.
- Auth flows live in `internal/cmd/auth.go`.
- Commands usually:
  1. build or resolve a runtime client,
  2. call a single Monarch client method,
  3. validate the GraphQL response,
  4. print normalized JSON and optional table output.

## Output Contract

- JSON is the default and should remain the primary contract.
- Table output is optional and should stay compact and easy to scan.
- `--raw` should return the raw GraphQL response shape instead of normalized command output.

## Auth And Storage

- Token precedence is: CLI flag, environment variable, stored keyring token.
- Config stores profile metadata but not the token itself.
- `UpsertProfile` writes non-secret profile metadata to config and the token to the keyring store.

## Testing Pattern

- Prefer `go test ./...`.
- Prefer `httptest.Server` for command-level tests instead of live API calls.
- Use golden JSON files in `internal/cmd/testdata` when the exact output shape matters.
- Use the in-memory secret store in tests to avoid touching the real keyring.

## Change Heuristics

- For new CLI surface area, update `README.md` with example usage.
- For new GraphQL-backed reads, keep the CLI normalized output small and stable even if upstream responses are verbose.
- For upstream API drift, patch the narrowest query and response structs necessary before widening broader abstractions.
