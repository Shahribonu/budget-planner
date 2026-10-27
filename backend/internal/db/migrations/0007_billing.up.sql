-- Premium through Telegram Stars. premium_until is the single source of entitlement.
alter table users add column premium_until timestamptz, add column trial_started_at timestamptz;

create table payments (
    id                 bigserial primary key,
    user_id            bigint not null references users on delete cascade,
    telegram_charge_id text not null unique,
    payload            text not null,
    stars              int not null check (stars > 0),
    recurring          boolean not null default false,
    premium_until      timestamptz not null,
    created_at         timestamptz not null default now()
);
create index payments_user_idx on payments (user_id);
