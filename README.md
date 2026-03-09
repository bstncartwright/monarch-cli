# monarch-cli

`monarch-cli` is a JSON-first command-line client for reading data from [Monarch Money](https://www.monarchmoney.com/).

It is built for scripting, agent workflows, shell automation, and quick inspection from a terminal. The CLI stores a Monarch session token securely in your system keyring, prints machine-readable JSON by default, and exposes a raw GraphQL escape hatch for power users.

## Status

This project is usable today, but it is still early.

- Read-only by design
- Built against Monarch's private web API, not an official public data API
- Best suited for local/agent workflows where JSON output matters more than a polished interactive TUI

If Monarch changes their internal API, this CLI may need updates.

## What It Can Do

- Log in with email/password and MFA, then store a session token securely
- List accounts and balances
- Fetch account history and holdings
- List transactions and transaction summaries
- Read budgets, cashflow, and recurring items
- Run arbitrary GraphQL queries against Monarch's API

## Important Caveat

This CLI uses Monarch's private web API at `https://api.monarch.com/graphql`.

That means:

- there is no stability guarantee from Monarch
- fields and query behavior can change without notice
- this project should stay conservative and read-only by default

If you want a stable, vendor-supported integration contract, this is not that.

## Installation

### Build from source

```bash
git clone <your-repo-url>
cd monarch-cli
go build -o monarch ./cmd/monarch
```

### Development build

```bash
go build ./cmd/monarch
```

## Quick Start

### 1. Authenticate

```bash
./monarch auth login
```

This prompts for:

- email
- password
- MFA code, if MFA is enabled

If your Monarch account uses Google Sign-In or Apple Sign-In, you need to create a direct Monarch password first in Monarch's security settings.

### 2. Check auth state

```bash
./monarch auth status
```

### 3. Run a few basic reads

```bash
./monarch me get
./monarch accounts list
./monarch transactions list --limit 5
```

## Authentication Model

`monarch-cli` stores only the Monarch session token. It does not persist your password.

Auth precedence for any command is:

1. `--token`
2. `MONARCH_TOKEN`
3. stored profile token from the keyring

### Login flow

```bash
./monarch auth login
```

### Store a token directly

```bash
echo 'YOUR_MONARCH_TOKEN' | ./monarch auth token set --stdin --email you@example.com
```

Or:

```bash
./monarch auth token set YOUR_MONARCH_TOKEN --email you@example.com
```

### Remove stored auth

```bash
./monarch auth logout
```

## Profiles, Config, and Secret Storage

The CLI supports named profiles via `--profile`.

Non-secret metadata is stored in a JSON config file. The path depends on your OS:

- macOS: `~/Library/Application Support/monarch/config.json`
- Linux: `~/.config/monarch/config.json`

Tokens are stored in the system keyring via `github.com/99designs/keyring`.

You can override the config path:

```bash
./monarch --config /tmp/monarch.json auth status
```

## Output Model

JSON is the default output format.

That is intentional. This project is meant to be easy to pipe into:

- `jq`
- shell scripts
- LLM/agent tools
- other CLIs

Example:

```bash
./monarch accounts list | jq '.[].displayName'
```

For a compact terminal view on supported commands:

```bash
./monarch --format table accounts list
```

For raw GraphQL response shapes instead of normalized command output:

```bash
./monarch --raw me get
```

## Commands

### Auth

```bash
./monarch auth login
./monarch auth token set [TOKEN] [--stdin] [--email you@example.com]
./monarch auth status
./monarch auth logout
```

### Profile

```bash
./monarch me get
```

### Accounts

```bash
./monarch accounts list
./monarch accounts balances
./monarch accounts balances --start-date 2026-02-01
./monarch accounts holdings <account-id>
./monarch accounts history <account-id>
```

### Transactions

```bash
./monarch transactions list
./monarch transactions list --limit 25
./monarch transactions list --start-date 2026-02-01 --end-date 2026-02-29
./monarch transactions get <transaction-id>
./monarch transactions summary
./monarch transactions summary --start-date 2026-02-01 --end-date 2026-02-29
```

### Budgets and cashflow

```bash
./monarch budgets get
./monarch budgets get --start-date 2026-03-01 --end-date 2026-03-31
./monarch cashflow get
./monarch cashflow get --start-date 2026-03-01 --end-date 2026-03-31
```

### Recurring items

```bash
./monarch recurring list
./monarch recurring list --start-date 2026-03-01 --end-date 2026-03-31
```

### Raw GraphQL

Inline query:

```bash
./monarch graphql query '{ me { id email } }'
```

Query with variables:

```bash
./monarch graphql query \
  'query Me($id: ID!) { me { id email } }' \
  --variables '{"id":"user_123"}'
```

Read query text from a file:

```bash
./monarch graphql query --file ./query.graphql --variables '{"foo":"bar"}'
```

Read query text from stdin:

```bash
cat query.graphql | ./monarch graphql query --stdin
```

## Useful Examples

Get account names and balances:

```bash
./monarch accounts list | jq '.[] | {name: .displayName, balance: .currentBalance}'
```

Get the last five transactions:

```bash
./monarch transactions list --limit 5 | jq '.results[] | {date, amount, merchant: .merchant.name}'
```

Check monthly cashflow summary:

```bash
./monarch cashflow get | jq '.summary'
```

Inspect the current authenticated user:

```bash
./monarch me get
```

## Global Flags

These flags are available on all commands:

- `--format json|table`
- `--raw`
- `--profile <name>`
- `--token <token>`
- `--config <path>`

See full help:

```bash
./monarch --help
./monarch <command> --help
```

## Development

### Run tests

```bash
go test ./...
```

### Format code

```bash
gofmt -w ./cmd ./internal
```

### Build

```bash
go build -o monarch ./cmd/monarch
```

## Security Notes

- Passwords are only used during login and are not written to disk
- Stored auth uses a session token in the OS keyring
- Budget, transaction, and account data can be sensitive; be careful when piping output into logs, shells, or agent transcripts
- `--token` and `MONARCH_TOKEN` are convenient, but shell history and process listings can expose secrets if you are careless

## Known Limitations

- This is a private-API client and may break when Monarch changes backend behavior
- `accounts history` currently returns valid account metadata and snapshots, but the nested `recent_transactions` ordering still needs refinement
- Output normalization is intentionally shallow; use `--raw` or `graphql query` when you need exact backend response shapes
- There are no write/mutation commands in v1

## Roadmap Ideas

- richer account and transaction filters
- better paging support
- first-class export formats
- more polished table output
- optional schema/version diagnostics
- a tighter query catalog for power-user workflows

## Contributing

Issues and pull requests are welcome.

When contributing:

- prefer small, focused query changes
- keep new commands read-only unless mutation support is explicitly discussed
- test against the real API carefully, because type validity alone is not enough with a private backend

## Disclaimer

This project is not affiliated with, endorsed by, or supported by Monarch Money.
