package integration

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/rs/zerolog"

	"tax-module/internal/config"
	"tax-module/internal/domain"
)

// MISAPublisher implements domain.InvoicePublisher using the MISA MeInvoice Integration API.
// MISA with SignType=2/5 is synchronous — results are returned immediately, no polling needed.
type MISAPublisher struct {
	client *MISAClient
	cfg    config.MISAConfig
	log    *zerolog.Logger
}

// NewMISAPublisher creates a new publisher backed by MISA MeInvoice.
func NewMISAPublisher(client *MISAClient, cfg config.MISAConfig, log *zerolog.Logger) *MISAPublisher {
	return &MISAPublisher{client: client, cfg: cfg, log: log}
}

// CreateInvoice publishes an invoice to MISA synchronously (SignType 2 or 5).
// Returns MISA's TransactionID as the external identifier (stored as external_id).
// An empty return means MISA is still processing (should not happen for SignType 2/5).
func (p *MISAPublisher) CreateInvoice(ctx context.Context, invoice *domain.Invoice) (string, error) {
	template, err := p.client.FetchTemplate(ctx)
	if err != nil {
		return "", domain.NewThirdPartyError("misa fetch template: "+err.Error(), err)
	}

	invoiceData := MapInvoiceToMISA(invoice, template, p.cfg.InvoiceCalcu)

	req := &MISAPublishRequest{
		SignType:           p.cfg.SignType,
		InvoiceData:        []MISAInvoiceData{*invoiceData},
		PublishInvoiceData: nil,
	}

	p.log.Info().
		Str("invoice_id", invoice.ID.String()).
		Str("ref_id", invoiceData.RefID).
		Int("sign_type", p.cfg.SignType).
		Msg("Publishing invoice to MISA MeInvoice")

	resp, err := p.client.PublishInvoice(ctx, req)
	if err != nil {
		return "", domain.NewThirdPartyError("misa publish: "+err.Error(), err)
	}

	if !resp.Success {
		p.log.Error().
			Str("error_code", resp.ErrorCode).
			Str("description", resp.DescriptionErrorCode).
			Msg("MISA publish response not successful")
		return "", domain.NewThirdPartyError("misa publish failed: "+resp.DescriptionErrorCode, nil)
	}

	if len(resp.PublishInvoiceResult) == 0 {
		return "", domain.NewThirdPartyError("misa publish returned empty result", nil)
	}

	result := resp.PublishInvoiceResult[0]

	if result.ErrorCode != "" {
		p.log.Error().
			Str("ref_id", result.RefID).
			Str("error_code", result.ErrorCode).
			Str("description", result.Description).
			Msg("MISA invoice publish error")

		misaErr := newMISAError(result.ErrorCode, result.Description)
		if misaErr.Retryable {
			return "", misaErr
		}
		return "", domain.NewThirdPartyError("misa invoice error: "+result.Description, misaErr)
	}

	p.log.Info().
		Str("transaction_id", result.TransactionID).
		Str("inv_no", result.InvNo).
		Str("ref_id", result.RefID).
		Msg("MISA invoice published successfully")

	// Return TransactionID — this is stored as external_id and used for downloads
	return result.TransactionID, nil
}

// QueryStatus checks invoice status using our RefID (inputType=2).
// For MISA with SignType 2/5, this is only used if the initial publish somehow
// did not return a TransactionID immediately.
func (p *MISAPublisher) QueryStatus(ctx context.Context, invoice *domain.Invoice) (string, string, []byte, error) {
	refID := ""
	if invoice.TransactionUuid != nil {
		refID = *invoice.TransactionUuid
	}

	resp, err := p.client.GetInvoiceStatus(ctx, []string{refID})
	if err != nil {
		return "", "", nil, domain.NewThirdPartyError("misa query status: "+err.Error(), err)
	}

	raw, _ := json.Marshal(resp)

	if !resp.Success {
		return "", "", raw, domain.NewThirdPartyError("misa status query failed: "+resp.ErrorCode, nil)
	}

	if len(resp.Data) == 0 {
		return "processing", "", raw, nil
	}

	status := resp.Data[0]
	if status.PublishStatus == 1 {
		return "completed", status.TransactionID, raw, nil
	}

	return "processing", "", raw, nil
}

// ReportToAuthority is a no-op for MISA — the platform handles CQT submission automatically.
func (p *MISAPublisher) ReportToAuthority(_ context.Context, _, _, _, _ string) (int, int, error) {
	p.log.Warn().Msg("ReportToAuthority called for MISA — not supported, MISA handles CQT automatically")
	return 0, 0, nil
}

// DownloadInvoiceFile downloads the invoice PDF from MISA using the TransactionID (= external_id).
func (p *MISAPublisher) DownloadInvoiceFile(ctx context.Context, _ string, invoice *domain.Invoice, fileType string) (string, error) {
	if invoice.ExternalID == nil || *invoice.ExternalID == "" {
		return "", domain.NewValidationError("invoice has no external_id (MISA TransactionID) — not yet published")
	}

	transactionID := *invoice.ExternalID

	p.log.Info().
		Str("transaction_id", transactionID).
		Str("file_type", fileType).
		Msg("Downloading invoice file from MISA")

	resp, err := p.client.DownloadInvoice(ctx, []string{transactionID}, strings.ToLower(fileType))
	if err != nil {
		return "", domain.NewThirdPartyError("misa download: "+err.Error(), err)
	}

	if !resp.Success {
		return "", domain.NewThirdPartyError("misa download failed: "+resp.ErrorCode, nil)
	}

	if len(resp.Data) == 0 || resp.Data[0].Data == "" {
		return "", domain.NewThirdPartyError("misa download returned empty file", nil)
	}

	return resp.Data[0].Data, nil
}
