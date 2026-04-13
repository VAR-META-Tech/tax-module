ALTER TABLE invoices ADD COLUMN transaction_uuid VARCHAR(36);
CREATE UNIQUE INDEX idx_invoices_transaction_uuid ON invoices(transaction_uuid) WHERE transaction_uuid IS NOT NULL;
