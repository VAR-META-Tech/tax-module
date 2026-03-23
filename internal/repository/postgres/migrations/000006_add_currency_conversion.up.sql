-- invoices: add currency conversion fields and transaction hash
ALTER TABLE invoices ADD COLUMN original_currency    VARCHAR(10) NOT NULL DEFAULT 'VND';
ALTER TABLE invoices ADD COLUMN exchange_rate         NUMERIC(24,8) NOT NULL DEFAULT 1;
ALTER TABLE invoices ADD COLUMN original_total_amount NUMERIC(24,8) NOT NULL DEFAULT 0;
ALTER TABLE invoices ADD COLUMN original_tax_amount   NUMERIC(24,8) NOT NULL DEFAULT 0;
ALTER TABLE invoices ADD COLUMN original_net_amount   NUMERIC(24,8) NOT NULL DEFAULT 0;
ALTER TABLE invoices ADD COLUMN transaction_hash      VARCHAR(255);

-- invoice_items: store original (pre-conversion) amounts
ALTER TABLE invoice_items ADD COLUMN original_unit_price NUMERIC(24,8) NOT NULL DEFAULT 0;
ALTER TABLE invoice_items ADD COLUMN original_tax_amount NUMERIC(24,8) NOT NULL DEFAULT 0;
ALTER TABLE invoice_items ADD COLUMN original_line_total NUMERIC(24,8) NOT NULL DEFAULT 0;
