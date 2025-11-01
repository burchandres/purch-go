-- initialiaze the db with a test user 
INSERT INTO users (
    first_name,
    last_name,
    username,
    password,
    income,
    income_rate
) VALUES ('test', 'user', 'testuser', '$2a$10$AQOLxQTZS/V3XS/3qFkUxe3RQ6UqqOW/a2zuxJAN0Nhpe5zUVnoSu', 1000, 'weekly')
RETURNING *;

-- some categories
INSERT INTO categories (
    user_id,
    label,
    current_spending,
    allocated_spending
) 
SELECT (SELECT id FROM users LIMIT 1), 'Groceries', 50, 350
UNION ALL
SELECT (SELECT id FROM users LIMIT 1), 'Dining', 150, 350
UNION ALL
SELECT (SELECT id FROM users LIMIT 1), 'Entertainment', 50, 100
RETURNING *;

-- some items for the user
INSERT INTO items (
    id,
    user_id,
    access_token,
    name,
    transaction_cursor
)
SELECT 'C1-id', (SELECT id FROM users LIMIT 1), 'C1-token', 'Capital One', 'C1-cursor'
UNION ALL
SELECT 'WF-id', (SELECT id FROM users LIMIT 1), 'WF-token', 'Wells Fargo', 'WF-cursor'
RETURNING *;

-- some accounts
INSERT INTO accounts (
    id,
    item_id,
    name,
    available_balance,
    current_balance,
    type,
    sub_type
)
SELECT 'C1-id-checking', (SELECT id FROM items WHERE name='Capital One'), 'Young Adult Checking', 100, 150, 'Depository', 'Checking'
UNION ALL
SELECT 'WF-id-credit-card', (SELECT id FROM items WHERE name='Wells Fargo'), 'Active Cash Credit Card', 100, 900, 'Credit', ''
RETURNING *;

-- some transactions under the credit card
INSERT INTO transactions (
    id,
    account_id,
    category_label,
    authorized_date,
    settled_date,
    merchant,
    amount,
    currency_code,
    pending
)
SELECT 'tx1-id', (SELECT id FROM accounts WHERE name='Active Cash Credit Card'), 'Groceries', now(), now(), 'Trader Joes', 36.63, '$', false
UNION ALL
SELECT 'tx2-id', (SELECT id FROM accounts WHERE name='Active Cash Credit Card'), 'Gas', now(), now(), 'Exxon Gas', 42.46, '$', false
UNION ALL
SELECT 'tx3-id', (SELECT id FROM accounts WHERE name='Active Cash Credit Card'), 'Dining', now(), now(), 'Rocklands', 23.76, '$', false
RETURNING *;