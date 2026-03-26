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
func MapInvoiceToViettel(invoice *domain.Invoice, cfg config.ThirdPartyConfig, sellerCfg config.SellerConfig) *ViettelInvoiceRequest {
	transactionUuid := ""
	if invoice.TransactionUuid != nil && *invoice.TransactionUuid != "" {
		transactionUuid = *invoice.TransactionUuid
	} else {
		transactionUuid = uuid.New().String()
	}

	now := time.Now().UnixMilli()
	selection := 1 // goods/service

	req := &ViettelInvoiceRequest{
		GeneralInvoiceInfo: GeneralInvoiceInfo{
			InvoiceType:       cfg.InvoiceType,
			TemplateCode:      cfg.TemplateCode,
			InvoiceSeries:     cfg.InvoiceSeries,
			TransactionUuid:   transactionUuid,
			CurrencyCode:      invoice.Currency,
			ExchangeRate:      float64Ptr(1),
			AdjustmentType:    "1", // original invoice
			PaymentStatus:     true,
			InvoiceIssuedDate: &now,
			InvoiceNote:       invoice.Notes,
			Validation:        intPtr(0),
		},
		SellerInfo: buildSellerInfo(sellerCfg),
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
		if item.Selection != nil {
			selection = *item.Selection
		}
		itemTotal := item.UnitPrice * item.Quantity

		info := ItemInfo{
			LineNumber:                &lineNum,
			Selection:                 &selection,
			ItemType:                  item.ItemType,
			ItemCode:                  item.ItemCode,
			ItemName:                  item.Description,
			UnitCode:                  item.UnitCode,
			UnitName:                  item.UnitName,
			Quantity:                  float64Ptr(item.Quantity),
			UnitPrice:                 float64Ptr(item.UnitPrice),
			UnitPriceWithTax:          item.UnitPriceWithTax,
			ItemTotalAmountWithoutTax: float64Ptr(itemTotal),
			TaxPercentage:             float64Ptr(item.TaxRate),
			TaxAmount:                 float64Ptr(item.TaxAmount),
			ItemTotalAmountWithTax:    float64Ptr(item.LineTotal),
			ItemNote:                  item.ItemNote,
			IsIncreaseItem:            item.IsIncreaseItem,
			BatchNo:                   item.BatchNo,
			ExpDate:                   item.ExpDate,
			AdjustRatio:               item.AdjustRatio,
		}
		if item.Discount != 0 {
			info.Discount = float64Ptr(item.Discount)
		}
		if item.Discount2 != 0 {
			info.Discount2 = float64Ptr(item.Discount2)
		}
		if len(item.SpecialInfo) > 0 {
			si := make([]SpecialInfoItem, len(item.SpecialInfo))
			for j, s := range item.SpecialInfo {
				si[j] = SpecialInfoItem{Name: s.Name, Value: s.Value}
			}
			info.SpecialInfo = si
		}

		result = append(result, info)
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

// buildSellerInfo returns a populated SellerInfo when sellerTaxCode is configured.
// When TaxCode is empty, returns nil so that Viettel uses seller data from the HDDT portal.
func buildSellerInfo(sellerCfg config.SellerConfig) *SellerInfo {
	if sellerCfg.TaxCode == "" {
		return nil
	}
	return &SellerInfo{
		SellerLegalName:   sellerCfg.LegalName,
		SellerTaxCode:     sellerCfg.TaxCode,
		SellerAddressLine: sellerCfg.Address,
		SellerPhoneNumber: sellerCfg.PhoneNumber,
		SellerEmail:       sellerCfg.Email,
		SellerBankName:    sellerCfg.BankName,
		SellerBankAccount: sellerCfg.BankAccount,
	}
}

func float64Ptr(v float64) *float64 {
	return &v
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func intPtr(v int) *int {
	return &v
}
