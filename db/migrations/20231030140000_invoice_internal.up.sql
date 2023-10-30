UPDATE invoices
SET internal = false
WHERE internal IS NULL;

ALTER TABLE invoices
ALTER COLUMN internal SET NOT NULL;