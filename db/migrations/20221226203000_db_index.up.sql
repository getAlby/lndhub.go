CREATE INDEX CONCURRENTLY IF NOT EXISTS index_invoices_on_user_id ON invoices(user_id);

--bun:split

CREATE INDEX CONCURRENTLY IF NOT EXISTS index_invoices_on_r_hash ON invoices(r_hash);


--bun:split

CREATE INDEX CONCURRENTLY IF NOT EXISTS index_invoices_on_type_and_r_hash_and_state ON invoices(type, r_hash, state);

--bun:split

CREATE INDEX CONCURRENTLY IF NOT EXISTS index_transaction_entries_on_user_id ON transaction_entries(user_id);

--bun:split

CREATE INDEX CONCURRENTLY IF NOT EXISTS index_transaction_entries_on_invoice_id ON transaction_entries(invoice_id);

--bun:split

CREATE INDEX CONCURRENTLY IF NOT EXISTS index_transaction_entries_on_credit_account_id ON transaction_entries(credit_account_id);

--bun:split

CREATE INDEX CONCURRENTLY IF NOT EXISTS index_transaction_entries_on_debit_account_id ON transaction_entries(debit_account_id);

--bun:split

CREATE INDEX IF NOT EXISTS index_accounts_user_id ON accounts(user_id);
