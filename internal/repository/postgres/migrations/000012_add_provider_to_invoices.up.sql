ALTER TABLE invoices
    ADD COLUMN provider VARCHAR(20) NOT NULL DEFAULT 'viettel';

CREATE INDEX idx_invoices_provider ON invoices(provider);
