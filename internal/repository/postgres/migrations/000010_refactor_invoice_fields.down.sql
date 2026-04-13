-- =============================================
-- Rollback Migration 000010
-- =============================================

-- ============ INVOICE_ITEMS: Drop new columns ============

ALTER TABLE invoice_items DROP COLUMN IF EXISTS item_discount;
ALTER TABLE invoice_items DROP COLUMN IF EXISTS item_total_amount_after_discount;
ALTER TABLE invoice_items DROP COLUMN IF EXISTS item_total_amount_without_tax;

-- ============ INVOICE_ITEMS: Reverse renames ============

ALTER TABLE invoice_items RENAME COLUMN token_line_total TO original_line_total;
ALTER TABLE invoice_items RENAME COLUMN token_tax_amount TO original_tax_amount;
ALTER TABLE invoice_items RENAME COLUMN token_unit_price TO original_unit_price;
ALTER TABLE invoice_items RENAME COLUMN line_number TO sort_order;
ALTER TABLE invoice_items RENAME COLUMN item_total_amount_with_tax TO line_total;
ALTER TABLE invoice_items RENAME COLUMN tax_percentage TO tax_rate;
ALTER TABLE invoice_items RENAME COLUMN item_name TO description;

-- ============ INVOICES: Re-add due_at ============

ALTER TABLE invoices ADD COLUMN due_at TIMESTAMPTZ;

-- ============ INVOICES: Drop new columns ============

ALTER TABLE invoices DROP COLUMN IF EXISTS hbar_amount;
ALTER TABLE invoices DROP COLUMN IF EXISTS exchange_rate_source;
ALTER TABLE invoices DROP COLUMN IF EXISTS erp_order_id;
ALTER TABLE invoices DROP COLUMN IF EXISTS payment_method;
ALTER TABLE invoices DROP COLUMN IF EXISTS buyer_code;
ALTER TABLE invoices DROP COLUMN IF EXISTS buyer_phone;
ALTER TABLE invoices DROP COLUMN IF EXISTS buyer_email;
ALTER TABLE invoices DROP COLUMN IF EXISTS buyer_legal_name;

-- ============ INVOICES: Reverse renames ============

ALTER TABLE invoices RENAME COLUMN token_net_amount TO original_net_amount;
ALTER TABLE invoices RENAME COLUMN token_tax_amount TO original_tax_amount;
ALTER TABLE invoices RENAME COLUMN token_total_amount TO original_total_amount;
ALTER TABLE invoices RENAME COLUMN token_currency TO original_currency;
ALTER TABLE invoices RENAME COLUMN total_amount_without_tax TO net_amount;
ALTER TABLE invoices RENAME COLUMN total_tax_amount TO tax_amount;
ALTER TABLE invoices RENAME COLUMN total_amount_with_tax TO total_amount;
ALTER TABLE invoices RENAME COLUMN buyer_address TO customer_address;
ALTER TABLE invoices RENAME COLUMN buyer_tax_code TO customer_tax_id;
ALTER TABLE invoices RENAME COLUMN buyer_name TO customer_name;
