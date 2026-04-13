DROP INDEX IF EXISTS idx_invoices_transaction_uuid;
ALTER TABLE invoices DROP COLUMN IF EXISTS transaction_uuid;
