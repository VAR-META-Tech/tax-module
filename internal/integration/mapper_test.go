package integration

import (
	"testing"
	"time"

	"tax-module/internal/config"
	"tax-module/internal/domain"

	"github.com/google/uuid"
)

func TestMapInvoiceToViettel(t *testing.T) {
	cfg := config.ThirdPartyConfig{
		InvoiceType:   "1",
		TemplateCode:  "01GTKT0/001",
		InvoiceSeries: "AA/22E",
	}

	txnUuid := uuid.New().String()
	invoice := &domain.Invoice{
		ID:              uuid.New(),
		TransactionUuid: &txnUuid,
		CustomerName:    "Cong ty ABC",
		CustomerTaxID:   strPtr("0123456789"),
		CustomerAddress: strPtr("123 Nguyen Hue, HCM"),
		Currency:        "VND",
		TotalAmount:     11000,
		TaxAmount:       1000,
		NetAmount:       10000,
		Notes:           strPtr("Test invoice"),
		Items: []*domain.InvoiceItem{
			{
				ID:          uuid.New(),
				Description: "Service A",
				Quantity:    2,
				UnitPrice:   5000,
				TaxRate:     10,
				TaxAmount:   1000,
				LineTotal:   11000,
			},
		},
	}

	result := MapInvoiceToViettel(invoice, cfg)

	if result.GeneralInvoiceInfo.InvoiceType != "1" {
		t.Errorf("InvoiceType = %q, want %q", result.GeneralInvoiceInfo.InvoiceType, "1")
	}
	if result.GeneralInvoiceInfo.TransactionUuid != txnUuid {
		t.Errorf("TransactionUuid = %q, want %q (should reuse invoice's TransactionUuid)", result.GeneralInvoiceInfo.TransactionUuid, txnUuid)
	}
	if result.GeneralInvoiceInfo.CurrencyCode != "VND" {
		t.Errorf("CurrencyCode = %q, want %q", result.GeneralInvoiceInfo.CurrencyCode, "VND")
	}
	if result.BuyerInfo.BuyerLegalName != "Cong ty ABC" {
		t.Errorf("BuyerLegalName = %q, want %q", result.BuyerInfo.BuyerLegalName, "Cong ty ABC")
	}
	if result.BuyerInfo.BuyerTaxCode != "0123456789" {
		t.Errorf("BuyerTaxCode = %q, want %q", result.BuyerInfo.BuyerTaxCode, "0123456789")
	}
	if len(result.ItemInfo) != 1 {
		t.Fatalf("ItemInfo count = %d, want 1", len(result.ItemInfo))
	}
	item := result.ItemInfo[0]
	if item.ItemName != "Service A" {
		t.Errorf("ItemName = %q, want %q", item.ItemName, "Service A")
	}
	if *item.Quantity != 2 {
		t.Errorf("Quantity = %f, want 2", *item.Quantity)
	}
	if *item.TaxPercentage != 10 {
		t.Errorf("TaxPercentage = %f, want 10", *item.TaxPercentage)
	}
	if len(result.TaxBreakdowns) != 1 {
		t.Fatalf("TaxBreakdowns count = %d, want 1", len(result.TaxBreakdowns))
	}
	tb := result.TaxBreakdowns[0]
	if *tb.TaxableAmount != 10000 {
		t.Errorf("TaxBreakdown.TaxableAmount = %f, want 10000", *tb.TaxableAmount)
	}
	if *result.SummarizeInfo.TotalAmountWithTax != 11000 {
		t.Errorf("TotalAmountWithTax = %f, want 11000", *result.SummarizeInfo.TotalAmountWithTax)
	}
}

func TestMapInvoiceToViettel_MultipleItems(t *testing.T) {
	cfg := config.ThirdPartyConfig{InvoiceType: "1", TemplateCode: "01GTKT0/001", InvoiceSeries: "AA/22E"}
	invoice := &domain.Invoice{
		ID: uuid.New(), Currency: "VND", TotalAmount: 23100, TaxAmount: 2100, NetAmount: 21000,
		Items: []*domain.InvoiceItem{
			{Description: "Item A", Quantity: 1, UnitPrice: 10000, TaxRate: 10, TaxAmount: 1000, LineTotal: 11000},
			{Description: "Item B", Quantity: 2, UnitPrice: 5000, TaxRate: 5, TaxAmount: 500, LineTotal: 10500},
		},
	}
	result := MapInvoiceToViettel(invoice, cfg)
	if len(result.ItemInfo) != 2 {
		t.Fatalf("ItemInfo count = %d, want 2", len(result.ItemInfo))
	}
	if *result.ItemInfo[0].LineNumber != 1 {
		t.Errorf("First item LineNumber = %d, want 1", *result.ItemInfo[0].LineNumber)
	}
	if len(result.TaxBreakdowns) != 2 {
		t.Fatalf("TaxBreakdowns count = %d, want 2", len(result.TaxBreakdowns))
	}
}

func TestMapInvoiceToViettel_EmptyItems(t *testing.T) {
	cfg := config.ThirdPartyConfig{InvoiceType: "1"}
	invoice := &domain.Invoice{ID: uuid.New(), Currency: "VND", Items: []*domain.InvoiceItem{}}
	result := MapInvoiceToViettel(invoice, cfg)
	if len(result.ItemInfo) != 0 {
		t.Errorf("ItemInfo count = %d, want 0", len(result.ItemInfo))
	}
}

func TestMapInvoiceToViettel_FallbackUuid(t *testing.T) {
	cfg := config.ThirdPartyConfig{InvoiceType: "1"}
	// No TransactionUuid set — mapper should generate a new one as fallback.
	invoice := &domain.Invoice{ID: uuid.New(), Currency: "VND", Items: []*domain.InvoiceItem{}}
	result := MapInvoiceToViettel(invoice, cfg)
	if result.GeneralInvoiceInfo.TransactionUuid == "" {
		t.Error("TransactionUuid should be generated as fallback when not set on invoice")
	}
	if _, err := uuid.Parse(result.GeneralInvoiceInfo.TransactionUuid); err != nil {
		t.Errorf("Fallback TransactionUuid is not a valid UUID: %v", err)
	}
}

func TestMapInvoiceToViettel_ReusesUuid(t *testing.T) {
	cfg := config.ThirdPartyConfig{InvoiceType: "1"}
	txnUuid := "550e8400-e29b-41d4-a716-446655440000"
	invoice := &domain.Invoice{
		ID: uuid.New(), Currency: "VND",
		TransactionUuid: &txnUuid,
		Items:           []*domain.InvoiceItem{},
	}

	// Call mapper twice — should return the same UUID both times (idempotent).
	r1 := MapInvoiceToViettel(invoice, cfg)
	r2 := MapInvoiceToViettel(invoice, cfg)
	if r1.GeneralInvoiceInfo.TransactionUuid != txnUuid {
		t.Errorf("First call TransactionUuid = %q, want %q", r1.GeneralInvoiceInfo.TransactionUuid, txnUuid)
	}
	if r2.GeneralInvoiceInfo.TransactionUuid != txnUuid {
		t.Errorf("Second call TransactionUuid = %q, want %q", r2.GeneralInvoiceInfo.TransactionUuid, txnUuid)
	}
}

func TestBuildTaxBreakdowns_GroupsByRate(t *testing.T) {
	items := []*domain.InvoiceItem{
		{UnitPrice: 1000, Quantity: 1, TaxRate: 10, TaxAmount: 100},
		{UnitPrice: 2000, Quantity: 1, TaxRate: 10, TaxAmount: 200},
		{UnitPrice: 3000, Quantity: 1, TaxRate: 5, TaxAmount: 150},
	}
	breakdowns := buildTaxBreakdowns(items)
	if len(breakdowns) != 2 {
		t.Fatalf("breakdowns count = %d, want 2", len(breakdowns))
	}
	var tb10, tb5 *TaxBreakdown
	for i := range breakdowns {
		if *breakdowns[i].TaxPercentage == 10 {
			tb10 = &breakdowns[i]
		}
		if *breakdowns[i].TaxPercentage == 5 {
			tb5 = &breakdowns[i]
		}
	}
	if tb10 == nil {
		t.Fatal("Missing 10%% tax breakdown")
	}
	if *tb10.TaxableAmount != 3000 {
		t.Errorf("10%%%% TaxableAmount = %f, want 3000", *tb10.TaxableAmount)
	}
	if *tb10.TaxAmount != 300 {
		t.Errorf("10%%%% TaxAmount = %f, want 300", *tb10.TaxAmount)
	}
	if tb5 == nil {
		t.Fatal("Missing 5%% tax breakdown")
	}
	if *tb5.TaxableAmount != 3000 {
		t.Errorf("5%%%% TaxableAmount = %f, want 3000", *tb5.TaxableAmount)
	}
}

func TestMapInvoiceToViettel_TimestampFormat(t *testing.T) {
	cfg := config.ThirdPartyConfig{InvoiceType: "1"}
	invoice := &domain.Invoice{ID: uuid.New(), Currency: "VND", Items: []*domain.InvoiceItem{}}
	result := MapInvoiceToViettel(invoice, cfg)
	ts := *result.GeneralInvoiceInfo.InvoiceIssuedDate
	now := time.Now().UnixMilli()
	diff := now - ts
	if diff < 0 || diff > 1000 {
		t.Errorf("InvoiceIssuedDate timestamp diff = %d ms, expected < 1000 ms", diff)
	}
}

func strPtr(s string) *string { return &s }
