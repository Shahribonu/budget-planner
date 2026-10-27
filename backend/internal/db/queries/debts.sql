-- name: CreateLiabilityAccount :one
insert into accounts (user_id, kind, name, currency)
values (@user_id, @kind, @name, @currency)
returning *;

-- name: RenameAccount :exec
update accounts set name = @name where id = @id and user_id = @user_id;

-- name: InsertDebt :one
insert into debts (user_id, direction, counterparty, account_id, currency, due_on, note)
values (@user_id, @direction, @counterparty, @account_id, @currency, @due_on, @note)
returning *;

-- name: ListDebts :many
-- balance: what is still owed, positive for lent (receivable) and negative for borrowed (payable).
select d.*, coalesce((select sum(p.amount) from postings p join transactions t on t.id = p.transaction_id
                      where p.account_id = d.account_id and t.deleted_at is null), 0)::bigint as balance
from debts d
where d.user_id = @user_id and d.deleted_at is null
order by d.created_at desc, d.id desc;

-- name: GetDebt :one
select d.*, coalesce((select sum(p.amount) from postings p join transactions t on t.id = p.transaction_id
                      where p.account_id = d.account_id and t.deleted_at is null), 0)::bigint as balance
from debts d
where d.id = @id and d.user_id = @user_id and d.deleted_at is null;

-- name: UpdateDebt :one
update debts
set counterparty = coalesce(sqlc.narg(counterparty), counterparty),
    note         = coalesce(sqlc.narg(note), note),
    due_on       = case when @set_due::bool then sqlc.narg(due_on)::date else due_on end
where id = @id and user_id = @user_id and deleted_at is null
returning *;

-- name: SoftDeleteDebt :exec
update debts set deleted_at = now() where id = @id and user_id = @user_id;

-- name: SoftDeleteAccountTransactions :exec
update transactions set deleted_at = now(), updated_at = now()
where user_id = @user_id and deleted_at is null
  and id in (select transaction_id from postings where account_id = @account_id);

-- name: InsertPlan :one
insert into installment_plans (user_id, title, provider, category_id, account_id, currency, total, months, pay_from_account_id)
values (@user_id, @title, @provider, @category_id, @account_id, @currency, @total, @months, @pay_from_account_id)
returning *;

-- name: InsertScheduleRow :exec
insert into installment_schedule (plan_id, n, due_on, amount) values (@plan_id, @n, @due_on, @amount);

-- name: DebtAccount :one
select account_id from debts where id = @id and user_id = @user_id and deleted_at is null;

-- name: PlanAccount :one
select account_id from installment_plans where id = @id and user_id = @user_id and deleted_at is null;

-- name: ListPlans :many
-- paid: total minus what is still owed (the payable balance is negative while owed).
select ip.*, (ip.total + coalesce((select sum(p.amount) from postings p join transactions t on t.id = p.transaction_id
                                   where p.account_id = ip.account_id and t.deleted_at is null), 0))::bigint as paid
from installment_plans ip
where ip.user_id = @user_id and ip.deleted_at is null
order by ip.created_at desc, ip.id desc;

-- name: GetPlan :one
select ip.*, (ip.total + coalesce((select sum(p.amount) from postings p join transactions t on t.id = p.transaction_id
                                   where p.account_id = ip.account_id and t.deleted_at is null), 0))::bigint as paid
from installment_plans ip
where ip.id = @id and ip.user_id = @user_id and ip.deleted_at is null;

-- name: ListSchedule :many
select * from installment_schedule where plan_id = any(@plan_ids::bigint[]) order by plan_id, n;

-- name: SoftDeletePlan :exec
update installment_plans set deleted_at = now() where id = @id and user_id = @user_id;

-- name: LiabilityTotals :many
-- Receivable (owed to the user, positive) and payable (owed by the user, negative) per
-- currency, with installment plans apart from debts.
select a.kind, (ip.id is not null)::bool as installment, p.currency, sum(p.amount)::bigint as amount
from postings p
join transactions t on t.id = p.transaction_id
join accounts a on a.id = p.account_id
left join installment_plans ip on ip.account_id = a.id
where a.user_id = @user_id and a.kind in ('receivable', 'payable') and t.deleted_at is null
group by a.kind, (ip.id is not null), p.currency
having sum(p.amount) <> 0
order by 1, 2, 3;
