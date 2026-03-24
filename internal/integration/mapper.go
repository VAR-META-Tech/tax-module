package integration

import (
	"time"

	"github.com/google/uuid"

	"tax-module/internal/config"
	"tax-module/internal/domain"
)

// MapInvoiceToViettel converts a domain Invoice into a ViettelInvoiceRequest.
// It reuses invoice.TransactionUuid for idempotent retries. Falls back to a new UUID
// if TransactionUuid is not set (e.g., legacy invoices before this field existed).
func MapInvoiceToViettel(invoice *domain.Invoice, cfg config.ThirdPartyConfig) *ViettelInvoiceRequest {
	transactionUuid := ""
	if invoice.TransactionUuid != nil && *invoice.TransactionUuid != "" {
		transactionUuid = *invoice.TransactionUuid
	} else {
		transactionUuid = uuid.New().String()
	}

	now := time.Now().UnixMilli()
	exchangeRate := 1
	selection := 1 // goods/service

	req := &ViettelInvoiceRequest{
		GeneralInvoiceInfo: GeneralInvoiceInfo{
			InvoiceType:       cfg.InvoiceType,
			TemplateCode:      cfg.TemplateCode,
			InvoiceSeries:     cfg.InvoiceSeries,
			TransactionUuid:   transactionUuid,
			CurrencyCode:      invoice.Currency,
			ExchangeRate:      &exchangeRate,
			AdjustmentType:    "1", // original invoice
			PaymentStatus:     true,
			InvoiceIssuedDate: &now,
			InvoiceNote:       invoice.Notes,
		},
		BuyerInfo: BuyerInfo{
			BuyerLegalName:   invoice.CustomerName,
			BuyerTaxCode:     invoice.CustomerTaxID,
			BuyerAddressLine: invoice.CustomerAddress,
		},
		Payments: []Payment{
			{PaymentMethodName: "TM/CK"},
		},
		ItemInfo:      mapItems(invoice.Items),
		TaxBreakdowns: buildTaxBreakdowns(invoice.Items),
		SummarizeInfo: SummarizeInfo{
			SumOfTotalLineAmountWithoutTax: float64Ptr(invoice.NetAmount),
			TotalAmountWithoutTax:          float64Ptr(invoice.NetAmount),
			TotalTaxAmount:                 float64Ptr(invoice.TaxAmount),
			TotalAmountWithTax:             float64Ptr(invoice.TotalAmount),
		},
	}

	_ = selection // used in mapItems

	return req
}

func mapItems(items []*domain.InvoiceItem) []ItemInfo {
	result := make([]ItemInfo, 0, len(items))
	for i, item := range items {
		lineNum := i + 1
		selection := 1
		itemTotal := item.UnitPrice * item.Quantity
		result = append(result, ItemInfo{
			LineNumber:                &lineNum,
			Selection:                 &selection,
			ItemName:                  item.Description,
			Quantity:                  float64Ptr(item.Quantity),
			UnitPrice:                 float64Ptr(item.UnitPrice),
			ItemTotalAmountWithoutTax: float64Ptr(itemTotal),
			TaxPercentage:             float64Ptr(item.TaxRate),
			TaxAmount:                 float64Ptr(item.TaxAmount),
			ItemTotalAmountWithTax:    float64Ptr(item.LineTotal),
		})
	}
	return result
}

func buildTaxBreakdowns(items []*domain.InvoiceItem) []TaxBreakdown {
	// Group by tax rate
	groups := make(map[float64]*TaxBreakdown)
	for _, item := range items {
		tb, ok := groups[item.TaxRate]
		if !ok {
			tb = &TaxBreakdown{
				TaxPercentage: float64Ptr(item.TaxRate),
				TaxableAmount: float64Ptr(0),
				TaxAmount:     float64Ptr(0),
			}
			groups[item.TaxRate] = tb
		}
		baseAmount := item.UnitPrice * item.Quantity
		*tb.TaxableAmount += baseAmount
		*tb.TaxAmount += item.TaxAmount
	}

	result := make([]TaxBreakdown, 0, len(groups))
	for _, tb := range groups {
		result = append(result, *tb)
	}
	return result
}

func float64Ptr(v float64) *float64 {
	return &v
}

