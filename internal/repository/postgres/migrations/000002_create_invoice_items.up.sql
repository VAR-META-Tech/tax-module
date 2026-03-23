CREATE TABLE IF NOT EXISTS invoice_items (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id  UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    description VARCHAR(500) NOT NULL,
    quantity    NUMERIC(12,3) NOT NULL DEFAULT 1,
    unit_price  NUMERIC(18,2) NOT NULL,
    tax_rate    NUMERIC(5,2) NOT NULL DEFAULT 0,
    tax_amount  NUMERIC(18,2) NOT NULL DEFAULT 0,
    line_total  NUMERIC(18,2) NOT NULL DEFAULT 0,
    sort_order  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invoice_items_invoice_id ON invoice_items(invoice_id);
