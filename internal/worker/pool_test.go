package worker

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"tax-module/internal/config"
	"tax-module/internal/domain"
)

type mockPublisher struct {
	mu          sync.Mutex
	createCalls int
	queryCalls  int
	createErr   error
	queryStatus string
}

func (m *mockPublisher) CreateInvoice(_ context.Context, _ *domain.Invoice) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.createCalls++
	if m.createErr != nil {
		return "", m.createErr
	}
	return "ext-" + uuid.New().String()[:8], nil
}

func (m *mockPublisher) QueryStatus(_ context.Context, _ string) (string, []byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.queryCalls++
	return m.queryStatus, nil, nil
}

func (m *mockPublisher) SendToTax(_ context.Context, _, _, _ string) (int, int, error) {
	return 0, 0, nil
}

type mockRepo struct {
	mu       sync.Mutex
	invoices map[uuid.UUID]*domain.Invoice
	items    map[uuid.UUID][]*domain.InvoiceItem
	statuses map[uuid.UUID]domain.InvoiceStatus
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		invoices: make(map[uuid.UUID]*domain.Invoice),
		items:    make(map[uuid.UUID][]*domain.InvoiceItem),
		statuses: make(map[uuid.UUID]domain.InvoiceStatus),
	}
}

func (r *mockRepo) Create(_ context.Context, inv *domain.Invoice) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.invoices[inv.ID] = inv
	r.statuses[inv.ID] = inv.Status
	return nil
}

func (r *mockRepo) CreateWithItems(_ context.Context, inv *domain.Invoice) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.invoices[inv.ID] = inv
	r.statuses[inv.ID] = inv.Status
	for _, item := range inv.Items {
		r.items[inv.ID] = append(r.items[inv.ID], item)
	}
	return nil
}

func (r *mockRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Invoice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	inv, ok := r.invoices[id]
	if !ok {
		return nil, domain.NewNotFoundError("invoice not found")
	}
	copy := *inv
	if s, ok := r.statuses[id]; ok {
		copy.Status = s
	}
	return &copy, nil
}

func (r *mockRepo) Update(_ context.Context, inv *domain.Invoice) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.invoices[inv.ID] = inv
	return nil
}

func (r *mockRepo) UpdateStatus(_ context.Context, id uuid.UUID, status domain.InvoiceStatus, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.statuses[id] = status
	if inv, ok := r.invoices[id]; ok {
		inv.Status = status
	}
	return nil
}

func (r *mockRepo) List(_ context.Context, _ domain.InvoiceFilter) ([]*domain.Invoice, int64, error) {
	return nil, 0, nil
}

func (r *mockRepo) GetByExternalID(_ context.Context, _ string) (*domain.Invoice, error) {
	return nil, domain.NewNotFoundError("not found")
}

func (r *mockRepo) GetByTransactionUuid(_ context.Context, _ string) (*domain.Invoice, error) {
	return nil, domain.NewNotFoundError("not found")
}

func (r *mockRepo) GetPendingPolling(_ context.Context, _ int) ([]*domain.Invoice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*domain.Invoice
	for _, inv := range r.invoices {
		if inv.Status == domain.StatusSubmitted || inv.Status == domain.StatusProcessing {
			result = append(result, inv)
		}
	}
	return result, nil
}

func (r *mockRepo) AddItem(_ context.Context, item *domain.InvoiceItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[item.InvoiceID] = append(r.items[item.InvoiceID], item)
	return nil
}

func (r *mockRepo) GetItemsByInvoiceID(_ context.Context, invoiceID uuid.UUID) ([]*domain.InvoiceItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.items[invoiceID], nil
}

func (r *mockRepo) UpdateItem(_ context.Context, _ *domain.InvoiceItem) error { return nil }
func (r *mockRepo) DeleteItem(_ context.Context, _ uuid.UUID) error           { return nil }
func (r *mockRepo) AddStatusHistory(_ context.Context, _ *domain.InvoiceStatusHistory) error {
	return nil
}
func (r *mockRepo) GetStatusHistory(_ context.Context, _ uuid.UUID) ([]*domain.InvoiceStatusHistory, error) {
	return nil, nil
}
func (r *mockRepo) AddAuditLog(_ context.Context, _ *domain.AuditLog) error { return nil }

func (r *mockRepo) getStatus(id uuid.UUID) domain.InvoiceStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.statuses[id]
}

func TestPool_PublishInvoice(t *testing.T) {
	log := zerolog.Nop()
	repo := newMockRepo()
	pub := &mockPublisher{}

	cfg := config.WorkerConfig{
		PoolSize:     2,
		QueueSize:    10,
		PollInterval: 1 * time.Hour,
		MaxRetries:   3,
	}

	pool := NewPool(cfg, pub, repo, &log)

	invID := uuid.New()
	inv := &domain.Invoice{ID: invID, Status: domain.StatusSubmitted, Currency: "VND"}
	repo.Create(context.Background(), inv)
	repo.items[invID] = []*domain.InvoiceItem{
		{ID: uuid.New(), InvoiceID: invID, ItemName: "Test", Quantity: 1, UnitPrice: 1000, ItemTotalAmountWithoutTax: 1000},
	}

	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	err := pool.Enqueue(Job{Type: JobPublishInvoice, InvoiceID: invID})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	time.Sleep(200 * time.Millisecond)
	cancel()
	pool.wg.Wait()

	status := repo.getStatus(invID)
	if status != domain.StatusCompleted {
		t.Errorf("status = %q, want %q", status, domain.StatusCompleted)
	}

	pub.mu.Lock()
	if pub.createCalls != 1 {
		t.Errorf("createCalls = %d, want 1", pub.createCalls)
	}
	pub.mu.Unlock()

	updated, _ := repo.GetByID(context.Background(), invID)
	if updated.ExternalID == nil || *updated.ExternalID == "" {
		t.Error("ExternalID should not be empty after publish")
	}
}

func TestPool_PublishError_Retry(t *testing.T) {
	log := zerolog.Nop()
	repo := newMockRepo()
	pub := &mockPublisher{createErr: domain.NewThirdPartyError("connection refused", nil)}

	cfg := config.WorkerConfig{
		PoolSize:     1,
		QueueSize:    10,
		PollInterval: 1 * time.Hour,
		MaxRetries:   3,
	}

	pool := NewPool(cfg, pub, repo, &log)

	invID := uuid.New()
	inv := &domain.Invoice{ID: invID, Status: domain.StatusSubmitted, Currency: "VND"}
	repo.Create(context.Background(), inv)
	repo.items[invID] = []*domain.InvoiceItem{
		{ID: uuid.New(), InvoiceID: invID, ItemName: "Test", Quantity: 1, UnitPrice: 1000, ItemTotalAmountWithoutTax: 1000},
	}

	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	pool.Enqueue(Job{Type: JobPublishInvoice, InvoiceID: invID})
	time.Sleep(200 * time.Millisecond)
	cancel()
	pool.wg.Wait()

	status := repo.getStatus(invID)
	if status != domain.StatusSubmitted {
		t.Errorf("status = %q, want %q (for retry)", status, domain.StatusSubmitted)
	}

	updated, _ := repo.GetByID(context.Background(), invID)
	if updated.RetryCount != 1 {
		t.Errorf("RetryCount = %d, want 1", updated.RetryCount)
	}
}

func TestPool_QueueFull(t *testing.T) {
	log := zerolog.Nop()
	repo := newMockRepo()
	pub := &mockPublisher{}

	cfg := config.WorkerConfig{PoolSize: 1, QueueSize: 1, PollInterval: 1 * time.Hour, MaxRetries: 3}
	pool := NewPool(cfg, pub, repo, &log)

	pool.Enqueue(Job{Type: JobPublishInvoice, InvoiceID: uuid.New()})

	err := pool.Enqueue(Job{Type: JobPublishInvoice, InvoiceID: uuid.New()})
	if err == nil {
		t.Fatal("expected QueueFull error, got nil")
	}

	appErr, ok := err.(*domain.AppError)
	if !ok {
		t.Fatalf("expected *AppError, got %T", err)
	}
	if appErr.Code != domain.ErrCodeQueueFull {
		t.Errorf("error code = %q, want %q", appErr.Code, domain.ErrCodeQueueFull)
	}
}

func TestAdapter_Enqueue(t *testing.T) {
	log := zerolog.Nop()
	repo := newMockRepo()
	pub := &mockPublisher{}

	cfg := config.WorkerConfig{PoolSize: 1, QueueSize: 10, PollInterval: 1 * time.Hour, MaxRetries: 3}
	pool := NewPool(cfg, pub, repo, &log)
	adapter := NewAdapter(pool)

	invID := uuid.New()
	err := adapter.Enqueue(invID)
	if err != nil {
		t.Fatalf("Adapter.Enqueue: %v", err)
	}

	select {
	case job := <-pool.queue:
		if job.Type != JobPublishInvoice {
			t.Errorf("job type = %q, want %q", job.Type, JobPublishInvoice)
		}
		if job.InvoiceID != invID {
			t.Errorf("job invoice ID = %s, want %s", job.InvoiceID, invID)
		}
	default:
		t.Fatal("expected a job in the queue")
	}
}
