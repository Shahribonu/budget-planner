-- Recurring income and bills: salary, rent, subscriptions. They are not posted
-- automatically; they feed the monthly plan's expected income and obligations.
create table recurring (
    id           bigserial primary key,
    user_id      bigint not null references users on delete cascade,
    kind         text not null check (kind in ('income', 'expense')),
    title        text not null,
    category_id  bigint references categories,
    currency     char(3) not null check (currency in ('UZS', 'USD')),
    amount       bigint not null check (amount > 0),
    day_of_month int not null check (day_of_month between 1 and 31),
    account_id   bigint references accounts on delete set null,
    created_at   timestamptz not null default now(),
    deleted_at   timestamptz
);
create index recurring_user_idx on recurring (user_id) where deleted_at is null;

-- One plan per user, month and currency. Overrides replace the computed estimates.
create table monthly_plans (
    id              bigserial primary key,
    user_id         bigint not null references users on delete cascade,
    month           date not null check (extract(day from month) = 1),
    currency        char(3) not null check (currency in ('UZS', 'USD')),
    income_override bigint check (income_override >= 0),
    income_day      int check (income_day between 1 and 31),
    savings         bigint not null default 0 check (savings >= 0),
    buffer_override bigint check (buffer_override >= 0),
    created_at      timestamptz not null default now(),
    unique (user_id, month, currency)
);

create table plan_items (
    id              bigserial primary key,
    plan_id         bigint not null references monthly_plans on delete cascade,
    title           text not null,
    category_id     bigint references categories,
    amount_min      bigint not null check (amount_min > 0),
    amount_max      bigint not null check (amount_max >= amount_min),
    priority        text not null check (priority in ('must', 'should', 'nice')),
    due_on          date,
    status          text not null default 'planned' check (status in ('planned', 'bought', 'postponed', 'dropped')),
    postponed_count int not null default 0,
    transaction_id  bigint references transactions on delete set null,
    carried_from    bigint references plan_items on delete set null,
    note            text not null default '',
    created_at      timestamptz not null default now()
);
create index plan_items_plan_idx on plan_items (plan_id);

alter table users add column chat_mode text, add column chat_mode_until timestamptz, add column plan_prompted_month date;
