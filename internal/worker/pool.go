package worker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"tax-module/internal/config"
	"tax-module/internal/domain"
	"tax-module/internal/integration"
)

// JobType identifies the kind of work to perform.
type JobType string

const (
	JobPublishInvoice JobType = "publish_invoice"
	JobPollStatus     JobType = "poll_status"
)

// Job represents a unit of work for the worker pool.
type Job struct {
	Type      JobType
	InvoiceID uuid.UUID
}

// Pool manages a fixed set of goroutines that process invoice jobs.
type Pool struct {
	cfg       config.WorkerConfig
	queue     chan Job
	publisher domain.InvoicePublisher
	repo      domain.InvoiceRepository
	log       *zerolog.Logger

	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// NewPool creates a worker pool. Call Start() to begin processing.
func NewPool(
	cfg config.WorkerConfig,
	publisher domain.InvoicePublisher,
	repo domain.InvoiceRepository,
	log *zerolog.Logger,
) *Pool {
	return &Pool{
		cfg:       cfg,
		queue:     make(chan Job, cfg.QueueSize),
		publisher: publisher,
		repo:      repo,
		log:       log,
	}
}

// Start launches the worker goroutines and the status poller.
func (p *Pool) Start(ctx context.Context) {
	ctx, p.cancel = context.WithCancel(ctx)

	for i := 0; i < p.cfg.PoolSize; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}

	// Background poller picks up invoices stuck in "submitted"
	p.wg.Add(1)
	go p.poller(ctx)

	p.log.Info().
		Int("pool_size", p.cfg.PoolSize).
		Int("queue_size", p.cfg.QueueSize).
		Msg("Worker pool started")
}

// Enqueue adds a job to the queue. Returns QueueFull error if the channel is full.
func (p *Pool) Enqueue(job Job) error {
	select {
	case p.queue <- job:
		p.log.Debug().
			Str("type", string(job.Type)).
			Str("invoice_id", job.InvoiceID.String()).
			Msg("Job enqueued")
		return nil
	default:
		return domain.NewQueueFullError()
	}
}

// Shutdown signals all workers to stop and waits for them to finish.
func (p *Pool) Shutdown() {
	p.log.Info().Msg("Worker pool shutting down...")
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
	p.log.Info().Msg("Worker pool stopped")
}

// worker processes jobs from the queue.
func (p *Pool) worker(ctx context.Context, id int) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-p.queue:
			if !ok {
				return
			}
			p.processJob(ctx, job, id)
		}
	}
}

// processJob dispatches to the correct handler based on job type.
func (p *Pool) processJob(ctx context.Context, job Job, workerID int) {
	log := p.log.With().
		Int("worker_id", workerID).
		Str("job_type", string(job.Type)).
		Str("invoice_id", job.InvoiceID.String()).
		Logger()

	log.Info().Msg("Processing job")

	switch job.Type {
	case JobPublishInvoice:
		p.handlePublish(ctx, job.InvoiceID, &log)
	case JobPollStatus:
		p.handlePoll(ctx, job.InvoiceID, &log)
	default:
		log.Error().Msg("Unknown job type")
	}
}

// handlePublish sends the invoice to Viettel.
func (p *Pool) handlePublish(ctx context.Context, invoiceID uuid.UUID, log *zerolog.Logger) {
	invoice, err := p.repo.GetByID(ctx, invoiceID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load invoice")
		return
	}

	// Allow invoices that are freshly submitted, or already in processing but missing an ExternalID
	isSubmitted := invoice.Status == domain.StatusSubmitted
	isRecoverableProcessing := invoice.Status == domain.StatusProcessing && (invoice.ExternalID == nil || *invoice.ExternalID == "")

	if !isSubmitted && !isRecoverableProcessing {
		log.Warn().
			Str("status", string(invoice.Status)).
			Msg("Invoice not in submitted or recoverable processing status, skipping")
		return
	}

	// Only transition to processing if we are coming from submitted; if already processing, keep status
	if isSubmitted {
		if err := p.repo.UpdateStatus(ctx, invoiceID, domain.StatusProcessing, "worker: sending to viettel"); err != nil {
			log.Error().Err(err).Msg("Failed to transition to processing")
			return
		}
	}

	items, err := p.repo.GetItemsByInvoiceID(ctx, invoiceID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load invoice items")
		_ = p.repo.UpdateStatus(ctx, invoiceID, domain.StatusFailed, err.Error())
		return
	}
	invoice.Items = items

	externalID, err := p.publisher.CreateInvoice(ctx, invoice)
	if err != nil {
		p.handlePublishError(ctx, invoice, err, log)
		return
	}

	invoice.ExternalID = &externalID
	now := time.Now()
	invoice.CompletedAt = &now
	invoice.UpdatedAt = now
	if err := p.repo.Update(ctx, invoice); err != nil {
		log.Error().Err(err).Msg("Failed to update invoice with external ID")
		return
	}

	if err := p.repo.UpdateStatus(ctx, invoiceID, domain.StatusCompleted, "published to viettel"); err != nil {
		log.Error().Err(err).Msg("Failed to transition to completed")
		return
	}

	_ = p.repo.AddStatusHistory(ctx, &domain.InvoiceStatusHistory{
		ID:         uuid.New(),
		InvoiceID:  invoiceID,
		FromStatus: string(domain.StatusProcessing),
		ToStatus:   string(domain.StatusCompleted),
		Reason:     "published to viettel, externalID=" + externalID,
		ChangedBy:  "worker",
		CreatedAt:  now,
	})

	log.Info().Str("external_id", externalID).Msg("Invoice published successfully")
}

func (p *Pool) handlePublishError(ctx context.Context, invoice *domain.Invoice, publishErr error, log *zerolog.Logger) {
	// Non-retryable errors (e.g. validation / HTTP 400) should fail immediately
	var vErr *integration.ViettelError
	if errors.As(publishErr, &vErr) && !vErr.Retryable {
		_ = p.repo.UpdateStatus(ctx, invoice.ID, domain.StatusFailed, "non-retryable: "+vErr.Error())
		log.Error().
			Str("viettel_code", string(vErr.ErrCode)).
			Str("description", vErr.Description).
			Msg("Invoice publish failed permanently (validation error)")
		return
	}

	invoice.RetryCount++
	errMsg := publishErr.Error()
	invoice.LastError = &errMsg
	invoice.UpdatedAt = time.Now()
	_ = p.repo.Update(ctx, invoice)

	if invoice.RetryCount >= p.cfg.MaxRetries {
		_ = p.repo.UpdateStatus(ctx, invoice.ID, domain.StatusFailed, "max retries exceeded: "+publishErr.Error())
		log.Error().Int("retry_count", invoice.RetryCount).Err(publishErr).Msg("Invoice publish failed permanently")
		return
	}

	_ = p.repo.UpdateStatus(ctx, invoice.ID, domain.StatusSubmitted, "retry pending: "+publishErr.Error())
	log.Warn().Int("retry_count", invoice.RetryCount).Err(publishErr).Msg("Invoice publish failed, will retry")
}

// handlePoll checks a processing invoice's status via Viettel.
func (p *Pool) handlePoll(ctx context.Context, invoiceID uuid.UUID, log *zerolog.Logger) {
	invoice, err := p.repo.GetByID(ctx, invoiceID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to load invoice for polling")
		return
	}

	if invoice.ExternalID == nil || *invoice.ExternalID == "" {
		log.Warn().Msg("Invoice has no external ID, skipping poll")
		return
	}

	status, rawResponse, err := p.publisher.QueryStatus(ctx, *invoice.ExternalID)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to query invoice status")
		return
	}

	log.Info().Str("viettel_status", status).Msg("Polled invoice status")

	switch status {
	case "completed":
		now := time.Now()
		invoice.CompletedAt = &now
		invoice.Metadata = rawResponse
		invoice.UpdatedAt = now
		_ = p.repo.Update(ctx, invoice)
		_ = p.repo.UpdateStatus(ctx, invoiceID, domain.StatusCompleted, "viettel confirmed")
	case "pending", "processing":
		invoice.Metadata = rawResponse
		invoice.UpdatedAt = time.Now()
		_ = p.repo.Update(ctx, invoice)
	}
}

// poller periodically picks up invoices in "submitted" status for retry.
func (p *Pool) poller(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.pollPendingInvoices(ctx)
		}
	}
}

func (p *Pool) pollPendingInvoices(ctx context.Context) {
	invoices, err := p.repo.GetPendingPolling(ctx, p.cfg.QueueSize)
	if err != nil {
		p.log.Error().Err(err).Msg("Failed to fetch pending invoices")
		return
	}

	for _, inv := range invoices {
		jobType := JobPublishInvoice
		if inv.ExternalID != nil && *inv.ExternalID != "" {
			jobType = JobPollStatus
		}

		if err := p.Enqueue(Job{Type: jobType, InvoiceID: inv.ID}); err != nil {
			p.log.Warn().Str("invoice_id", inv.ID.String()).Msg("Queue full, skipping invoice in poll cycle")
			break
		}
	}

	if len(invoices) > 0 {
		p.log.Info().Int("count", len(invoices)).Msg("Re-enqueued pending invoices")
	}
}
