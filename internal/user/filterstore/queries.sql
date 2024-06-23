-- name: get-user-filters
SELECT * from user_filters where user_id = $1 and page = $2;