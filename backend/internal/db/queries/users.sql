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
