alter table users
    add column started_at      timestamptz,             -- first /start in the bot
    add column notify_enabled  boolean not null default true,
    add column notify_time     time not null default '21:00',
    add column last_summary_on date;

-- Every inbound bot update, so retries and replays are processed once.
create table ingestion_events (
    id          bigserial primary key,
    source      text not null,          -- 'telegram'
    external_id text not null,          -- update_id
    user_id     bigint references users on delete cascade,
    kind        text not null,          -- text, voice, callback, command
    created_at  timestamptz not null default now(),
    unique (source, external_id)
);

-- Captured items waiting for one answer (account, amount, duplicate) before they
-- touch balances. Confident items skip this table and go straight to the ledger.
create table drafts (
    id             bigserial primary key,
    user_id        bigint not null references users on delete cascade,
    status         text not null default 'pending' check (status in ('pending', 'committed', 'cancelled', 'expired')),
    question       text not null check (question in ('account', 'amount', 'duplicate')),
    payload        jsonb not null,
    source         text not null check (source in ('chat', 'voice')),
    source_ref     text not null,
    transaction_id bigint references transactions,
    created_at     timestamptz not null default now(),
    expires_at     timestamptz not null default now() + interval '24 hours'
);
create index drafts_pending_idx on drafts (user_id) where status = 'pending';

-- Learned per-user mapping from normalized description to category and account.
create table merchant_rules (
    user_id     bigint not null references users on delete cascade,
    pattern     text not null,
    category_id bigint not null references categories,
    account_id  bigint references accounts on delete set null,
    origin      text not null check (origin in ('user', 'ai')),
    hits        int not null default 1,
    updated_at  timestamptz not null default now(),
    primary key (user_id, pattern)
);

-- Product analytics. Never store amounts, descriptions or transcripts here.
create table events (
    id      bigserial primary key,
    user_id bigint references users on delete cascade,
    name    text not null,
    props   jsonb not null default '{}',
    at      timestamptz not null default now()
);
create index events_name_at_idx on events (name, at);
create index events_user_idx on events (user_id, name);
