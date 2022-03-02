CREATE TABLE public.users (
    id SERIAL PRIMARY KEY,
    email character varying UNIQUE,
    login character varying NOT NULL UNIQUE,
    password character varying NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone
);

--bun:split

CREATE TABLE public.invoices (
    id SERIAL PRIMARY KEY,
    type character varying,
    user_id bigint,
    amount bigint,
    memo character varying,
    description_hash character varying,
    payment_request character varying,
    destination_pubkey_hex character varying NOT NULL,
    r_hash character varying,
    preimage character varying,
    internal boolean,
    state character varying DEFAULT 'initialized'::character varying,
    error_message character varying,
    add_index bigint,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    expires_at timestamp with time zone,
    updated_at timestamp with time zone,
    settled_at timestamp with time zone,
    CONSTRAINT fk_user
        FOREIGN KEY(user_id) 
        REFERENCES users(id)
        ON DELETE CASCADE
);

--bun:split

CREATE TABLE public.accounts (
    id SERIAL PRIMARY KEY,
    user_id bigint NOT NULL,
    type character varying NOT NULL,
    CONSTRAINT fk_user
        FOREIGN KEY(user_id) 
        REFERENCES users(id)
        ON DELETE CASCADE
);

--bun:split

CREATE TABLE public.transaction_entries (
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
