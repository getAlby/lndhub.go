CREATE INDEX CONCURRENTLY IF NOT EXISTS index_invoices_on_user_id_settled_at
  ON invoices(user_id, settled_at)
  INCLUDE(amount);
