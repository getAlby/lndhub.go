UPDATE invoices
SET keysend = false
WHERE keysend IS NULL;

ALTER TABLE invoices
ALTER COLUMN keysend SET DEFAULT false;

ALTER TABLE invoices
ALTER COLUMN keysend SET NOT NULL;