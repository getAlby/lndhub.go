alter table invoices ALTER COLUMN fee SET DEFAULT 0;
alter table invoices ADD COLUMN IF NOT EXISTS service_fee bigint default 0;
alter table invoices ADD COLUMN IF NOT EXISTS routing_fee bigint default 0;

-- migrate existing data
update invoices set fee = 0 where fee IS NULL;
update invoices set routing_fee = fee where fee > 0;
