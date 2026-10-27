-- Debts (qarz) and installment plans (nasiya). Each one owns a receivable or payable
-- ledger account; what is still owed is that account's balance, so undoing a payment
-- in the ledger automatically reopens the debt.

-- Receivable and payable accounts belong to one debt or plan and have no system key.
alter table accounts drop constraint accounts_check1;
alter table accounts add constraint accounts_asset_no_key check (kind <> 'asset' or system_key is null);
alter table accounts add constraint accounts_system_has_key check (kind not in ('income', 'expense', 'equity') or system_key is not null);

alter table transactions drop constraint transactions_type_check;
alter table transactions add constraint transactions_type_check check (type in (
    'expense', 'income', 'transfer', 'exchange', 'opening',
    'debt', 'debt_payment', 'installment', 'installment_payment'));
alter table transactions drop constraint transactions_check;
alter table transactions add constraint transactions_category_check check (
    (type in ('expense', 'income', 'installment')) = (category_id is not null));

create table debts (
    id           bigserial primary key,
    user_id      bigint not null references users on delete cascade,
    direction    text not null check (direction in ('lent', 'borrowed')),
    counterparty text not null,
    account_id   bigint not null unique references accounts,
    currency     char(3) not null,
    due_on       date,
    note         text not null default '',
    created_at   timestamptz not null default now(),
    deleted_at   timestamptz
);
create index debts_user_idx on debts (user_id) where deleted_at is null;

create table installment_plans (
    id                  bigserial primary key,
    user_id             bigint not null references users on delete cascade,
    title               text not null,
    provider            text not null default '',
    category_id         bigint not null references categories,
    account_id          bigint not null unique references accounts,
    currency            char(3) not null,
    total               bigint not null check (total > 0),
    months              int not null check (months between 1 and 60),
    pay_from_account_id bigint references accounts on delete set null,
    created_at          timestamptz not null default now(),
    deleted_at          timestamptz
);
create index installment_plans_user_idx on installment_plans (user_id) where deleted_at is null;

-- Which rows are paid is derived from the total paid so far, not stored.
create table installment_schedule (
    plan_id     bigint not null references installment_plans on delete cascade,
    n           int not null,
    due_on      date not null,
    amount      bigint not null check (amount > 0),
    primary key (plan_id, n)
);
