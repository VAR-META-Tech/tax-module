package integration

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog"

	"tax-module/internal/config"
	"tax-module/internal/domain"
)

// ViettelPublisher implements domain.InvoicePublisher using the Viettel SInvoice API.
type ViettelPublisher struct {
	client *ViettelClient
	cfg    config.ThirdPartyConfig
	log    *zerolog.Logger
}

// NewViettelPublisher creates a new publisher backed by Viettel.
func NewViettelPublisher(client *ViettelClient, cfg config.ThirdPartyConfig, log *zerolog.Logger) *ViettelPublisher {
	return &ViettelPublisher{client: client, cfg: cfg, log: log}
}

// CreateInvoice maps the domain invoice, calls Viettel, and returns the transactionUuid as externalID.
func (p *ViettelPublisher) CreateInvoice(ctx context.Context, invoice *domain.Invoice) (string, error) {
	viettelReq := MapInvoiceToViettel(invoice, p.cfg)
	transactionUuid := viettelReq.GeneralInvoiceInfo.TransactionUuid

	p.log.Info().
		Str("invoice_id", invoice.ID.String()).
		Str("transaction_uuid", transactionUuid).
		Msg("Sending invoice to Viettel SInvoice")

	resp, err := p.client.CreateInvoice(ctx, viettelReq)
	if err != nil {
		return "", err
	}

	if resp.ErrorCode != nil && *resp.ErrorCode != "" {
		desc := ""
		if resp.Description != nil {
			desc = *resp.Description
		}
		p.log.Error().
			Str("error_code", *resp.ErrorCode).
			Str("description", desc).
			Str("transaction_uuid", transactionUuid).
			Msg("Viettel createInvoice returned error")
		return "", domain.NewThirdPartyError("viettel createInvoice error: "+desc, nil)
	}

	externalID := transactionUuid
	if resp.Result != nil && resp.Result.InvoiceNo != "" {
		externalID = resp.Result.InvoiceNo
		p.log.Info().
			Str("invoice_no", resp.Result.InvoiceNo).
			Str("transaction_uuid", transactionUuid).
			Msg("Viettel invoice created with invoiceNo")
	} else {
		p.log.Info().
			Str("transaction_uuid", transactionUuid).
			Msg("Viettel invoice created (invoiceNo pending — async)")
	}

	return externalID, nil
}

// QueryStatus checks invoice status via searchByTransactionUuid.
func (p *ViettelPublisher) QueryStatus(ctx context.Context, externalID string) (string, []byte, error) {
	resp, err := p.client.SearchByTransactionUuid(ctx, externalID, p.cfg.SupplierCode)
	if err != nil {
		return "", nil, err
	}

	rawResponse, _ := json.Marshal(resp)

	if resp.ErrorCode != nil && *resp.ErrorCode != "" {
		desc := ""
		if resp.Description != nil {
			desc = *resp.Description
		}
		return "", rawResponse, domain.NewThirdPartyError("viettel query error: "+desc, nil)
	}

	if len(resp.Result) == 0 {
		return "pending", rawResponse, nil
	}

	result := resp.Result[0]
	if result.InvoiceNo != "" {
		return "completed", rawResponse, nil
	}

	return "processing", rawResponse, nil
}
