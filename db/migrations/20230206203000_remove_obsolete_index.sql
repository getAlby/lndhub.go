--bun:split

DROP INDEX IF EXISTS index_transaction_entries_on_user_id ON transaction_entries(user_id);