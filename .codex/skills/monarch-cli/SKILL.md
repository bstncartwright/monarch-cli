---
name: monarch-cli
description: Use the monarch CLI to authenticate to Monarch Money, inspect accounts, balances, holdings, transactions, budgets, cashflow, recurring items, and run raw GraphQL queries. Use when an agent needs to answer finance questions or gather Monarch data from a terminal with JSON-first output, shell pipelines, or direct GraphQL access.
---

# Monarch CLI

Use `monarch-cli` for read-only access to Monarch Money data from the terminal.

## Core Rules

- Prefer the CLI's first-class commands before falling back to raw GraphQL.
- Expect JSON output by default and pipe it into `jq` when filtering or summarizing.
- Use `--format table` only for quick terminal inspection.
- Use `--raw` when a first-class command exists but the agent needs the upstream GraphQL response shape.
- Use `YYYY-MM-DD` dates.
- Assume the tool is read-only.

## Authentication

Use one of these auth paths:

1. `monarch auth login`
2. `monarch auth token set <TOKEN> --email you@example.com`
3. `echo "$MONARCH_TOKEN" | monarch auth token set --stdin --email you@example.com`

Token precedence for any command:

1. `--token`
2. `MONARCH_TOKEN`
3. stored profile token

Useful checks:

```bash
monarch auth status
monarch auth logout
```

If the Monarch account uses Google Sign-In or Apple Sign-In, create a direct Monarch password first.

## Command Map

```bash
monarch me get

monarch accounts list
monarch accounts balances --start-date 2026-02-01
monarch accounts holdings <account-id>
monarch accounts history <account-id>

monarch transactions list --limit 25
monarch transactions list --start-date 2026-02-01 --end-date 2026-02-28
monarch transactions get <transaction-id>
monarch transactions summary --start-date 2026-02-01 --end-date 2026-02-28

monarch budgets get
monarch budgets get --start-date 2026-03-01 --end-date 2026-03-31

monarch cashflow get
monarch cashflow get --start-date 2026-03-01 --end-date 2026-03-31

monarch recurring list
monarch recurring list --start-date 2026-03-01 --end-date 2026-03-31

monarch graphql query '{ me { id email } }'
```

## Default Date Behavior

- `accounts balances` defaults `--start-date` to 31 days ago.
- `transactions list` and `transactions summary` default to the last 30 days.
- `budgets get`, `cashflow get`, and `recurring list` default to the current month.

## Common Workflows

### Inspect identity and auth state

```bash
monarch auth status
monarch me get
```

### List accounts and balances

```bash
monarch accounts list
monarch --format table accounts list
monarch accounts balances --start-date 2026-02-01
```

### Investigate an account

```bash
monarch accounts holdings <account-id>
monarch accounts history <account-id>
```

### Search recent transactions

```bash
monarch transactions list --limit 50 --search "amazon"
monarch transactions list --start-date 2026-02-01 --end-date 2026-02-28
monarch transactions get <transaction-id>
monarch transactions summary --start-date 2026-02-01 --end-date 2026-02-28
```

### Review budgets, cashflow, and recurring items

```bash
monarch budgets get
monarch cashflow get
monarch recurring list
```

## Shell Patterns

Extract fields with `jq`:

```bash
monarch accounts list | jq '.[].displayName'
monarch transactions list --limit 5 | jq '.results[] | {date, amount, merchant: .merchant.name}'
monarch cashflow get | jq '.summary'
```

Switch profiles or override config:

```bash
monarch --profile work auth status
monarch --config /tmp/monarch.json auth status
```

Use table output for quick inspection:

```bash
monarch --format table accounts list
monarch --format table transactions list --limit 10
```

## Raw GraphQL

Use raw GraphQL when the first-class commands do not expose the field set you need.

Inline query:

```bash
monarch graphql query '{ me { id email } }'
```

Query with variables:

```bash
monarch graphql query \
  'query Account($id: UUID!) { account(id: $id) { id displayName } }' \
  --variables '{"id":"<account-id>"}'
```

Read from a file:

```bash
monarch graphql query --file ./query.graphql --variables '{"id":"<account-id>"}'
```

Read from stdin:

```bash
cat ./query.graphql | monarch graphql query --stdin --variables '{"id":"<account-id>"}'
```

## Agent Guidance

- Start with `monarch auth status` if auth state is unclear.
- Prefer normalized first-class commands for downstream automation.
- Reach for `--raw` only when the normalized command omits needed fields.
- Reach for `graphql query` only when no first-class command fits.
- When reporting date-based results, include the exact start and end dates used in the command.
