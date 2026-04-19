DROP INDEX IF EXISTS idx_invoices_provider;
ALTER TABLE invoices DROP COLUMN IF EXISTS provider;
