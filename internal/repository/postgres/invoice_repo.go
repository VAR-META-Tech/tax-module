package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"tax-module/internal/domain"
)

// InvoiceRepo implements domain.InvoiceRepository using PostgreSQL.
type InvoiceRepo struct {
	pool *pgxpool.Pool
	log  *zerolog.Logger
}

func NewInvoiceRepo(pool *pgxpool.Pool, log *zerolog.Logger) *InvoiceRepo {
	return &InvoiceRepo{pool: pool, log: log}
}

// --- Invoice CRUD ---

func (r *InvoiceRepo) Create(ctx context.Context, invoice *domain.Invoice) error {
	return fmt.Errorf("not implemented") // TODO: Part 4
}

func (r *InvoiceRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	return nil, fmt.Errorf("not implemented") // TODO: Part 4
}

func (r *InvoiceRepo) Update(ctx context.Context, invoice *domain.Invoice) error {
	return fmt.Errorf("not implemented") // TODO: Part 4
}

func (r *InvoiceRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.InvoiceStatus, reason string) error {
	return fmt.Errorf("not implemented") // TODO: Part 4
}

func (r *InvoiceRepo) List(ctx context.Context, filter domain.InvoiceFilter) ([]*domain.Invoice, int64, error) {
	return nil, 0, fmt.Errorf("not implemented") // TODO: Part 4
}

func (r *InvoiceRepo) GetByExternalID(ctx context.Context, externalID string) (*domain.Invoice, error) {
	return nil, fmt.Errorf("not implemented") // TODO: Part 4
}

func (r *InvoiceRepo) GetPendingPolling(ctx context.Context, limit int) ([]*domain.Invoice, error) {
	return nil, fmt.Errorf("not implemented") // TODO: Part 4
}

// --- Items ---

func (r *InvoiceRepo) AddItem(ctx context.Context, item *domain.InvoiceItem) error {
	return fmt.Errorf("not implemented") // TODO: Part 4
}

func (r *InvoiceRepo) GetItemsByInvoiceID(ctx context.Context, invoiceID uuid.UUID) ([]*domain.InvoiceItem, error) {
	return nil, fmt.Errorf("not implemented") // TODO: Part 4
}

func (r *InvoiceRepo) DeleteItem(ctx context.Context, itemID uuid.UUID) error {
	return fmt.Errorf("not implemented") // TODO: Part 4
}

// --- Status history ---

func (r *InvoiceRepo) AddStatusHistory(ctx context.Context, history *domain.InvoiceStatusHistory) error {
	return fmt.Errorf("not implemented") // TODO: Part 4
}

func (r *InvoiceRepo) GetStatusHistory(ctx context.Context, invoiceID uuid.UUID) ([]*domain.InvoiceStatusHistory, error) {
	return nil, fmt.Errorf("not implemented") // TODO: Part 4
}

// --- Audit ---

func (r *InvoiceRepo) AddAuditLog(ctx context.Context, log *domain.AuditLog) error {
	return fmt.Errorf("not implemented") // TODO: Part 4
}
