-- name: ListCategories :many
select * from categories
where (user_id is null or user_id = @user_id) and archived_at is null
order by kind, sort, id;

-- name: GetCategory :one
select * from categories
where id = @id and (user_id is null or user_id = @user_id) and archived_at is null;
