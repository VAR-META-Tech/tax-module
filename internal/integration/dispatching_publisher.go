package integration

import (
	"context"
	"fmt"

	"tax-module/internal/domain"
)

// DispatchingPublisher implements domain.InvoicePublisher by routing calls to
// the appropriate provider based on invoice.Provider or an explicit provider string.
type DispatchingPublisher struct {
	providers       map[string]domain.InvoicePublisher
	defaultProvider string
}

// NewDispatchingPublisher creates a publisher that routes between Viettel and MISA.
// defaultProvider is used when an invoice has no provider set.
func NewDispatchingPublisher(defaultProvider string, viettel, misa domain.InvoicePublisher) *DispatchingPublisher {
	return &DispatchingPublisher{
		providers: map[string]domain.InvoicePublisher{
			domain.ProviderViettel: viettel,
			domain.ProviderMISA:   misa,
		},
		defaultProvider: defaultProvider,
	}
}

// resolve returns the publisher for the given provider name, falling back to default.
func (d *DispatchingPublisher) resolve(provider string) (domain.InvoicePublisher, error) {
	if provider == "" {
		provider = d.defaultProvider
	}
	pub, ok := d.providers[provider]
	if !ok {
		return nil, fmt.Errorf("unknown invoice provider: %q", provider)
	}
	return pub, nil
}

// CreateInvoice routes to the provider specified in invoice.Provider.
func (d *DispatchingPublisher) CreateInvoice(ctx context.Context, invoice *domain.Invoice) (string, error) {
	pub, err := d.resolve(invoice.Provider)
	if err != nil {
		return "", domain.NewValidationError(err.Error())
	}
	return pub.CreateInvoice(ctx, invoice)
}

// QueryStatus routes to the provider specified in invoice.Provider.
func (d *DispatchingPublisher) QueryStatus(ctx context.Context, invoice *domain.Invoice) (string, string, []byte, error) {
	pub, err := d.resolve(invoice.Provider)
	if err != nil {
		return "", "", nil, domain.NewValidationError(err.Error())
	}
	return pub.QueryStatus(ctx, invoice)
}

// ReportToAuthority routes to the provider specified by the provider param.
func (d *DispatchingPublisher) ReportToAuthority(ctx context.Context, provider, transactionUuid, startDate, endDate string) (int, int, error) {
	pub, err := d.resolve(provider)
	if err != nil {
		return 0, 0, domain.NewValidationError(err.Error())
	}
	return pub.ReportToAuthority(ctx, provider, transactionUuid, startDate, endDate)
}

// DownloadInvoiceFile routes to the provider specified by the provider param.
func (d *DispatchingPublisher) DownloadInvoiceFile(ctx context.Context, provider string, invoice *domain.Invoice, fileType string) (string, error) {
	pub, err := d.resolve(provider)
	if err != nil {
		return "", domain.NewValidationError(err.Error())
	}
	return pub.DownloadInvoiceFile(ctx, provider, invoice, fileType)
}
