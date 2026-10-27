-- name: RecordPayment :one
-- Telegram may deliver a payment twice; the charge id makes it count once.
insert into payments (user_id, telegram_charge_id, payload, stars, recurring, premium_until)
values (@user_id, @telegram_charge_id, @payload, @stars, @recurring, @premium_until)
on conflict (telegram_charge_id) do nothing
returning id;

-- name: ExtendPremium :one
update users set premium_until = greatest(coalesce(premium_until, now()), @until::timestamptz)
where id = @id
returning premium_until;

-- name: StartTrial :one
-- Once per user: a 7-day trial that never shortens paid Premium.
update users
set trial_started_at = now(), premium_until = greatest(coalesce(premium_until, now()), now() + interval '7 days')
where id = @id and trial_started_at is null
returning premium_until;

-- name: LoggedDays :one
select count(distinct occurred_on)::int from transactions
where user_id = @user_id and deleted_at is null and type not in ('opening');

-- name: CountActiveBudgets :one
select count(*)::int from budgets where user_id = @user_id and deleted_at is null;
