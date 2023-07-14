alter table transaction_entries drop constraint unique_tx_entry_tuple,
add constraint unique_tx_entry_tuple UNIQUE(user_id, invoice_id, debit_account_id, credit_account_id, entry_type);