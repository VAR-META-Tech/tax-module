ALTER TABLE invoices DROP COLUMN IF EXISTS original_currency;
ALTER TABLE invoices DROP COLUMN IF EXISTS exchange_rate;
ALTER TABLE invoices DROP COLUMN IF EXISTS original_total_amount;
ALTER TABLE invoices DROP COLUMN IF EXISTS original_tax_amount;
ALTER TABLE invoices DROP COLUMN IF EXISTS original_net_amount;
ALTER TABLE invoices DROP COLUMN IF EXISTS transaction_hash;

ALTER TABLE invoice_items DROP COLUMN IF EXISTS original_unit_price;
ALTER TABLE invoice_items DROP COLUMN IF EXISTS original_tax_amount;
ALTER TABLE invoice_items DROP COLUMN IF EXISTS original_line_total;
