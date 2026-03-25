package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// InvoiceRepository defines data access for invoices.
type InvoiceRepository interface {
	// Invoice CRUD
	Create(ctx context.Context, invoice *Invoice) error
	GetByID(ctx context.Context, id uuid.UUID) (*Invoice, error)
	Update(ctx context.Context, invoice *Invoice) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status InvoiceStatus, reason string) error
	List(ctx context.Context, filter InvoiceFilter) ([]*Invoice, int64, error)
	GetByExternalID(ctx context.Context, externalID string) (*Invoice, error)
	GetByTransactionUuid(ctx context.Context, transactionUuid string) (*Invoice, error)
	GetPendingPolling(ctx context.Context, limit int) ([]*Invoice, error)

	// Items
	AddItem(ctx context.Context, item *InvoiceItem) error
	UpdateItem(ctx context.Context, item *InvoiceItem) error
	GetItemsByInvoiceID(ctx context.Context, invoiceID uuid.UUID) ([]*InvoiceItem, error)
	DeleteItem(ctx context.Context, itemID uuid.UUID) error

	// Status history
	AddStatusHistory(ctx context.Context, history *InvoiceStatusHistory) error
	GetStatusHistory(ctx context.Context, invoiceID uuid.UUID) ([]*InvoiceStatusHistory, error)

	// Audit
	AddAuditLog(ctx context.Context, log *AuditLog) error
}

// InvoiceFilter holds query parameters for listing invoices.
type InvoiceFilter struct {
	Status   *InvoiceStatus
	FromDate *time.Time
	ToDate   *time.Time
	Limit    int
	Offset   int
}
