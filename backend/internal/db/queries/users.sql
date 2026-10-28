-- name: UpsertUser :one
insert into users (telegram_id, first_name, username, language)
values (@telegram_id, @first_name, @username, @language)
on conflict (telegram_id) do update
    set first_name = excluded.first_name,
        username   = excluded.username
returning *;

-- name: GetUser :one
select * from users where id = @id;

-- name: UpdateUserSettings :one
update users
set language      = coalesce(sqlc.narg(language), language),
    main_currency = coalesce(sqlc.narg(main_currency), main_currency),
    timezone      = coalesce(sqlc.narg(timezone), timezone)
where id = @id
returning *;

-- name: UserHasAssetAccount :one
select exists (select 1 from accounts where user_id = @user_id and kind = 'asset');

-- name: AcceptTerms :one
update users set terms_accepted_at = now(), terms_version = @version where id = @id
returning *;

-- name: ClaimExport :one
-- Claims the export slot unless one was taken within the cooldown; no row means throttled.
update users set last_export_at = now()
where id = @id and (last_export_at is null or last_export_at < now() - make_interval(secs => @cooldown_secs::int))
returning telegram_id, language;

-- name: DeleteUserDrafts :exec
-- Postings and drafts reference accounts and transactions without a cascade,
-- so the ledger goes first: DeleteUserDrafts, DeleteUserTransactions, DeleteUser.
delete from drafts where user_id = @id;

-- name: DeleteUserTransactions :exec
delete from transactions where user_id = @id;

-- name: DeleteUser :exec
-- Cascades to every other row the user owns. Run after DeleteUserLedger.
delete from users where id = @id;

-- name: PurgeOldDrafts :execrows
-- Drafts keep the raw message text; drop resolved ones after a week.
delete from drafts where status <> 'pending' and created_at < now() - interval '7 days';

-- name: ReleaseExport :exec
update users set last_export_at = null where id = @id;
