-- name: AssetTotals :many
select p.currency, sum(p.amount)::bigint as amount
from postings p
join transactions t on t.id = p.transaction_id
join accounts a on a.id = p.account_id
where a.user_id = @user_id and a.kind = 'asset' and a.include_in_total and t.deleted_at is null
group by p.currency
order by p.currency;

-- name: FlowTotals :many
-- Income postings are negative (credit), expense postings positive (debit).
select a.system_key::text as flow, p.currency, sum(p.amount)::bigint as amount
from postings p
join transactions t on t.id = p.transaction_id
join accounts a on a.id = p.account_id
where t.user_id = @user_id and t.deleted_at is null
  and t.occurred_on >= @from_date and t.occurred_on < @to_date
  and a.system_key in ('income', 'expense')
group by a.system_key, p.currency
order by a.system_key, p.currency;

-- name: ExpenseByCategory :many
select t.category_id::bigint as category_id, p.currency, sum(p.amount)::bigint as amount
from postings p
join transactions t on t.id = p.transaction_id
join accounts a on a.id = p.account_id
where t.user_id = @user_id and t.deleted_at is null
  and t.occurred_on >= @from_date and t.occurred_on < @to_date
  and a.system_key = 'expense'
group by t.category_id, p.currency
order by amount desc;
