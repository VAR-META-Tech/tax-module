CREATE TABLE IF NOT EXISTS invoice_status_history (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id  UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    from_status VARCHAR(50),
    to_status   VARCHAR(50) NOT NULL,
    reason      TEXT,
    changed_by  VARCHAR(255),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_status_history_invoice_id ON invoice_status_history(invoice_id);
CREATE INDEX idx_status_history_created_at ON invoice_status_history(created_at);
