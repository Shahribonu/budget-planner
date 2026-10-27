-- name: GetUserByTelegramID :one
select * from users where telegram_id = @telegram_id;

-- name: MarkStarted :one
-- Returns true only the first time.
update users set started_at = now() where id = @id and started_at is null
returning id;

-- name: ClaimIngestion :one
-- Returns a row only for the first delivery of an update.
insert into ingestion_events (source, external_id, user_id, kind)
values (@source, @external_id, @user_id, @kind)
on conflict (source, external_id) do nothing
returning id;

-- name: InsertEvent :exec
insert into events (user_id, name, props) values (@user_id, @name, @props);

-- name: HasEvent :one
select exists (select 1 from events where user_id = @user_id and name = @name);

-- name: ListAccountsForCapture :many
-- Active accounts in one currency, most recently used first.
select a.id, a.name, a.type, a.currency,
       coalesce((select max(t.created_at) from postings p join transactions t on t.id = p.transaction_id
                 where p.account_id = a.id and t.deleted_at is null), 'epoch')::timestamptz as last_used
from accounts a
where a.user_id = @user_id and a.kind = 'asset' and a.archived_at is null and a.currency = @currency
order by last_used desc, a.created_at, a.id;

-- name: CountAssetAccounts :one
select count(*) from accounts where user_id = @user_id and kind = 'asset' and archived_at is null;

-- name: GetMerchantRule :one
select * from merchant_rules where user_id = @user_id and pattern = @pattern;

-- name: UpsertMerchantRule :exec
-- A user's own choice always overrides an AI guess; an AI guess never overrides the user.
insert into merchant_rules (user_id, pattern, category_id, account_id, origin)
values (@user_id, @pattern, @category_id, @account_id, @origin)
on conflict (user_id, pattern) do update
set category_id = case when merchant_rules.origin = 'user' and excluded.origin = 'ai'
                       then merchant_rules.category_id else excluded.category_id end,
    account_id  = coalesce(excluded.account_id, merchant_rules.account_id),
    origin      = case when merchant_rules.origin = 'user' then 'user' else excluded.origin end,
    hits        = merchant_rules.hits + 1,
    updated_at  = now();

-- name: FindPossibleDuplicate :one
-- Same amount and currency on the same day, recorded in the last 15 minutes from another message.
select t.id, t.created_at, t.description
from transactions t
join postings p on p.transaction_id = t.id
join accounts a on a.id = p.account_id and a.kind = 'asset'
where t.user_id = @user_id and t.deleted_at is null and t.type = @type
  and t.occurred_on = @occurred_on and p.currency = @currency and abs(p.amount) = @amount
  and t.created_at > now() - interval '15 minutes'
  and (t.source_ref is null or t.source_ref not like @source_prefix::text || '%')
order by t.created_at desc
limit 1;

-- name: CreateDraft :one
insert into drafts (user_id, question, payload, source, source_ref)
values (@user_id, @question, @payload, @source, @source_ref)
returning *;

-- name: GetPendingDraft :one
select * from drafts
where id = @id and user_id = @user_id and status = 'pending' and expires_at > now()
for update;

-- name: ResolveDraft :exec
update drafts set status = @status, transaction_id = @transaction_id where id = @id;

-- name: ExpireDrafts :execrows
update drafts set status = 'expired' where status = 'pending' and expires_at <= now();

-- name: LastChatTransaction :one
select * from transactions
where user_id = @user_id and deleted_at is null and source in ('chat', 'voice')
order by created_at desc
limit 1;

-- name: UsersDueForSummary :many
-- Users whose local clock has passed their notify time today and who haven't had a summary today.
select u.* from users u
where u.notify_enabled and u.started_at is not null
  -- within two hours after notify_time; time arithmetic wraps at midnight
  and case when u.notify_time + interval '2 hours' > u.notify_time
           then (now() at time zone u.timezone)::time >= u.notify_time
                and (now() at time zone u.timezone)::time < u.notify_time + interval '2 hours'
           else (now() at time zone u.timezone)::time >= u.notify_time
                or (now() at time zone u.timezone)::time < u.notify_time + interval '2 hours'
      end
  and (u.last_summary_on is null or u.last_summary_on < (now() at time zone u.timezone)::date);

-- name: MarkSummarySent :exec
update users set last_summary_on = @day where id = @id;

-- name: DaySpending :many
select p.currency, count(distinct t.id)::bigint as n, sum(p.amount)::bigint as amount
from transactions t
join postings p on p.transaction_id = t.id
join accounts a on a.id = p.account_id and a.system_key = 'expense'
where t.user_id = @user_id and t.deleted_at is null and t.occurred_on = @day
group by p.currency
order by p.currency;

-- name: SetNotify :one
update users
set notify_enabled = coalesce(sqlc.narg(enabled), notify_enabled),
    notify_time    = coalesce(sqlc.narg(notify_time)::time, notify_time)
where id = @id
returning *;
