package integration

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/rs/zerolog"

	"tax-module/internal/config"
	"tax-module/internal/domain"
)

// ViettelPublisher implements domain.InvoicePublisher using the Viettel SInvoice API.
type ViettelPublisher struct {
	client    *ViettelClient
	cfg       config.ThirdPartyConfig
	sellerCfg config.SellerConfig
	log       *zerolog.Logger
}

// NewViettelPublisher creates a new publisher backed by Viettel.
func NewViettelPublisher(client *ViettelClient, cfg config.ThirdPartyConfig, sellerCfg config.SellerConfig, log *zerolog.Logger) *ViettelPublisher {
	return &ViettelPublisher{client: client, cfg: cfg, sellerCfg: sellerCfg, log: log}
}

// CreateInvoice maps the domain invoice, calls Viettel, and returns the transactionUuid as externalID.
func (p *ViettelPublisher) CreateInvoice(ctx context.Context, invoice *domain.Invoice) (string, error) {
	viettelReq := MapInvoiceToViettel(invoice, p.cfg, p.sellerCfg)
	transactionUuid := viettelReq.GeneralInvoiceInfo.TransactionUuid

	if err := ValidateViettelRequest(viettelReq); err != nil {
		return "", domain.NewValidationError(err.Error())
	}

	invoice.TransactionUuid = &transactionUuid

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

// ReportToAuthority sends a completed invoice to the tax authority (CQT) via Viettel (§7.36).
func (p *ViettelPublisher) ReportToAuthority(ctx context.Context, transactionUuid, startDate, endDate string) (int, int, error) {
	req := &ReportToAuthorityRequest{
		SupplierTaxCode: p.cfg.SupplierCode,
		TransactionUuid: transactionUuid,
		StartDate:       startDate,
		EndDate:         endDate,
	}

	p.log.Info().
		Str("transaction_uuid", transactionUuid).
		Str("start_date", startDate).
		Str("end_date", endDate).
		Msg("Sending invoice to tax authority (CQT)")

	resp, err := p.client.ReportToAuthorityByTransactionUuid(ctx, req)
	if err != nil {
		return 0, 0, err
	}

	successCount, _ := strconv.Atoi(resp.Success)
	errorCount, _ := strconv.Atoi(resp.Fail)

	if errorCount > 0 && len(resp.ErrorList) > 0 {
		for _, e := range resp.ErrorList {
			p.log.Warn().
				Str("transaction_uuid", e.TransactionUuid).
				Str("message", e.Message).
				Str("detail", e.Detail).
				Msg("Send to tax error detail")
		}
		return successCount, errorCount, domain.NewThirdPartyError(
			"send to tax failed: "+resp.ErrorList[0].Detail, nil)
	}

	p.log.Info().
		Str("transaction_uuid", transactionUuid).
		Int("success", successCount).
		Msg("Invoice sent to tax authority successfully")

	return successCount, errorCount, nil
}
