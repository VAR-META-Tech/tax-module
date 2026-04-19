package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"tax-module/internal/domain"
)

var testCtx = context.TODO()

// --- formatVATRate ---

func TestFormatVATRate(t *testing.T) {
	tests := []struct {
		taxPercentage float64
		want          string
	}{
		{10.0, "10%"},
		{0.0, "0%"},
		{5.0, "5%"},
		{8.0, "8%"},
		{-1.0, "KCT"},
		{-2.0, "KKKNT"},
	}
	for _, tc := range tests {
		got := formatVATRate(tc.taxPercentage)
		if got != tc.want {
			t.Errorf("formatVATRate(%v) = %q, want %q", tc.taxPercentage, got, tc.want)
		}
	}
}

// --- amountToWordsVND ---

func TestAmountToWordsVND(t *testing.T) {
	tests := []struct {
		amount float64
		want   string
	}{
		{0, "Không đồng."},
		{1000, "Một nghìn đồng."},
		{5500000, "Năm triệu năm trăm nghìn đồng."},
		{1000000, "Một triệu đồng."},
		{100000, "Một trăm nghìn đồng."},
		{10000, "Mười nghìn đồng."},
	}
	for _, tc := range tests {
		got := amountToWordsVND(tc.amount)
		if got != tc.want {
			t.Errorf("amountToWordsVND(%v) = %q, want %q", tc.amount, got, tc.want)
		}
	}
}

// --- buildTaxRateInfo ---

func TestBuildTaxRateInfo(t *testing.T) {
	itemType1 := 1

	items := []*domain.InvoiceItem{
		{ItemType: &itemType1, TaxPercentage: 10.0, ItemTotalAmountWithoutTax: 100, TaxAmount: 10},
		{ItemType: &itemType1, TaxPercentage: 10.0, ItemTotalAmountWithoutTax: 200, TaxAmount: 20},
		{ItemType: &itemType1, TaxPercentage: 0.0, ItemTotalAmountWithoutTax: 50, TaxAmount: 0},
	}

	result := buildTaxRateInfo(items)

	if len(result) != 2 {
		t.Fatalf("expected 2 tax groups, got %d", len(result))
	}

	// First group: 10%
	if result[0].VATRateName != "10%" {
		t.Errorf("group[0].VATRateName = %q, want %q", result[0].VATRateName, "10%")
	}
	if result[0].AmountWithoutVATOC != 300 {
		t.Errorf("group[0].AmountWithoutVATOC = %v, want 300", result[0].AmountWithoutVATOC)
	}
	if result[0].VATAmountOC != 30 {
		t.Errorf("group[0].VATAmountOC = %v, want 30", result[0].VATAmountOC)
	}

	// Second group: 0%
	if result[1].VATRateName != "0%" {
		t.Errorf("group[1].VATRateName = %q, want %q", result[1].VATRateName, "0%")
	}
	if result[1].AmountWithoutVATOC != 50 {
		t.Errorf("group[1].AmountWithoutVATOC = %v, want 50", result[1].AmountWithoutVATOC)
	}
}

func TestBuildTaxRateInfo_DiscountSubtracted(t *testing.T) {
	itemType1 := 1
	itemType3 := 3 // commercial discount — subtracted

	// 10% group: 500 normal - 100 discount = 400 base, 50 - 10 = 40 VAT
	items := []*domain.InvoiceItem{
		{ItemType: &itemType1, TaxPercentage: 10.0, ItemTotalAmountWithoutTax: 500, TaxAmount: 50},
		{ItemType: &itemType3, TaxPercentage: 10.0, ItemTotalAmountWithoutTax: 100, TaxAmount: 10},
	}

	result := buildTaxRateInfo(items)

	if len(result) != 1 {
		t.Fatalf("expected 1 tax group, got %d", len(result))
	}
	if result[0].AmountWithoutVATOC != 400 {
		t.Errorf("AmountWithoutVATOC = %v, want 400 (500 - 100)", result[0].AmountWithoutVATOC)
	}
	if result[0].VATAmountOC != 40 {
		t.Errorf("VATAmountOC = %v, want 40 (50 - 10)", result[0].VATAmountOC)
	}
}

func TestBuildTaxRateInfo_ExcludedItemTypes(t *testing.T) {
	itemType1 := 1
	itemType2 := 2 // promotion — excluded
	itemType4 := 4 // note — excluded

	// Only ItemType=1 should contribute; types 2 and 4 are ignored
	items := []*domain.InvoiceItem{
		{ItemType: &itemType1, TaxPercentage: 10.0, ItemTotalAmountWithoutTax: 300, TaxAmount: 30},
		{ItemType: &itemType2, TaxPercentage: 10.0, ItemTotalAmountWithoutTax: 999, TaxAmount: 99},
		{ItemType: &itemType4, TaxPercentage: 10.0, ItemTotalAmountWithoutTax: 999, TaxAmount: 99},
	}

	result := buildTaxRateInfo(items)

	if len(result) != 1 {
		t.Fatalf("expected 1 tax group (excluded types skipped), got %d", len(result))
	}
	if result[0].AmountWithoutVATOC != 300 {
		t.Errorf("AmountWithoutVATOC = %v, want 300 (types 2 and 4 excluded)", result[0].AmountWithoutVATOC)
	}
	if result[0].VATAmountOC != 30 {
		t.Errorf("VATAmountOC = %v, want 30", result[0].VATAmountOC)
	}
}

// --- MapInvoiceToMISA ---

func TestMapInvoiceToMISA(t *testing.T) {
	txnUuid := "test-uuid-1234"
	itemType := 1
	invoice := &domain.Invoice{
		ID:                    uuid.New(),
		TransactionUuid:       &txnUuid,
		BuyerName:             "Nguyen Van A",
		BuyerLegalName:        "Cong Ty A",
		BuyerTaxCode:          "0123456789",
		BuyerAddress:          "123 ABC Street",
		BuyerEmail:            "a@example.com",
		BuyerPhone:            "0901234567",
		Currency:              "VND",
		ExchangeRate:          1,
		TotalAmountWithoutTax: 1000000,
		TotalTaxAmount:        100000,
		TotalAmountWithTax:    1100000,
		PaymentMethod:         "TM/CK",
		Items: []*domain.InvoiceItem{
			{
				ItemType:                  &itemType,
				ItemName:                  "Service A",
				Quantity:                  1,
				UnitPrice:                 1000000,
				ItemTotalAmountWithoutTax: 1000000,
				TaxPercentage:             10,
				TaxAmount:                 100000,
			},
		},
	}

	template := &MISATemplate{
		InvSeries:     "C26TAA",
		IsSendSummary: false,
	}

	data := MapInvoiceToMISA(invoice, template, false)

	if data.RefID != txnUuid {
		t.Errorf("RefID = %q, want %q", data.RefID, txnUuid)
	}
	if data.InvSeries != "C26TAA" {
		t.Errorf("InvSeries = %q, want %q", data.InvSeries, "C26TAA")
	}
	if data.BuyerLegalName != "Cong Ty A" {
		t.Errorf("BuyerLegalName = %q, want %q", data.BuyerLegalName, "Cong Ty A")
	}
	if data.BuyerAddress != "123 ABC Street" {
		t.Errorf("BuyerAddress = %q, want %q", data.BuyerAddress, "123 ABC Street")
	}
	if data.TotalAmountOC != 1100000 {
		t.Errorf("TotalAmountOC = %v, want 1100000", data.TotalAmountOC)
	}
	if data.TotalVATAmountOC != 100000 {
		t.Errorf("TotalVATAmountOC = %v, want 100000", data.TotalVATAmountOC)
	}
	if data.TotalAmountWithoutVATOC != 1000000 {
		t.Errorf("TotalAmountWithoutVATOC = %v, want 1000000", data.TotalAmountWithoutVATOC)
	}
	if data.TotalAmountInWords == "" {
		t.Error("TotalAmountInWords should not be empty")
	}

	// Verify items
	if len(data.OriginalInvoiceDetail) != 1 {
		t.Fatalf("expected 1 item, got %d", len(data.OriginalInvoiceDetail))
	}
	item := data.OriginalInvoiceDetail[0]
	if item.VATRateName != "10%" {
		t.Errorf("item.VATRateName = %q, want %q", item.VATRateName, "10%")
	}
	if item.ItemName != "Service A" {
		t.Errorf("item.ItemName = %q, want %q", item.ItemName, "Service A")
	}

	// Verify TaxRateInfo
	if len(data.TaxRateInfo) != 1 {
		t.Fatalf("expected 1 TaxRateInfo group, got %d", len(data.TaxRateInfo))
	}
	if data.TaxRateInfo[0].VATRateName != "10%" {
		t.Errorf("TaxRateInfo[0].VATRateName = %q, want %q", data.TaxRateInfo[0].VATRateName, "10%")
	}
}

// --- DispatchingPublisher.resolve ---

func TestDispatchingPublisher_Routing(t *testing.T) {
	viettelCalled := false
	misaCalled := false

	viettelPub := &stubPublisher{onCreateInvoice: func() { viettelCalled = true }}
	misaPub := &stubPublisher{onCreateInvoice: func() { misaCalled = true }}

	dp := NewDispatchingPublisher(domain.ProviderViettel, viettelPub, misaPub)

	// Route to viettel
	_, _ = dp.CreateInvoice(testCtx,&domain.Invoice{Provider: domain.ProviderViettel})
	if !viettelCalled {
		t.Error("expected viettelPub to be called for provider=viettel")
	}

	// Route to misa
	_, _ = dp.CreateInvoice(testCtx,&domain.Invoice{Provider: domain.ProviderMISA})
	if !misaCalled {
		t.Error("expected misaPub to be called for provider=misa")
	}
}

func TestDispatchingPublisher_DefaultFallback(t *testing.T) {
	viettelCalled := false
	viettelPub := &stubPublisher{onCreateInvoice: func() { viettelCalled = true }}
	misaPub := &stubPublisher{}

	dp := NewDispatchingPublisher(domain.ProviderViettel, viettelPub, misaPub)

	// Empty provider → default (viettel)
	_, _ = dp.CreateInvoice(testCtx,&domain.Invoice{Provider: ""})
	if !viettelCalled {
		t.Error("expected viettelPub to be called when provider is empty (default fallback)")
	}
}

func TestDispatchingPublisher_UnknownProvider(t *testing.T) {
	dp := NewDispatchingPublisher(domain.ProviderViettel, &stubPublisher{}, &stubPublisher{})

	_, err := dp.CreateInvoice(testCtx,&domain.Invoice{Provider: "unknown_provider"})
	if err == nil {
		t.Error("expected error for unknown provider, got nil")
	}
}

// --- MISAError retryable ---

func TestMISAError_Retryable(t *testing.T) {
	if !IsMISARetryable(MISAErrInvoiceDuplicated) {
		t.Errorf("%s should be retryable", MISAErrInvoiceDuplicated)
	}
	if !IsMISARetryable(MISAErrInvoiceNumberNotContinuous) {
		t.Errorf("%s should be retryable", MISAErrInvoiceNumberNotContinuous)
	}
	if IsMISARetryable("InvalidTaxCode") {
		t.Error("InvalidTaxCode should NOT be retryable")
	}
	if IsMISARetryable("LicenseInfo_Expired") {
		t.Error("LicenseInfo_Expired should NOT be retryable")
	}
}

// --- stub publisher for routing tests ---

type stubPublisher struct {
	onCreateInvoice func()
}

func (s *stubPublisher) CreateInvoice(_ context.Context, _ *domain.Invoice) (string, error) {
	if s.onCreateInvoice != nil {
		s.onCreateInvoice()
	}
	return "", nil
}

func (s *stubPublisher) QueryStatus(_ context.Context, _ *domain.Invoice) (string, string, []byte, error) {
	return "", "", nil, nil
}

func (s *stubPublisher) ReportToAuthority(_ context.Context, _, _, _, _ string) (int, int, error) {
	return 0, 0, nil
}

func (s *stubPublisher) DownloadInvoiceFile(_ context.Context, _ string, _ *domain.Invoice, _ string) (string, error) {
	return "", nil
}
