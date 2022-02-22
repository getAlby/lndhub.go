CREATE TABLE public.accounts (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    type character varying NOT NULL
);

--bun:split

CREATE SEQUENCE public.accounts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--bun:split

ALTER SEQUENCE public.accounts_id_seq OWNED BY public.accounts.id;

--bun:split

ALTER TABLE ONLY public.accounts ALTER COLUMN id SET DEFAULT nextval('public.accounts_id_seq'::regclass);

--bun:split

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_pkey PRIMARY KEY (id);

--bun:split

CREATE TABLE public.invoices (
    id bigint NOT NULL,
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
    settled_at timestamp with time zone
);

--bun:split

CREATE SEQUENCE public.invoices_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--bun:split

ALTER SEQUENCE public.invoices_id_seq OWNED BY public.invoices.id;

--bun:split

ALTER TABLE ONLY public.invoices ALTER COLUMN id SET DEFAULT nextval('public.invoices_id_seq'::regclass);

--bun:split

ALTER TABLE ONLY public.invoices
    ADD CONSTRAINT invoices_pkey PRIMARY KEY (id);

--bun:split

CREATE TABLE public.transaction_entries (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    invoice_id bigint NOT NULL,
    parent_id bigint,
    credit_account_id bigint NOT NULL,
    debit_account_id bigint NOT NULL,
    amount bigint NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);

--bun:split

CREATE SEQUENCE public.transaction_entries_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--bun:split

ALTER SEQUENCE public.transaction_entries_id_seq OWNED BY public.transaction_entries.id;

--bun:split

ALTER TABLE ONLY public.transaction_entries ALTER COLUMN id SET DEFAULT nextval('public.transaction_entries_id_seq'::regclass);

--bun:split

ALTER TABLE ONLY public.transaction_entries
    ADD CONSTRAINT transaction_entries_pkey PRIMARY KEY (id);

--bun:split

CREATE TABLE public.users (
    id bigint NOT NULL,
    email jsonb,
    login character varying NOT NULL,
    password character varying NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp with time zone
);

--bun:split

CREATE SEQUENCE public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

--bun:split

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;

--bun:split

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);

--bun:split

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_key UNIQUE (email);

--bun:split

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_login_key UNIQUE (login);

--bun:split

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);