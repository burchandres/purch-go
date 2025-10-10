CREATE TABLE users (
    id bigserial,
    first_name text NOT NULL,
    last_name text NOT NULL,
    username text NOT NULL,
    password text NOT NULL,
    is_active bool NOT NULL DEFAULT false,
    income money,
    income_rate text,

    PRIMARY KEY (id, username)
);

CREATE TABLE categories (
    id bigserial,
    user_id bigint REFERENCES users (id) ON DELETE CASCADE,
    label text NOT NULL,
    current_spending numeric NOT NULL DEFAULT 0,
    allocated_spending numeric NOT NULL CHECK (allocated_spending > 0),

    PRIMARY KEY (id, user_id)
);

CREATE TABLE items (
    id text,
    user_id bigint REFERENCES users (id) ON DELETE CASCADE,
    access_token text NOT NULL,
    name text NOT NULL,
    transaction_cursor text NOT NULL default '',

    PRIMARY KEY (id, user_id)
);

CREATE TABLE accounts (
    id text,
    item_id text REFERENCES items (id) ON DELETE CASCADE,
    name text NOT NULL,

    PRIMARY KEY (id, item_id)
);

CREATE TABLE transactions (
    id text,
    account_id text REFERENCES accounts (id) ON DELETE CASCADE,
    category_id text REFERENCES categories (id),
    authorized_date date NOT NULL default current_date,
    settled_date date,
    merchant text,
    amount money NOT NULL,
    currency_code text,
    pending bool NOT NULL DEFAULT false,

    PRIMARY KEY (id, account_id, category_id)
);