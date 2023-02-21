CREATE INDEX CONCURRENTLY IF NOT EXISTS index_invoices_on_id_user_id_type_state
    ON invoices (id, user_id, type, state);