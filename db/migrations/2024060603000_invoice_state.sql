CREATE INDEX CONCURRENTLY IF NOT EXISTS index_invoices_on_state
    ON invoices (state);
