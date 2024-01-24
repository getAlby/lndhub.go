CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    pubkey character varying NOT NULL UNIQUE,
    password character varying NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone
);
--bun:split
CREATE TABLE assets (
    id SERIAL PRIMARY KEY,
    ta_asset_id character varying NOT NULL,
    asset_name character varying NOT NULL,
    asset_type int DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone
);
--bun:split
INSERT INTO assets(id, ta_asset_id, asset_name) SELECT 1, 'BTC', 'bitcoin' WHERE NOT EXISTS (SELECT id FROM assets WHERE id = 1);
--bun:split

CREATE TABLE accounts (
    id SERIAL PRIMARY KEY,
    user_id bigint NOT NULL,
    asset_id bigint NOT NULL,
    type character varying NOT NULL,
    CONSTRAINT fk_user
        FOREIGN KEY(user_id)
        REFERENCES users(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_asset
        FOREIGN KEY(asset_id)
        REFERENCES assets(id)
        ON DELETE NO ACTION
);
--bun:split
CREATE TABLE invoices (
    id SERIAL PRIMARY KEY,
    type character varying,
    user_id bigint,
    asset_id bigint,
    amount bigint,
    memo character varying,
    description_hash character varying,
    payment_request character varying,
    destination_pubkey_hex character varying NOT NULL,
    r_hash character varying,
    preimage character varying,
    internal boolean,
    state character varying DEFAULT 'initialized',
    error_message character varying,
    add_index bigint,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    expires_at timestamp with time zone,
    updated_at timestamp with time zone,
    settled_at timestamp with time zone,
    CONSTRAINT fk_user
        FOREIGN KEY(user_id)
        REFERENCES users(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_asset
        FOREIGN KEY(asset_id)
        REFERENCES assets(id)
        ON DELETE NO ACTION
);
--bun:split
CREATE TABLE transaction_entries (
    id SERIAL PRIMARY KEY,
    user_id bigint NOT NULL,
    invoice_id bigint NOT NULL,
    parent_id bigint,
    credit_account_id bigint NOT NULL,
    debit_account_id bigint NOT NULL,
    amount bigint NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT fk_user
        FOREIGN KEY(user_id)
        REFERENCES users(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_credit_account
        FOREIGN KEY(credit_account_id)
        REFERENCES accounts(id)
        ON DELETE CASCADE,
    CONSTRAINT fk_debit_account
        FOREIGN KEY(debit_account_id)
        REFERENCES accounts(id)
        ON DELETE CASCADE
);
