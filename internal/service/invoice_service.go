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

func (s *InvoiceService) CreateDraft(ctx context.Context, invoice *domain.Invoice) error {
	invoice.ID = uuid.New()
	invoice.Status = domain.StatusDraft
	now := time.Now()
	invoice.CreatedAt = now
	invoice.UpdatedAt = now

	if err := s.repo.Create(ctx, invoice); err != nil {
		return err
	}

	s.log.Info().Str("invoice_id", invoice.ID.String()).Msg("Invoice draft created")
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

func (s *InvoiceService) UpdateInvoice(ctx context.Context, id uuid.UUID, invoice *domain.Invoice) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.Status != domain.StatusDraft {
		return domain.NewValidationError("can only update invoices in draft status")
	}

	exchangeRateChanged := existing.ExchangeRate != invoice.ExchangeRate

	invoice.ID = id
	invoice.Status = existing.Status
	invoice.CreatedAt = existing.CreatedAt
	invoice.UpdatedAt = time.Now()

	// Preserve financial totals (will be recalculated if rate changed)
	invoice.TotalAmount = existing.TotalAmount
	invoice.TaxAmount = existing.TaxAmount
	invoice.NetAmount = existing.NetAmount
	invoice.OriginalTotalAmount = existing.OriginalTotalAmount
	invoice.OriginalTaxAmount = existing.OriginalTaxAmount
	invoice.OriginalNetAmount = existing.OriginalNetAmount

	if err := s.repo.Update(ctx, invoice); err != nil {
		return err
	}

	if exchangeRateChanged {
		if err := s.reconvertItems(ctx, id, invoice.ExchangeRate); err != nil {
			return err
		}
	}

	return nil
}

func (s *InvoiceService) CancelInvoice(ctx context.Context, id uuid.UUID, reason string) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !existing.Status.CanTransitionTo(domain.StatusCancelled) {
		return domain.NewInvalidTransitionError(string(existing.Status), string(domain.StatusCancelled))
	}

	if err := s.repo.UpdateStatus(ctx, id, domain.StatusCancelled, reason); err != nil {
		return err
	}

	_ = s.repo.AddStatusHistory(ctx, &domain.InvoiceStatusHistory{
		ID:         uuid.New(),
		InvoiceID:  id,
		FromStatus: string(existing.Status),
		ToStatus:   string(domain.StatusCancelled),
		Reason:     reason,
		ChangedBy:  "api",
		CreatedAt:  time.Now(),
	})

	s.log.Info().Str("invoice_id", id.String()).Msg("Invoice cancelled")
	return nil
}

// AddItem adds a line item to an invoice and recalculates totals.
// The incoming UnitPrice is in the invoice's original currency.
func (s *InvoiceService) AddItem(ctx context.Context, invoiceID uuid.UUID, item *domain.InvoiceItem) error {
	existing, err := s.repo.GetByID(ctx, invoiceID)
	if err != nil {
		return err
	}
	if existing.Status != domain.StatusDraft {
		return domain.NewValidationError("can only add items to invoices in draft status")
	}

	item.ID = uuid.New()
	item.InvoiceID = invoiceID
	item.CreatedAt = time.Now()

	// Compute amounts in original currency
	originalUnitPrice := item.UnitPrice
	originalTaxAmount := originalUnitPrice * item.Quantity * item.TaxRate / 100
	originalLineTotal := originalUnitPrice*item.Quantity + originalTaxAmount

	item.OriginalUnitPrice = originalUnitPrice
	item.OriginalTaxAmount = originalTaxAmount
	item.OriginalLineTotal = originalLineTotal

	// Convert to VND using invoice exchange rate
	rate := existing.ExchangeRate
	item.UnitPrice = originalUnitPrice * rate
	item.TaxAmount = originalTaxAmount * rate
	item.LineTotal = originalLineTotal * rate

	if err := s.repo.AddItem(ctx, item); err != nil {
		return err
	}

	return s.recalculateTotals(ctx, invoiceID)
}

func (s *InvoiceService) RemoveItem(ctx context.Context, invoiceID uuid.UUID, itemID uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, invoiceID)
	if err != nil {
		return err
	}
	if existing.Status != domain.StatusDraft {
		return domain.NewValidationError("can only remove items from invoices in draft status")
	}

	if err := s.repo.DeleteItem(ctx, itemID); err != nil {
		return err
	}

	return s.recalculateTotals(ctx, invoiceID)
}

func (s *InvoiceService) GetStatusHistory(ctx context.Context, invoiceID uuid.UUID) ([]*domain.InvoiceStatusHistory, error) {
	if _, err := s.repo.GetByID(ctx, invoiceID); err != nil {
		return nil, err
	}
	return s.repo.GetStatusHistory(ctx, invoiceID)
}

// SubmitInvoice transitions to submitted. Actual 3rd party call is Part 4.2.
func (s *InvoiceService) SubmitInvoice(ctx context.Context, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !existing.Status.CanTransitionTo(domain.StatusSubmitted) {
		return domain.NewInvalidTransitionError(string(existing.Status), string(domain.StatusSubmitted))
	}

	items, err := s.repo.GetItemsByInvoiceID(ctx, id)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return domain.NewValidationError("invoice must have at least one item before submitting")
	}

	// Generate a stable transactionUuid for idempotent Viettel API calls.
	// This UUID is reused across retries to prevent duplicate invoices.
	txnUuid := uuid.New().String()
	existing.TransactionUuid = &txnUuid
	existing.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, existing); err != nil {
		return err
	}

	now := time.Now()
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
		CreatedAt:  now,
	})

	// Enqueue async job to publish invoice to Viettel
	if err := s.enqueuer.Enqueue(id); err != nil {
		// Treat enqueue failures as non-fatal: invoice is already submitted in persistent state.
		s.log.Error().Err(err).Str("invoice_id", id.String()).Msg("Failed to enqueue invoice publish job")
	}

	s.log.Info().Str("invoice_id", id.String()).Msg("Invoice submitted, enqueued for publishing")
	return nil
}

// recalculateTotals updates the invoice totals from its items.
func (s *InvoiceService) recalculateTotals(ctx context.Context, invoiceID uuid.UUID) error {
	items, err := s.repo.GetItemsByInvoiceID(ctx, invoiceID)
	if err != nil {
		return err
	}

	var totalAmount, taxAmount float64
	var originalTotalAmount, originalTaxAmount float64
	for _, item := range items {
		totalAmount += item.LineTotal
		taxAmount += item.TaxAmount
		originalTotalAmount += item.OriginalLineTotal
		originalTaxAmount += item.OriginalTaxAmount
	}

	invoice, err := s.repo.GetByID(ctx, invoiceID)
	if err != nil {
		return err
	}

	invoice.TotalAmount = totalAmount
	invoice.TaxAmount = taxAmount
	invoice.NetAmount = totalAmount - taxAmount
	invoice.OriginalTotalAmount = originalTotalAmount
	invoice.OriginalTaxAmount = originalTaxAmount
	invoice.OriginalNetAmount = originalTotalAmount - originalTaxAmount
	invoice.UpdatedAt = time.Now()

	return s.repo.Update(ctx, invoice)
}

// reconvertItems recalculates VND amounts for all items when exchange rate changes.
func (s *InvoiceService) reconvertItems(ctx context.Context, invoiceID uuid.UUID, newRate float64) error {
	items, err := s.repo.GetItemsByInvoiceID(ctx, invoiceID)
	if err != nil {
		return err
	}

	for _, item := range items {
		item.UnitPrice = item.OriginalUnitPrice * newRate
		item.TaxAmount = item.OriginalTaxAmount * newRate
		item.LineTotal = item.OriginalLineTotal * newRate
		if err := s.repo.UpdateItem(ctx, item); err != nil {
			return err
		}
	}

	return s.recalculateTotals(ctx, invoiceID)
}
