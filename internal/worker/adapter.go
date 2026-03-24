package worker

import "github.com/google/uuid"

// Adapter implements service.JobEnqueuer by wrapping the worker Pool.
type Adapter struct {
	pool *Pool
}

// NewAdapter creates a JobEnqueuer that delegates to the worker pool.
func NewAdapter(pool *Pool) *Adapter {
	return &Adapter{pool: pool}
}

// Enqueue creates a PublishInvoice job and submits it to the pool.
func (a *Adapter) Enqueue(invoiceID uuid.UUID) error {
	return a.pool.Enqueue(Job{
		Type:      JobPublishInvoice,
		InvoiceID: invoiceID,
	})
}
