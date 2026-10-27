-- Monthly spending limits, overall (category_id null) or per expense category, in one currency.
create table budgets (
    id          bigserial primary key,
    user_id     bigint not null references users on delete cascade,
    category_id bigint references categories,
    currency    char(3) not null check (currency in ('UZS', 'USD')),
    amount      bigint not null check (amount > 0),
    created_at  timestamptz not null default now(),
    deleted_at  timestamptz
);
create unique index budgets_one_per_scope on budgets (user_id, currency, coalesce(category_id, 0)) where deleted_at is null;

-- One alert per budget, month and threshold (80, 100).
create table budget_alerts (
    budget_id bigint not null references budgets on delete cascade,
    month     text not null,
    threshold int not null,
    sent_at   timestamptz not null default now(),
    primary key (budget_id, month, threshold)
);
