-- name: GetAccountByCard :one
select * from accounts
where user_id = @user_id and card_mask = @card_mask and kind = 'asset' and archived_at is null;

-- name: ReleaseCardMask :exec
update accounts set card_mask = null where user_id = @user_id and card_mask = @card_mask;

-- name: SetCardMask :exec
update accounts set card_mask = @card_mask where id = @id and user_id = @user_id and kind = 'asset';

-- name: RecordBankBalance :one
-- Keeps only the newest reported balance. Returns a row when this one is the newest.
update accounts set bank_balance = @balance, bank_balance_at = @at
where id = @id and user_id = @user_id and (bank_balance_at is null or bank_balance_at <= @at)
returning id;

-- name: FindSameDayDuplicate :one
-- An expense of the same amount and currency on the same day that did not come from a bank message.
select t.id, t.created_at, t.description
from transactions t
join postings p on p.transaction_id = t.id
join accounts a on a.id = p.account_id and a.kind = 'asset'
where t.user_id = @user_id and t.deleted_at is null and t.type = 'expense' and t.source <> 'bank'
  and t.occurred_on = @occurred_on and p.currency = @currency and p.amount < 0 and -p.amount = @amount
order by t.created_at desc
limit 1;
