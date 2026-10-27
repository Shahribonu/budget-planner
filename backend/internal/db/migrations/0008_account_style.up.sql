-- How an account looks and whether it counts in the total balance.
alter table accounts
    add column color text check (color ~ '^#[0-9a-fA-F]{6}$'),
    add column include_in_total boolean not null default true;
