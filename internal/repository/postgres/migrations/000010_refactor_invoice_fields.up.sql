-- =============================================
-- Migration 000010: Refactor invoice/item fields
--   - Rename columns to match Viettel API naming
--   - Add buyer info fields
--   - Add token tracking fields
--   - Drop due_at
-- =============================================

-- ============ INVOICES: Renames ============

ALTER TABLE invoices RENAME COLUMN customer_name TO buyer_name;
ALTER TABLE invoices RENAME COLUMN customer_tax_id TO buyer_tax_code;
ALTER TABLE invoices RENAME COLUMN customer_address TO buyer_address;
ALTER TABLE invoices RENAME COLUMN total_amount TO total_amount_with_tax;
ALTER TABLE invoices RENAME COLUMN tax_amount TO total_tax_amount;
ALTER TABLE invoices RENAME COLUMN net_amount TO total_amount_without_tax;
ALTER TABLE invoices RENAME COLUMN original_currency TO token_currency;
ALTER TABLE invoices RENAME COLUMN original_total_amount TO token_total_amount;
ALTER TABLE invoices RENAME COLUMN original_tax_amount TO token_tax_amount;
ALTER TABLE invoices RENAME COLUMN original_net_amount TO token_net_amount;

-- ============ INVOICES: New columns ============

ALTER TABLE invoices ADD COLUMN buyer_legal_name     VARCHAR(400);
ALTER TABLE invoices ADD COLUMN buyer_email          VARCHAR(2000);
ALTER TABLE invoices ADD COLUMN buyer_phone          VARCHAR(15);
ALTER TABLE invoices ADD COLUMN buyer_code           VARCHAR(400);
ALTER TABLE invoices ADD COLUMN payment_method       VARCHAR(50);
ALTER TABLE invoices ADD COLUMN erp_order_id         VARCHAR(255);
ALTER TABLE invoices ADD COLUMN exchange_rate_source  VARCHAR(100);
ALTER TABLE invoices ADD COLUMN hbar_amount          NUMERIC(24,8);

-- ============ INVOICES: Drop due_at ============

ALTER TABLE invoices DROP COLUMN IF EXISTS due_at;

-- ============ INVOICE_ITEMS: Renames ============

ALTER TABLE invoice_items RENAME COLUMN description TO item_name;
ALTER TABLE invoice_items RENAME COLUMN tax_rate TO tax_percentage;
ALTER TABLE invoice_items RENAME COLUMN line_total TO item_total_amount_with_tax;
ALTER TABLE invoice_items RENAME COLUMN sort_order TO line_number;
ALTER TABLE invoice_items RENAME COLUMN original_unit_price TO token_unit_price;
ALTER TABLE invoice_items RENAME COLUMN original_tax_amount TO token_tax_amount;
ALTER TABLE invoice_items RENAME COLUMN original_line_total TO token_line_total;

-- ============ INVOICE_ITEMS: New columns ============

ALTER TABLE invoice_items ADD COLUMN item_total_amount_without_tax  NUMERIC(18,2) NOT NULL DEFAULT 0;
ALTER TABLE invoice_items ADD COLUMN item_total_amount_after_discount NUMERIC(18,2);
ALTER TABLE invoice_items ADD COLUMN item_discount                   NUMERIC(18,2);
