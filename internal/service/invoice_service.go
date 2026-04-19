package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"tax-module/internal/domain"
)

// JobEnqueuer is the interface for enqueuing async work.
type JobEnqueuer interface {
	Enqueue(invoiceID uuid.UUID) error
}

type InvoiceService struct {
	repo      domain.InvoiceRepository
	publisher domain.InvoicePublisher
	enqueuer  JobEnqueuer
	log       *zerolog.Logger
}

func NewInvoiceService(repo domain.InvoiceRepository, publisher domain.InvoicePublisher, enqueuer JobEnqueuer, log *zerolog.Logger) *InvoiceService {
	return &InvoiceService{repo: repo, publisher: publisher, enqueuer: enqueuer, log: log}
}

// CreateInvoice saves an invoice with its items in a single DB transaction.
// The invoice stays in draft status until SubmitInvoice is called.
func (s *InvoiceService) CreateInvoice(ctx context.Context, invoice *domain.Invoice) error {
	now := time.Now()
	invoice.ID = uuid.New()
	invoice.Status = domain.StatusDraft
	invoice.CreatedAt = now
	invoice.UpdatedAt = now

	txnUuid := uuid.New().String()
	invoice.TransactionUuid = &txnUuid

	// Assign IDs to items
	for _, item := range invoice.Items {
		item.ID = uuid.New()
		item.InvoiceID = invoice.ID
		item.CreatedAt = now
	}

	// Save invoice + items atomically
	if err := s.repo.CreateWithItems(ctx, invoice); err != nil {
		return err
	}

	s.log.Info().Str("invoice_id", invoice.ID.String()).Msg("Invoice created with items (draft)")
	return nil
}

// UpdateTransactionHash saves the blockchain transaction hash for a draft invoice
// after payment has been completed on the frontend.
func (s *InvoiceService) UpdateTransactionHash(ctx context.Context, id uuid.UUID, transactionHash string) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if existing.Status != domain.StatusDraft {
		return domain.NewValidationError("transaction hash can only be updated on draft invoices")
	}

	if err := s.repo.UpdateTransactionHash(ctx, id, transactionHash); err != nil {
		return err
	}

	s.log.Info().Str("invoice_id", id.String()).Str("transaction_hash", transactionHash).Msg("Transaction hash updated")
	return nil
}

// SubmitInvoice transitions a draft invoice to submitted and enqueues it
// for async publishing to Viettel SInvoice.
func (s *InvoiceService) SubmitInvoice(ctx context.Context, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if !existing.Status.CanTransitionTo(domain.StatusSubmitted) {
		return domain.NewInvalidTransitionError(string(existing.Status), string(domain.StatusSubmitted))
	}

	if err := s.repo.UpdateStatus(ctx, id, domain.StatusSubmitted, "submitted via API"); err != nil {
		return err
	}

	_ = s.repo.AddStatusHistory(ctx, &domain.InvoiceStatusHistory{
		ID:         uuid.New(),
		InvoiceID:  id,
		FromStatus: string(existing.Status),
		ToStatus:   string(domain.StatusSubmitted),
		Reason:     "submitted via API",
		ChangedBy:  "api",
		CreatedAt:  time.Now(),
	})

	if err := s.enqueuer.Enqueue(id); err != nil {
		s.log.Error().Err(err).Str("invoice_id", id.String()).Msg("Failed to enqueue invoice publish job")
	}

	s.log.Info().Str("invoice_id", id.String()).Msg("Invoice submitted, enqueued for publishing")
	return nil
}

func (s *InvoiceService) GetInvoice(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	invoice, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	items, err := s.repo.GetItemsByInvoiceID(ctx, id)
	if err != nil {
		return nil, err
	}
	invoice.Items = items

	return invoice, nil
}

func (s *InvoiceService) ListInvoices(ctx context.Context, filter domain.InvoiceFilter) ([]*domain.Invoice, int64, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	return s.repo.List(ctx, filter)
}

func (s *InvoiceService) GetStatusHistory(ctx context.Context, invoiceID uuid.UUID) ([]*domain.InvoiceStatusHistory, error) {
	if _, err := s.repo.GetByID(ctx, invoiceID); err != nil {
		return nil, err
	}
	return s.repo.GetStatusHistory(ctx, invoiceID)
}

// DownloadInvoiceFile downloads the invoice PDF from the third-party provider and returns the base64 string.
func (s *InvoiceService) DownloadInvoiceFile(ctx context.Context, id uuid.UUID) (string, string, error) {
	invoice, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", "", err
	}

	if invoice.ExternalID == nil || *invoice.ExternalID == "" {
		return "", "", domain.NewValidationError("invoice has not been published to the provider yet")
	}

	fileBase64, err := s.publisher.DownloadInvoiceFile(ctx, invoice.Provider, invoice, "PDF")
	if err != nil {
		return "", "", err
	}

	return fileBase64, *invoice.ExternalID, nil
}

// ReportToAuthority sends a completed invoice to the tax authority (CQT).
func (s *InvoiceService) ReportToAuthority(ctx context.Context, transactionUuid, startDate, endDate string) (int, int, error) {
	invoice, err := s.repo.GetByTransactionUuid(ctx, transactionUuid)
	if err != nil {
		return 0, 0, err
	}

	successCount, errorCount, err := s.publisher.ReportToAuthority(ctx, invoice.Provider, transactionUuid, startDate, endDate)
	if err != nil {
		s.log.Error().Err(err).
			Str("transaction_uuid", transactionUuid).
			Int("success", successCount).
			Int("errors", errorCount).
			Msg("Failed to send invoice to tax authority")
		return successCount, errorCount, err
	}

	s.log.Info().
		Str("transaction_uuid", transactionUuid).
		Int("success", successCount).
		Msg("Invoice sent to tax authority")

	return successCount, errorCount, nil
}
