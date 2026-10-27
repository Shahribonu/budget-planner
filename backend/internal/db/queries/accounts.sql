-- name: CreateAssetAccount :one
insert into accounts (user_id, kind, type, name, currency)
values (@user_id, 'asset', @type, @name, @currency)
returning *;

-- name: GetAssetAccount :one
select * from accounts where id = @id and user_id = @user_id and kind = 'asset';

-- name: ListAssetAccounts :many
select a.id, a.name, a.type, a.currency, a.archived_at, a.color, a.include_in_total, a.card_mask,
       coalesce(sum(p.amount) filter (where t.deleted_at is null), 0)::bigint as balance
from accounts a
left join postings p on p.account_id = a.id
left join transactions t on t.id = p.transaction_id
where a.user_id = @user_id and a.kind = 'asset'
  and (@include_archived::bool or a.archived_at is null)
group by a.id
order by a.created_at, a.id;

-- name: AccountBalance :one
select coalesce(sum(p.amount), 0)::bigint
from postings p
join transactions t on t.id = p.transaction_id
where p.account_id = @account_id and t.deleted_at is null;

-- name: UpdateAssetAccount :one
update accounts
set name        = coalesce(sqlc.narg(name), name),
    color       = case when @set_color::bool then sqlc.narg(color)::text else color end,
    include_in_total = coalesce(sqlc.narg(include_in_total), include_in_total),
    archived_at = case
        when sqlc.narg(archived)::bool is null then archived_at
        when sqlc.narg(archived)::bool then coalesce(archived_at, now())
        else null
    end
where id = @id and user_id = @user_id and kind = 'asset'
returning *;

-- name: EnsureSystemAccount :one
insert into accounts (user_id, kind, system_key, name, currency)
values (@user_id, @kind, @system_key, @system_key, @currency)
on conflict (user_id, system_key, currency) where system_key is not null
do update set name = excluded.name
returning id;
