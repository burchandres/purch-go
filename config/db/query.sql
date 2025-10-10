-----------------------
-- User related queries
-----------------------

-- name: GetUserById :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByUsername :one
SELECT * FROM users
WHERE username = $1 LIMIT 1;

-- name: StoreUser :one
INSERT INTO users (
    first_name, 
    last_name,
    username,
    password,
    is_active,
    income,
    income_rate
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
    set first_name = $2,
    last_name = $3,
    username = $4,
    password = $5,
    is_active = $6,
    income = $7,
    income_rate = $8
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1

-- name: GetUserCategories :many
SELECT * FROM categories
WHERE user_id = $1;

-- name: GetUserItems :many
SELECT * FROM items
WHERE user_id = $1;

-- name: GetUserAccounts :many
SELECT a.*
FROM items i WHERE i.user_id = $1
JOIN accounts a ON i.id = a.item_id;

-- name: GetUserTransactions :many
SELECT t.*
FROM items i WHERE i.user_id = $1
JOIN accounts a ON i.id = a.item_id
JOIN transactions t ON a.id = t.account_id;

-----------------------------
-- Categories related queries
-----------------------------

-- name: GetCategory :one
SELECT * FROM categories
WHERE id = $1 LIMIT 1;

-- name: StoreCategory :one
INSERT INTO categories (
    user_id,
    label,
    allocated_spending
) VALUES (
    $1, $2, $3
)
RETURNING *;


-- name: UpdateCategory :one
UPDATE categories
    set current_spending = $2,
    allocated_spending = $3
WHERE id = $1
RETURNING *;

-- name: DeleteCategory :exec
DELETE FROM categories WHERE id = $1;

-----------------------
-- Items related queries
-----------------------

-- name: GetItem :one
SELECT * FROM items
WHERE id = $1;

-- name: StoreItem :one
INSERT INTO items (
    user_id,
    access_token
    name
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: UpdateItem :one
UPDATE items
    set name = $2,
    transaction_cursor = $3
WHERE id = $1
RETURNING *;

-- name: DeleteItem :exec
DELETE FROM items WHERE id = $1;

-- name: GetItemAccounts :many
SELECT * FROM accounts
WHERE item_id = $1;

---------------------------
-- Accounts related queries
---------------------------

-- name: GetAccount :one
SELECT * FROM accounts
WHERE id = $1;

-- name: StoreAccount :one
INSERT INTO accounts (
    item_id,
    name
) VALUES (
    $1, $2
)
RETURNING *;

-- name: UpdateAccount :one
UPDATE accounts
    set name = $2
WHERE id = $1
RETURNING *;

-- name: GetAccountTransactions :one
SELECT t.*
FROM transactions t
JOIN accounts a ON t.account_id = a.id;

-------------------------------
-- Transactions related queries
-------------------------------

-- name: GetTransaction :one
SELECT * FROM transactions
WHERE id = $1;

-- name: StoreTransaction :one
INSERT INTO transactions (
    account_id,
    category_id,
    authorized_date,
    merchant,
    amount,
    currency_code,
    pending
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: UpdateTransaction :one
UPDATE transactions
    set settled_date = $2,
    amount = $3,
    pending = $4
WHERE id = $1
RETURNING *;

-- name: DeleteTransaction :exec
DELETE FROM transactions WHERE id = $1;
