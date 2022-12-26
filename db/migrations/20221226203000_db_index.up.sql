CREATE INDEX CONCURRENTLY IF NOT EXISTS index_invoices_on_user_id ON invoices(user_id);

--migration:splitt

CREATE INDEX CONCURRENTLY IF NOT EXISTS index_invoices_on_payment_request ON invoices(payment_request);

--migration:split

CREATE INDEX CONCURRENTLY IF NOT EXISTS index_invoices_on_type ON invoices(type);

--migration:split

CREATE INDEX CONCURRENTLY IF NOT EXISTS index_invoices_on_state ON invoices(state);

--migration:split

CREATE INDEX CONCURRENTLY IF NOT EXISTS index_invoices_on_r_hash ON invoices(r_hash);

--migration:split

CREATE INDEX CONCURRENTLY IF NOT EXISTS index_invoices_on_created_at ON invoices(created_at);

--migration:split

CREATE INDEX CONCURRENTLY IF NOT EXISTS index_invoices_on_expires_at ON invoices(expires_at)

--migration:split

CREATE INDEX CONCURRENTLY IF NOT EXISTS index_invoices_on_settled_at ON invoices(settled_at)

--migration:split

CREATE INDEX CONCURRENTLY IF NOT EXISTS index_invoices_on_type_and_r_hash_and_state ON invoices(type, r_hash, state);

--migration:split

CREATE INDEX CONCURRENTLY IF NOT EXISTS index_invoices_on_user_id_and_invoice_type_and_state_and_created_at ON invoices(user_id, invoice_type, state, created_at);

--migration:split

CREATE INDEX CONCURRENTLY IF NOT EXISTS index_transaction_entries_on_user_id ON transaction_entries(user_id);

--migration:split

CREATE INDEX CONCURRENTLY IF NOT EXISTS index_transaction_entries_on_invoice_id ON transaction_entries(invoice_id);

--migration:split

CREATE INDEX CONCURRENTLY IF NOT EXISTS index_transaction_entries_on_credit_account_id ON transaction_entries(credit_account_id);

--migration:split

CREATE INDEX CONCURRENTLY IF NOT EXISTS index_transaction_entries_on_debit_account_id ON transaction_entries(debit_account_id);

--migration:split


CREATE INDEX IF NOT EXISTS index_accounts_user_id ON accounts(user_id);

--migration:split

CREATE INDEX IF NOT EXISTS index_accounts_type ON accounts(type);
