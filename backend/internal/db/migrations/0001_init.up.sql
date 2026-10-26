-- Double-entry ledger. Every transaction has postings that sum to zero per currency.
-- User-visible money accounts are kind 'asset'; income, expense and equity accounts are
-- system accounts, one per user and currency, so reports never mix currencies.

create table users (
    id            bigserial primary key,
    telegram_id   bigint not null unique,
    first_name    text not null default '',
    username      text,
    language      text not null default 'uz-Latn' check (language in ('uz-Latn', 'uz-Cyrl', 'ru', 'en')),
    main_currency char(3) not null default 'UZS' check (main_currency in ('UZS', 'USD')),
    timezone      text not null default 'Asia/Tashkent',
    created_at    timestamptz not null default now()
);

create table households (
    id         bigserial primary key,
    name       text not null,
    created_at timestamptz not null default now()
);

create table household_members (
    household_id bigint not null references households on delete cascade,
    user_id      bigint not null references users on delete cascade,
    role         text not null check (role in ('owner', 'member')),
    primary key (household_id, user_id)
);

create table accounts (
    id           bigserial primary key,
    user_id      bigint not null references users on delete cascade,
    household_id bigint references households,
    kind         text not null check (kind in ('asset', 'income', 'expense', 'equity', 'receivable', 'payable')),
    type         text check (type in ('cash', 'humo', 'uzcard', 'visa', 'mastercard', 'savings', 'other')),
    name         text not null,
    currency     char(3) not null check (currency in ('UZS', 'USD')),
    -- system accounts: 'income', 'expense', 'opening', 'fx'
    system_key   text,
    archived_at  timestamptz,
    created_at   timestamptz not null default now(),
    unique (id, currency),
    check ((kind = 'asset') = (type is not null)),
    check ((kind = 'asset') = (system_key is null))
);
create unique index accounts_system_uniq on accounts (user_id, system_key, currency) where system_key is not null;
create index accounts_user_idx on accounts (user_id) where kind = 'asset';

create table categories (
    id          bigserial primary key,
    user_id     bigint references users on delete cascade, -- null: default category
    kind        text not null check (kind in ('expense', 'income')),
    key         text,                                      -- i18n key for defaults
    name        text,                                      -- custom categories
    icon        text not null,
    parent_id   bigint references categories,
    sort        int not null default 0,
    archived_at timestamptz,
    check ((key is null) <> (name is null))
);
create unique index categories_key_uniq on categories (key) where user_id is null;

create table transactions (
    id           bigserial primary key,
    user_id      bigint not null references users on delete cascade,
    household_id bigint references households,
    type         text not null check (type in ('expense', 'income', 'transfer', 'exchange', 'opening')),
    occurred_on  date not null,
    category_id  bigint references categories,
    description  text not null default '',
    source       text not null default 'manual' check (source in ('manual', 'chat', 'voice', 'bank', 'receipt')),
    source_ref   text,
    created_at   timestamptz not null default now(),
    updated_at   timestamptz not null default now(),
    deleted_at   timestamptz,
    check ((type in ('expense', 'income')) = (category_id is not null))
);
create unique index transactions_source_uniq on transactions (user_id, source, source_ref) where source_ref is not null;
create index transactions_user_date_idx on transactions (user_id, occurred_on desc, id desc) where deleted_at is null;

create table postings (
    id             bigserial primary key,
    transaction_id bigint not null references transactions on delete cascade,
    account_id     bigint not null,
    amount         bigint not null check (amount <> 0),
    currency       char(3) not null,
    -- a posting's currency always matches its account's
    foreign key (account_id, currency) references accounts (id, currency)
);
create index postings_tx_idx on postings (transaction_id);
create index postings_account_idx on postings (account_id);

-- Postings of a transaction must balance per currency. Checked at commit so a
-- transaction's postings can be inserted one by one.
create function postings_balanced() returns trigger language plpgsql as $$
declare
    tx bigint := coalesce(new.transaction_id, old.transaction_id);
begin
    if exists (
        select 1 from postings where transaction_id = tx
        group by currency having sum(amount) <> 0
    ) then
        raise exception 'transaction % postings do not balance', tx using errcode = '23514';
    end if;
    return null;
end $$;

create constraint trigger postings_balanced
    after insert or update or delete on postings
    deferrable initially deferred
    for each row execute function postings_balanced();

create table audit_log (
    id         bigserial primary key,
    user_id    bigint not null references users on delete cascade,
    entity     text not null,
    entity_id  bigint not null,
    action     text not null,
    before     jsonb,
    after      jsonb,
    at         timestamptz not null default now()
);
create index audit_log_entity_idx on audit_log (entity, entity_id);

insert into categories (kind, key, icon, sort) values
    ('expense', 'groceries',       '🛒', 10),
    ('expense', 'cafe',            '🍽', 20),
    ('expense', 'taxi',            '🚕', 30),
    ('expense', 'transport',       '🚌', 40),
    ('expense', 'fuel',            '⛽', 50),
    ('expense', 'utilities',       '💡', 60),
    ('expense', 'rent',            '🏠', 70),
    ('expense', 'mobile_internet', '📱', 80),
    ('expense', 'health',          '💊', 90),
    ('expense', 'clothing',        '👕', 100),
    ('expense', 'education',       '📚', 110),
    ('expense', 'entertainment',   '🎬', 120),
    ('expense', 'gifts',           '🎁', 130),
    ('expense', 'family',          '👨‍👩‍👧', 140),
    ('expense', 'beauty',          '💇', 150),
    ('expense', 'home',            '🛋', 160),
    ('expense', 'travel',          '✈️', 170),
    ('expense', 'fees',            '🏦', 180),
    ('expense', 'other_expense',   '📦', 1000),
    ('income',  'salary',          '💼', 10),
    ('income',  'freelance',       '💻', 20),
    ('income',  'gift_income',     '🎁', 30),
    ('income',  'cashback',        '💳', 40),
    ('income',  'other_income',    '💰', 1000);
