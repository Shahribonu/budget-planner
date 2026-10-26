# budget-planner

Personal finance assistant for Uzbekistan as a Telegram bot + Mini App: log spending by chat or voice in Uzbek (Latin/Cyrillic), Russian or English; plan the month; track debts and installments.

## Layout

```
backend/                 Go modular monolith (API, bot webhook, job worker)
  internal/money/        integer amounts in ISO minor units, explicit currency
  internal/capture/
    normalizer/          deterministic amount/date parser: "qirq besh ming", "полтора миллиона", "$20"
app/                     Vue 3 Mini App (not started)
```

## Development

```sh
cd backend
go test ./...
```

## Principles

- Money is integer minor units with explicit currency; UZS and USD are never added silently.
- Transfers never count as income or expense.
- AI proposes drafts, the Go domain service validates and writes.
