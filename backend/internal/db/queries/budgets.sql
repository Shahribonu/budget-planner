-- name: InsertBudget :one
insert into budgets (user_id, category_id, currency, amount)
values (@user_id, @category_id, @currency, @amount)
returning *;

-- name: UpdateBudgetAmount :one
update budgets set amount = @amount
where id = @id and user_id = @user_id and deleted_at is null
returning *;

-- name: SoftDeleteBudget :execrows
update budgets set deleted_at = now() where id = @id and user_id = @user_id and deleted_at is null;

-- name: BudgetProgress :many
-- Spending in [from_date, to_date) against each active budget. Installment purchases count;
-- transfers, debts and installment payments don't.
select b.*,
       coalesce((select sum(p.amount) from postings p
                 join transactions t on t.id = p.transaction_id
                 join accounts a on a.id = p.account_id
                 where t.user_id = b.user_id and t.deleted_at is null and a.system_key = 'expense'
                   and p.currency = b.currency
                   and (b.category_id is null or t.category_id = b.category_id)
                   and t.occurred_on >= @from_date and t.occurred_on < @to_date), 0)::bigint as spent
from budgets b
where b.user_id = @user_id and b.deleted_at is null
order by b.category_id nulls first, b.currency, b.id;

-- name: BudgetsForAlerts :many
-- Every active budget with this month's spending in its owner's time zone.
select b.id, b.user_id, b.category_id, b.currency, b.amount,
       u.telegram_id, u.language,
       to_char(now() at time zone u.timezone, 'YYYY-MM')::text as month,
       coalesce((select sum(p.amount) from postings p
                 join transactions t on t.id = p.transaction_id
                 join accounts a on a.id = p.account_id
                 where t.user_id = b.user_id and t.deleted_at is null and a.system_key = 'expense'
                   and p.currency = b.currency
                   and (b.category_id is null or t.category_id = b.category_id)
                   and t.occurred_on >= date_trunc('month', now() at time zone u.timezone)::date
                   and t.occurred_on < (date_trunc('month', now() at time zone u.timezone) + interval '1 month')::date), 0)::bigint as spent
from budgets b
join users u on u.id = b.user_id
where b.deleted_at is null and u.started_at is not null;

-- name: ClaimBudgetAlert :one
-- Returns a row only the first time an alert is claimed.
insert into budget_alerts (budget_id, month, threshold) values (@budget_id, @month, @threshold)
on conflict do nothing
returning budget_id;
