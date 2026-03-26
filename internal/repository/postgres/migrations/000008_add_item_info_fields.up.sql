-- invoice_items: add Viettel ItemInfo fields
ALTER TABLE invoice_items ADD COLUMN selection           INT;
ALTER TABLE invoice_items ADD COLUMN item_type           INT;
ALTER TABLE invoice_items ADD COLUMN item_code           VARCHAR(50);
ALTER TABLE invoice_items ADD COLUMN unit_code           VARCHAR(100);
ALTER TABLE invoice_items ADD COLUMN unit_name           VARCHAR(300);
ALTER TABLE invoice_items ADD COLUMN discount            NUMERIC(12,4) NOT NULL DEFAULT 0;
ALTER TABLE invoice_items ADD COLUMN discount2           NUMERIC(12,4) NOT NULL DEFAULT 0;
ALTER TABLE invoice_items ADD COLUMN item_note           VARCHAR(300);
ALTER TABLE invoice_items ADD COLUMN is_increase_item    BOOLEAN;
ALTER TABLE invoice_items ADD COLUMN batch_no            VARCHAR(300);
ALTER TABLE invoice_items ADD COLUMN exp_date            VARCHAR(50);
ALTER TABLE invoice_items ADD COLUMN adjust_ratio        VARCHAR(10);
ALTER TABLE invoice_items ADD COLUMN unit_price_with_tax NUMERIC(18,2);
ALTER TABLE invoice_items ADD COLUMN special_info        JSONB;
