package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"tax-module/internal/domain"
)

// --- Mock publisher ---

type mockPublisher struct {
	reportToAuthorityFn func(ctx context.Context, transactionUuid, startDate, endDate string) (int, int, error)
}

func (m *mockPublisher) CreateInvoice(_ context.Context, _ *domain.Invoice) (string, error) {
	return "", nil
}

func (m *mockPublisher) QueryStatus(_ context.Context, _ string) (string, []byte, error) {
	return "", nil, nil
}

func (m *mockPublisher) ReportToAuthority(ctx context.Context, transactionUuid, startDate, endDate string) (int, int, error) {
	if m.reportToAuthorityFn != nil {
		return m.reportToAuthorityFn(ctx, transactionUuid, startDate, endDate)
	}
	return 1, 0, nil
}

func (m *mockPublisher) DownloadInvoiceFile(_ context.Context, _ string, _ string) (string, error) {
	return "", nil
}

// --- Mock repo ---

type mockInvoiceRepo struct {
	invoice *domain.Invoice
}

func (r *mockInvoiceRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Invoice, error) {
	if r.invoice != nil && r.invoice.ID == id {
		return r.invoice, nil
	}
	return nil, domain.NewNotFoundError("invoice not found")
}

func (r *mockInvoiceRepo) Create(_ context.Context, _ *domain.Invoice) error { return nil }
func (r *mockInvoiceRepo) CreateWithItems(_ context.Context, _ *domain.Invoice) error {
	return nil
}
func (r *mockInvoiceRepo) Update(_ context.Context, _ *domain.Invoice) error              { return nil }
func (r *mockInvoiceRepo) List(_ context.Context, _ domain.InvoiceFilter) ([]*domain.Invoice, int64, error) {
	return nil, 0, nil
}
func (r *mockInvoiceRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ domain.InvoiceStatus, _ string) error {
	return nil
}
func (r *mockInvoiceRepo) AddItem(_ context.Context, _ *domain.InvoiceItem) error    { return nil }
func (r *mockInvoiceRepo) UpdateItem(_ context.Context, _ *domain.InvoiceItem) error { return nil }
func (r *mockInvoiceRepo) DeleteItem(_ context.Context, _ uuid.UUID) error           { return nil }
func (r *mockInvoiceRepo) GetItemsByInvoiceID(_ context.Context, _ uuid.UUID) ([]*domain.InvoiceItem, error) {
	return nil, nil
}
func (r *mockInvoiceRepo) GetByExternalID(_ context.Context, _ string) (*domain.Invoice, error) {
	return nil, nil
}
func (r *mockInvoiceRepo) GetByTransactionUuid(_ context.Context, txnUuid string) (*domain.Invoice, error) {
	if r.invoice != nil && r.invoice.TransactionUuid != nil && *r.invoice.TransactionUuid == txnUuid {
		return r.invoice, nil
	}
	return nil, domain.NewNotFoundError("invoice not found")
}
func (r *mockInvoiceRepo) GetPendingPolling(_ context.Context, _ int) ([]*domain.Invoice, error) {
	return nil, nil
}
func (r *mockInvoiceRepo) AddStatusHistory(_ context.Context, _ *domain.InvoiceStatusHistory) error {
	return nil
}
func (r *mockInvoiceRepo) GetStatusHistory(_ context.Context, _ uuid.UUID) ([]*domain.InvoiceStatusHistory, error) {
	return nil, nil
}
func (r *mockInvoiceRepo) AddAuditLog(_ context.Context, _ *domain.AuditLog) error { return nil }
func (r *mockInvoiceRepo) UpdateTransactionHash(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

// --- Tests ---

func TestReportToAuthority_Success(t *testing.T) {
	log := zerolog.Nop()
	svc := NewInvoiceService(
		&mockInvoiceRepo{},
		&mockPublisher{reportToAuthorityFn: func(_ context.Context, _, _, _ string) (int, int, error) {
			return 1, 0, nil
		}},
		nil,
		&log,
	)

	successCount, errorCount, err := svc.ReportToAuthority(context.Background(), "txn-abc-123", "2026-03-01", "2026-03-31")
	if err != nil {
		t.Fatalf("ReportToAuthority: %v", err)
	}
	if successCount != 1 {
		t.Errorf("successCount = %d, want 1", successCount)
	}
	if errorCount != 0 {
		t.Errorf("errorCount = %d, want 0", errorCount)
	}
}

func TestReportToAuthority_PublisherError(t *testing.T) {
	log := zerolog.Nop()
	svc := NewInvoiceService(
		&mockInvoiceRepo{},
		&mockPublisher{reportToAuthorityFn: func(_ context.Context, _, _, _ string) (int, int, error) {
			return 0, 0, domain.NewThirdPartyError("network error", nil)
		}},
		nil,
		&log,
	)

	_, _, err := svc.ReportToAuthority(context.Background(), "txn-abc-123", "2026-03-01", "2026-03-31")
	if err == nil {
		t.Fatal("expected error from publisher, got nil")
	}
}

func TestReportToAuthority_PartialFailure(t *testing.T) {
	log := zerolog.Nop()
	svc := NewInvoiceService(
		&mockInvoiceRepo{},
		&mockPublisher{reportToAuthorityFn: func(_ context.Context, _, _, _ string) (int, int, error) {
			return 1, 1, domain.NewThirdPartyError("partial failure", nil)
		}},
		nil,
		&log,
	)

	successCount, errorCount, err := svc.ReportToAuthority(context.Background(), "txn-abc-123", "2026-03-01", "2026-03-31")
	if err == nil {
		t.Fatal("expected error for partial failure, got nil")
	}
	if successCount != 1 {
		t.Errorf("successCount = %d, want 1", successCount)
	}
	if errorCount != 1 {
		t.Errorf("errorCount = %d, want 1", errorCount)
	}
}
