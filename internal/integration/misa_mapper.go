package integration

import (
	"fmt"
	"math"
	"strings"
	"time"

	"tax-module/internal/domain"
)

// MapInvoiceToMISA converts a domain Invoice into a MISAInvoiceData ready for publishing.
// template must be pre-fetched from MISAClient.FetchTemplate.
// invoiceCalcu mirrors the MISA_INVOICE_CALCU config flag.
func MapInvoiceToMISA(invoice *domain.Invoice, template *MISATemplate, invoiceCalcu bool) *MISAInvoiceData {
	invDate := time.Now().Format("2006-01-02")
	if invoice.IssuedAt != nil {
		invDate = invoice.IssuedAt.Format("2006-01-02")
	}

	details := buildMISADetails(invoice.Items)
	taxRateInfo := buildTaxRateInfo(invoice.Items)

	data := &MISAInvoiceData{
		RefID:            safeString(invoice.TransactionUuid),
		InvSeries:        template.InvSeries,
		InvDate:          invDate,
		IsInvoiceSummary: template.IsSendSummary,

		CurrencyCode:                invoice.Currency,
		ExchangeRate:                invoice.ExchangeRate,
		PaymentMethodName:           invoice.PaymentMethod,
		IsInvoiceCalculatingMachine: invoiceCalcu,

		BuyerCode:        invoice.BuyerCode,
		BuyerLegalName:   invoice.BuyerLegalName,
		BuyerTaxCode:     invoice.BuyerTaxCode,
		BuyerAddress:     invoice.BuyerAddress,
		BuyerFullName:    invoice.BuyerName,
		BuyerPhoneNumber: invoice.BuyerPhone,
		BuyerEmail:       invoice.BuyerEmail,
		ReceiverEmail:    invoice.BuyerEmail,

		// All totals are pass-through from ERP — no recalculation
		TotalSaleAmountOC:       invoice.TotalAmountWithoutTax,
		TotalSaleAmount:         invoice.TotalAmountWithoutTax,
		TotalDiscountAmountOC:   0,
		TotalDiscountAmount:     0,
		TotalAmountWithoutVATOC: invoice.TotalAmountWithoutTax,
		TotalAmountWithoutVAT:   invoice.TotalAmountWithoutTax,
		TotalVATAmountOC:        invoice.TotalTaxAmount,
		TotalVATAmount:          invoice.TotalTaxAmount,
		TotalAmountOC:           invoice.TotalAmountWithTax,
		TotalAmount:             invoice.TotalAmountWithTax,
		TotalAmountInWords:      amountToWordsVND(invoice.TotalAmountWithTax),

		OriginalInvoiceDetail: details,
		TaxRateInfo:           taxRateInfo,
	}

	return data
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// buildMISADetails converts domain InvoiceItems to MISA OriginalInvoiceDetail.
func buildMISADetails(items []*domain.InvoiceItem) []MISAInvoiceDetail {
	details := make([]MISAInvoiceDetail, 0, len(items))
	for i, item := range items {
		lineNo := i + 1
		sortOrder := lineNo

		itemType := 1 // default: normal goods
		if item.ItemType != nil {
			itemType = *item.ItemType
		}

		var sortOrderPtr *int
		// SortOrder is null for discount (3) and note (4) item types
		if itemType != 3 && itemType != 4 {
			sortOrderPtr = &sortOrder
		}

		vatRate := formatVATRate(item.TaxPercentage)

		details = append(details, MISAInvoiceDetail{
			ItemType:           itemType,
			SortOrder:          sortOrderPtr,
			LineNumber:         lineNo,
			ItemCode:           item.ItemCode,
			ItemName:           item.ItemName,
			UnitName:           item.UnitName,
			Quantity:           item.Quantity,
			UnitPrice:          item.UnitPrice,
			AmountOC:           item.ItemTotalAmountWithoutTax,
			Amount:             item.ItemTotalAmountWithoutTax,
			DiscountRate:       item.Discount,
			DiscountAmountOC:   item.ItemTotalAmountWithoutTax * item.Discount / 100,
			DiscountAmount:     item.ItemTotalAmountWithoutTax * item.Discount / 100,
			AmountWithoutVATOC: item.ItemTotalAmountWithoutTax - (item.ItemTotalAmountWithoutTax * item.Discount / 100),
			AmountWithoutVAT:   item.ItemTotalAmountWithoutTax - (item.ItemTotalAmountWithoutTax * item.Discount / 100),
			VATRateName:        vatRate,
			VATAmountOC:        item.TaxAmount,
			VATAmount:          item.TaxAmount,
		})
	}
	return details
}

// buildTaxRateInfo aggregates tax amounts per VATRateName (required by MISA).
// Per MISA doc Section 13.5:
//   - ItemType=1 (normal goods): added to the aggregate
//   - ItemType=3 (commercial discount): subtracted from the aggregate
//   - ItemType=2/4/5 (promotion/note/transport): excluded entirely
func buildTaxRateInfo(items []*domain.InvoiceItem) []MISATaxRateInfo {
	type aggregate struct {
		amountWithoutVAT float64
		vatAmount        float64
	}
	grouped := make(map[string]*aggregate)
	order := make([]string, 0)

	for _, item := range items {
		itemType := 1
		if item.ItemType != nil {
			itemType = *item.ItemType
		}
		if itemType != 1 && itemType != 3 {
			continue
		}

		vatRate := formatVATRate(item.TaxPercentage)
		if _, ok := grouped[vatRate]; !ok {
			grouped[vatRate] = &aggregate{}
			order = append(order, vatRate)
		}
		if itemType == 1 {
			grouped[vatRate].amountWithoutVAT += item.ItemTotalAmountWithoutTax
			grouped[vatRate].vatAmount += item.TaxAmount
		} else { // itemType == 3: commercial discount — subtract
			grouped[vatRate].amountWithoutVAT -= item.ItemTotalAmountWithoutTax
			grouped[vatRate].vatAmount -= item.TaxAmount
		}
	}

	result := make([]MISATaxRateInfo, 0, len(grouped))
	for _, vatRate := range order {
		agg := grouped[vatRate]
		result = append(result, MISATaxRateInfo{
			VATRateName:        vatRate,
			AmountWithoutVATOC: agg.amountWithoutVAT,
			VATAmountOC:        agg.vatAmount,
		})
	}
	return result
}

// formatVATRate converts a numeric tax percentage to MISA's VATRateName string format.
// Convention for special values (matching Viettel's negative codes):
//
//	-1 → "KCT"  (không chịu thuế — not subject to VAT)
//	-2 → "KKKNT" (không kê khai, không nộp thuế)
//	 0 → "0%"
//	 5 → "5%"
//	 8 → "8%"
//	10 → "10%"
func formatVATRate(taxPercentage float64) string {
	switch {
	case taxPercentage < -1.5:
		return "KKKNT"
	case taxPercentage < -0.5:
		return "KCT"
	default:
		// Round to nearest integer to avoid float noise (e.g. 9.9999999...)
		pct := int(math.Round(taxPercentage))
		return fmt.Sprintf("%d%%", pct)
	}
}

// amountToWordsVND converts a VND amount (as float64) to Vietnamese words.
// Example: 5500000 → "Năm triệu năm trăm nghìn đồng."
func amountToWordsVND(amount float64) string {
	n := int64(math.Round(amount))
	if n == 0 {
		return "Không đồng."
	}
	if n < 0 {
		return "Âm " + strings.ToLower(amountToWordsVND(-amount))
	}
	return toWordsVND(n) + "đồng."
}

// toWordsVND converts a positive integer to Vietnamese words (without trailing "đồng.").
func toWordsVND(n int64) string {
	if n == 0 {
		return ""
	}

	units := []string{"", "một", "hai", "ba", "bốn", "năm", "sáu", "bảy", "tám", "chín"}
	teens := []string{"mười", "mười một", "mười hai", "mười ba", "mười bốn", "mười lăm",
		"mười sáu", "mười bảy", "mười tám", "mười chín"}

	var hundreds func(int64) string
	hundreds = func(n int64) string {
		if n == 0 {
			return ""
		}
		if n < 10 {
			return units[n] + " "
		}
		if n < 20 {
			return teens[n-10] + " "
		}
		if n < 100 {
			ten := n / 10
			one := n % 10
			s := units[ten] + " mươi "
			if one == 1 {
				s += "mốt "
			} else if one == 5 {
				s += "lăm "
			} else if one > 0 {
				s += units[one] + " "
			}
			return s
		}
		// 100–999
		h := n / 100
		rem := n % 100
		s := units[h] + " trăm "
		if rem > 0 && rem < 10 {
			s += "lẻ " + units[rem] + " "
		} else if rem >= 10 {
			s += hundreds(rem)
		}
		return s
	}

	type group struct {
		divisor int64
		name    string
	}
	groups := []group{
		{1_000_000_000_000, "nghìn tỷ "},
		{1_000_000_000, "tỷ "},
		{1_000_000, "triệu "},
		{1_000, "nghìn "},
		{1, ""},
	}

	result := ""
	for _, g := range groups {
		if n >= g.divisor {
			chunk := n / g.divisor
			n %= g.divisor
			result += hundreds(chunk) + g.name
		}
	}

	// Capitalize first letter
	r := []rune(strings.TrimSpace(result))
	if len(r) > 0 {
		first := strings.ToUpper(string(r[0]))
		return first + string(r[1:]) + " "
	}
	return result
}
