-- name: InsertTransaction :one
insert into transactions (user_id, type, occurred_on, category_id, description, source, source_ref)
values (@user_id, @type, @occurred_on, @category_id, @description, @source, @source_ref)
on conflict (user_id, source, source_ref) where source_ref is not null do nothing
returning *;

-- name: InsertPosting :exec
insert into postings (transaction_id, account_id, amount, currency)
values (@transaction_id, @account_id, @amount, @currency);

-- name: DeletePostings :exec
delete from postings where transaction_id = @transaction_id;

-- name: GetTransactionBySourceRef :one
select * from transactions
where user_id = @user_id and source = @source and source_ref = @source_ref;

-- name: GetTransaction :one
select * from transactions
where id = @id and user_id = @user_id and (@with_deleted::bool or deleted_at is null)
for update;

-- name: UpdateTransaction :one
update transactions
set type = @type, occurred_on = @occurred_on, category_id = @category_id,
    description = @description, updated_at = now()
where id = @id and user_id = @user_id
returning *;

-- name: SetTransactionDeleted :one
update transactions
set deleted_at = case when @deleted::bool then now() else null end, updated_at = now()
where id = @id and user_id = @user_id
returning *;

-- name: ListPostings :many
select p.transaction_id, p.account_id, p.amount, p.currency, a.kind
from postings p
join accounts a on a.id = p.account_id
where p.transaction_id = any(@transaction_ids::bigint[])
order by p.transaction_id, p.id;

-- name: ListTransactions :many
select t.* from transactions t
where t.user_id = @user_id and t.deleted_at is null
  and (sqlc.narg(from_date)::date is null or t.occurred_on >= sqlc.narg(from_date)::date)
  and (sqlc.narg(to_date)::date is null or t.occurred_on <= sqlc.narg(to_date)::date)
  and (sqlc.narg(type)::text is null or t.type = sqlc.narg(type)::text)
  and (sqlc.narg(category_id)::bigint is null or t.category_id = sqlc.narg(category_id)::bigint)
  and (sqlc.narg(q)::text is null or t.description ilike '%' || sqlc.narg(q)::text || '%')
  and (sqlc.narg(account_id)::bigint is null or exists (
        select 1 from postings p where p.transaction_id = t.id and p.account_id = sqlc.narg(account_id)::bigint))
  and (sqlc.narg(currency)::text is null or exists (
        select 1 from postings p join accounts a on a.id = p.account_id
        where p.transaction_id = t.id and a.kind = 'asset' and p.currency = sqlc.narg(currency)::text))
  and (sqlc.narg(cursor_date)::date is null
       or (t.occurred_on, t.id) < (sqlc.narg(cursor_date)::date, sqlc.narg(cursor_id)::bigint))
order by t.occurred_on desc, t.id desc
limit @lim;

-- name: InsertAudit :exec
insert into audit_log (user_id, entity, entity_id, action, before, after)
values (@user_id, @entity, @entity_id, @action, @before, @after);
