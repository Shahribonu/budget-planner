-- name: ListRecurring :many
select * from recurring where user_id = @user_id and deleted_at is null order by kind, day_of_month, id;

-- name: InsertRecurring :one
insert into recurring (user_id, kind, title, category_id, currency, amount, day_of_month, account_id)
values (@user_id, @kind, @title, @category_id, @currency, @amount, @day_of_month, @account_id)
returning *;

-- name: UpdateRecurring :one
update recurring
set title = coalesce(sqlc.narg(title), title),
    amount = coalesce(sqlc.narg(amount), amount),
    day_of_month = coalesce(sqlc.narg(day_of_month), day_of_month)
where id = @id and user_id = @user_id and deleted_at is null
returning *;

-- name: SoftDeleteRecurring :execrows
update recurring set deleted_at = now() where id = @id and user_id = @user_id and deleted_at is null;

-- name: GetMonthlyPlan :one
select * from monthly_plans where user_id = @user_id and month = @month and currency = @currency;

-- name: InsertMonthlyPlan :one
insert into monthly_plans (user_id, month, currency) values (@user_id, @month, @currency)
on conflict (user_id, month, currency) do update set month = excluded.month
returning *;

-- name: UpdateMonthlyPlan :one
update monthly_plans
set income_override = case when @set_income::bool then sqlc.narg(income_override)::bigint else income_override end,
    income_day      = case when @set_income_day::bool then sqlc.narg(income_day)::int else income_day end,
    savings         = coalesce(sqlc.narg(savings), savings),
    buffer_override = case when @set_buffer::bool then sqlc.narg(buffer_override)::bigint else buffer_override end
where id = @id and user_id = @user_id
returning *;

-- name: ListPlanItems :many
select * from plan_items where plan_id = @plan_id order by created_at, id;

-- name: InsertPlanItem :one
insert into plan_items (plan_id, title, category_id, amount_min, amount_max, priority, due_on, postponed_count, carried_from, note)
values (@plan_id, @title, @category_id, @amount_min, @amount_max, @priority, @due_on, @postponed_count, @carried_from, @note)
returning *;

-- name: GetPlanItem :one
select pi.* from plan_items pi join monthly_plans mp on mp.id = pi.plan_id
where pi.id = @id and mp.user_id = @user_id;

-- name: UpdatePlanItem :one
update plan_items
set title          = coalesce(sqlc.narg(title), title),
    category_id    = case when @set_category::bool then sqlc.narg(category_id)::bigint else category_id end,
    amount_min     = coalesce(sqlc.narg(amount_min), amount_min),
    amount_max     = coalesce(sqlc.narg(amount_max), amount_max),
    priority       = coalesce(sqlc.narg(priority), priority),
    due_on         = case when @set_due::bool then sqlc.narg(due_on)::date else due_on end,
    status         = coalesce(sqlc.narg(status), status),
    transaction_id = case when @set_tx::bool then sqlc.narg(transaction_id)::bigint else transaction_id end,
    note           = coalesce(sqlc.narg(note), note)
where plan_items.id = @id and plan_items.plan_id in (select mp.id from monthly_plans mp where mp.user_id = @user_id)
returning *;

-- name: DeletePlanItem :execrows
delete from plan_items where plan_items.id = @id and plan_items.plan_id in (select mp.id from monthly_plans mp where mp.user_id = @user_id);

-- name: UnfinishedItemsOfMonth :many
-- Items still planned or postponed in a month, to carry into the next one.
select pi.* from plan_items pi join monthly_plans mp on mp.id = pi.plan_id
where mp.user_id = @user_id and mp.month = @month and mp.currency = @currency
  and pi.status in ('planned', 'postponed')
  and not exists (select 1 from plan_items c where c.carried_from = pi.id);

-- name: MarkPostponed :exec
update plan_items set status = 'postponed' where id = @id and status = 'planned';

-- name: IncomeByMonth :many
-- Income per month over a range, for a conservative estimate when there is no recurring salary.
select date_trunc('month', t.occurred_on)::date as month, sum(-p.amount)::bigint as amount
from postings p
join transactions t on t.id = p.transaction_id
join accounts a on a.id = p.account_id
where t.user_id = @user_id and t.deleted_at is null and a.system_key = 'income' and p.currency = @currency
  and t.occurred_on >= @from_date and t.occurred_on < @to_date
group by 1 order by 1;

-- name: EverydaySpending :one
-- Spending on groceries, cafes, transport, taxi and fuel over a range.
select coalesce(sum(p.amount), 0)::bigint
from postings p
join transactions t on t.id = p.transaction_id
join accounts a on a.id = p.account_id
join categories c on c.id = t.category_id
where t.user_id = @user_id and t.deleted_at is null and a.system_key = 'expense' and p.currency = @currency
  and c.key in ('groceries', 'cafe', 'transport', 'taxi', 'fuel')
  and t.occurred_on >= @from_date and t.occurred_on < @to_date;

-- name: FirstTransactionDate :one
select coalesce(min(occurred_on), current_date)::date from transactions where user_id = @user_id and deleted_at is null;

-- name: OpenPlanItemsForMatch :many
-- Planned items in the month of a purchase, to ask whether the purchase was one of them.
select pi.* from plan_items pi join monthly_plans mp on mp.id = pi.plan_id
where mp.user_id = @user_id and mp.currency = @currency and pi.status = 'planned'
  and mp.month = date_trunc('month', @day::date)::date
  and pi.transaction_id is null;

-- name: SetChatMode :exec
update users set chat_mode = sqlc.narg(mode), chat_mode_until = sqlc.narg(until) where id = @id;

-- name: UsersDueForPlanPrompt :many
-- On the 25th or later, once per upcoming month, for users who log regularly enough to plan.
select u.* from users u
where u.started_at is not null and u.notify_enabled
  and extract(day from now() at time zone u.timezone) >= 25
  and (now() at time zone u.timezone)::time >= time '10:00'
  and (u.plan_prompted_month is null
       or u.plan_prompted_month < (date_trunc('month', now() at time zone u.timezone) + interval '1 month')::date)
  and exists (select 1 from transactions t where t.user_id = u.id and t.deleted_at is null
              and t.created_at > now() - interval '14 days');

-- name: MarkPlanPrompted :exec
update users set plan_prompted_month = @month where id = @id;

-- name: GetMonthlyPlanByID :one
select * from monthly_plans where id = @id and user_id = @user_id;
