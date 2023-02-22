CREATE INDEX CONCURRENTLY IF NOT EXISTS index_invoices_on_expires_at
    ON invoices (expires_at);