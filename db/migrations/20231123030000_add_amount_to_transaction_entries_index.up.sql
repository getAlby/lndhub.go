CREATE INDEX CONCURRENTLY IF NOT EXISTS index_transaction_entries_on_credit_account_id_include_amount
    ON public.transaction_entries USING btree
    (credit_account_id ASC) 
    INCLUDE 
    (amount);

CREATE INDEX CONCURRENTLY IF NOT EXISTS index_transaction_entries_on_debit_account_id_include_amount
    ON public.transaction_entries USING btree
    (debit_account_id ASC) 
    INCLUDE 
    (amount);

DROP INDEX IF EXISTS index_transaction_entries_on_debit_account_id;
DROP INDEX IF EXISTS index_transaction_entries_on_credit_account_id;