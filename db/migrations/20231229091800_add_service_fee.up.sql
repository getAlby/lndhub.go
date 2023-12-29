alter table invoices ADD COLUMN service_fee bigint default 0;
alter table invoices ADD COLUMN routing_fee bigint default 0;

-- maybe manually migrate existing data?
-- alter table invoices ALTER COLUMN fee SET DEFAULT 0;
-- update invoices set fee = 0 where fee IS NULL;
-- update invoices set routing_fee = fee where routing_fee=0;
