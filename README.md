# budget-planner

Personal finance assistant for Uzbekistan as a Telegram bot + Mini App: log spending by chat or voice in Uzbek (Latin/Cyrillic), Russian or English; plan the month; track debts and installments.

## Layout

```
api/openapi.yaml         Mini App API contract
backend/                 Go modular monolith (API now; bot webhook and job worker later)
  cmd/server/            HTTP server entry point
  internal/api/          HTTP handlers, Telegram init data auth
  internal/auth/         init data HMAC validation
  internal/ledger/       domain service: the only writer of money data (double-entry)
  internal/db/           migrations, sqlc queries and generated code (dbq/)
  internal/money/        integer amounts in ISO minor units, explicit currency
  internal/capture/
    normalizer/          deterministic amount/date parser: "qirq besh ming", "полтора миллиона", "$20"
app/                     Vue 3 Telegram Mini App
```

## Development

Needs Go, Node 20 and PostgreSQL.

```sh
createdb budget_planner && createdb budget_planner_test

# backend
cd backend
cp .env.example .env            # then edit
set -a; source .env; set +a
go run ./cmd/server             # migrates and serves :8080
go test ./...                   # integration tests run when TEST_DATABASE_URL is set

# after editing internal/db/queries or migrations
sqlc generate                   # go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.29.0

# Mini App
cd app
cp .env.example .env            # VITE_DEV_TELEGRAM_ID=1 to use dev auth in a browser
npm install && npm run dev      # proxies /api to :8080
```

## Principles

- Money is integer minor units with explicit currency; UZS and USD are never added silently.
- Transfers never count as income or expense.
- AI proposes drafts, the Go domain service validates and writes.
