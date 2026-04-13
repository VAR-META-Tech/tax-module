CREATE TABLE IF NOT EXISTS invoices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id     VARCHAR(255),
    status          VARCHAR(50) NOT NULL DEFAULT 'draft',
    customer_name   VARCHAR(255) NOT NULL,
    customer_tax_id VARCHAR(50),
    customer_address TEXT,
    currency        VARCHAR(3) NOT NULL DEFAULT 'VND',
    total_amount    NUMERIC(18,2) NOT NULL DEFAULT 0,
    tax_amount      NUMERIC(18,2) NOT NULL DEFAULT 0,
    net_amount      NUMERIC(18,2) NOT NULL DEFAULT 0,
    notes           TEXT,
    issued_at       TIMESTAMPTZ,
    due_at          TIMESTAMPTZ,
    submitted_at    TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    retry_count     INT NOT NULL DEFAULT 0,
    last_error      TEXT,
    metadata        JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invoices_status ON invoices(status);
CREATE INDEX idx_invoices_external_id ON invoices(external_id);
CREATE INDEX idx_invoices_created_at ON invoices(created_at);
