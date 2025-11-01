CREATE TABLE IF NOT EXISTS users (
    id text UNIQUE DEFAULT gen_random_uuid(),
    first_name text NOT NULL,
    last_name text NOT NULL,
    username text UNIQUE NOT NULL,
    password text NOT NULL,
    income numeric,
    income_rate text,

    PRIMARY KEY (id, username)
);

CREATE TABLE IF NOT EXISTS categories (
    id text UNIQUE DEFAULT gen_random_uuid(),
    user_id text REFERENCES users (id) ON DELETE CASCADE,
    label text NOT NULL,
    current_spending numeric NOT NULL DEFAULT 0,
    allocated_spending numeric NOT NULL CHECK (allocated_spending > 0),

    PRIMARY KEY (id, user_id)
);

CREATE TABLE IF NOT EXISTS items (
    id text UNIQUE,
    user_id text REFERENCES users (id) ON DELETE CASCADE,
    access_token text NOT NULL,
    name text NOT NULL,
    transaction_cursor text NOT NULL default '',

    PRIMARY KEY (id, user_id)
);

CREATE TABLE IF NOT EXISTS accounts (
    id text UNIQUE,
    item_id text REFERENCES items (id) ON DELETE CASCADE,
    name text NOT NULL,
    available_balance numeric,
    current_balance numeric,
    type text,
    sub_type text,

    PRIMARY KEY (id, item_id)
);

CREATE TABLE IF NOT EXISTS transactions (
    id text UNIQUE,
    account_id text REFERENCES accounts (id) ON DELETE CASCADE,
    -- later on have it reference category_id instead once semantic search is figured out
    category_label text,
    authorized_date date NOT NULL default current_date,
    settled_date date,
    merchant text,
    amount numeric NOT NULL default 0,
    currency_code text,
    pending bool NOT NULL DEFAULT false,

    PRIMARY KEY (id, account_id)
);
