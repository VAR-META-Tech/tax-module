package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"tax-module/internal/config"
	"tax-module/internal/domain"
)

// --- In-memory token repo for tests ---

type memTokenRepo struct {
	tokens map[string]*domain.AccessToken
}

func newMemTokenRepo() *memTokenRepo {
	return &memTokenRepo{tokens: make(map[string]*domain.AccessToken)}
}

func (r *memTokenRepo) Get(_ context.Context, provider string) (*domain.AccessToken, error) {
	t, ok := r.tokens[provider]
	if !ok {
		return nil, domain.NewNotFoundError("token not found")
	}
	return t, nil
}

func (r *memTokenRepo) Set(_ context.Context, token *domain.AccessToken) error {
	r.tokens[token.Provider] = token
	return nil
}

// --- Tests ---

func TestViettelClient_Login(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/login" {
			t.Errorf("expected path /auth/login, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var req AuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Username != "testuser" || req.Password != "testpass" {
			t.Errorf("unexpected credentials: %s/%s", req.Username, req.Password)
		}

		resp := AuthResponse{
			AccessToken: "token123",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer authServer.Close()

	log := zerolog.Nop()
	cfg := config.ThirdPartyConfig{
		AuthURL:  authServer.URL + "/auth/login",
		Username: "testuser",
		Password: "testpass",
		Timeout:  5e9, // 5s
	}

	tokenRepo := newMemTokenRepo()
	client := NewViettelClient(cfg, tokenRepo, &log)

	token, err := client.getToken(context.Background())
	if err != nil {
		t.Fatalf("getToken: %v", err)
	}
	if token != "token123" {
		t.Errorf("token = %q, want %q", token, "token123")
	}

	// Token should be persisted
	stored, err := tokenRepo.Get(context.Background(), providerName)
	if err != nil {
		t.Fatalf("Get stored token: %v", err)
	}
	if stored.AccessToken != "token123" {
		t.Errorf("stored token = %q, want %q", stored.AccessToken, "token123")
	}
}

func TestViettelClient_CreateInvoice(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(AuthResponse{AccessToken: "tok", ExpiresIn: 3600})
	})

	mux.HandleFunc("/api/InvoiceAPI/InvoiceWS/createInvoice/TAX123", func(w http.ResponseWriter, r *http.Request) {
		cookie := r.Header.Get("Cookie")
		if cookie != "access_token=tok" {
			t.Errorf("Cookie = %q, want %q", cookie, "access_token=tok")
		}

		resp := ViettelInvoiceResponse{
			Result: &InvoiceCreateResult{
				SupplierTaxCode: "TAX123",
				InvoiceNo:       "AA/22E0000001",
				TransactionID:   "txn-001",
				ReservationCode: "rc-001",
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	log := zerolog.Nop()
	cfg := config.ThirdPartyConfig{
		BaseURL:           server.URL + "/api",
		AuthURL:           server.URL + "/auth/login",
		CreateInvoicePath: "/InvoiceAPI/InvoiceWS/createInvoice",
		SupplierCode:      "TAX123",
		Username:          "u",
		Password:          "p",
		Timeout:           5e9,
	}

	client := NewViettelClient(cfg, newMemTokenRepo(), &log)

	req := &ViettelInvoiceRequest{}
	resp, err := client.CreateInvoice(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("resp.Result is nil")
	}
	if resp.Result.InvoiceNo != "AA/22E0000001" {
		t.Errorf("InvoiceNo = %q, want %q", resp.Result.InvoiceNo, "AA/22E0000001")
	}
}

func TestViettelClient_401Retry(t *testing.T) {
	callCount := 0

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(AuthResponse{AccessToken: "new_tok", ExpiresIn: 3600})
	})

	mux.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		w.Write([]byte(`{"ok":true}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	log := zerolog.Nop()
	cfg := config.ThirdPartyConfig{
		BaseURL:  server.URL + "/api",
		AuthURL:  server.URL + "/auth/login",
		Username: "u",
		Password: "p",
		Timeout:  5e9,
	}

	client := NewViettelClient(cfg, newMemTokenRepo(), &log)

	body, err := client.doAuthenticatedRequest(context.Background(), http.MethodGet, server.URL+"/api/test", "application/json", nil)
	if err != nil {
		t.Fatalf("doAuthenticatedRequest: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("body = %q, want %q", string(body), `{"ok":true}`)
	}
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2 (initial + retry)", callCount)
	}
}

func TestViettelPublisher_CreateInvoice(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(AuthResponse{AccessToken: "tok", ExpiresIn: 3600})
	})

	mux.HandleFunc("/api/InvoiceAPI/InvoiceWS/createInvoice/TAX123", func(w http.ResponseWriter, r *http.Request) {
		var req ViettelInvoiceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}

		if req.BuyerInfo.BuyerLegalName != "Test Corp" {
			t.Errorf("BuyerLegalName = %q, want %q", req.BuyerInfo.BuyerLegalName, "Test Corp")
		}

		resp := ViettelInvoiceResponse{
			Result: &InvoiceCreateResult{
				InvoiceNo:     "INV001",
				TransactionID: req.GeneralInvoiceInfo.TransactionUuid,
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	log := zerolog.Nop()
	cfg := config.ThirdPartyConfig{
		BaseURL:           server.URL + "/api",
		AuthURL:           server.URL + "/auth/login",
		CreateInvoicePath: "/InvoiceAPI/InvoiceWS/createInvoice",
		SupplierCode:      "TAX123",
		InvoiceType:       "1",
		TemplateCode:      "01GTKT0/001",
		InvoiceSeries:     "AA/22E",
		Username:          "u",
		Password:          "p",
		Timeout:           5e9,
	}

	client := NewViettelClient(cfg, newMemTokenRepo(), &log)
	publisher := NewViettelPublisher(client, cfg, config.SellerConfig{}, &log)

	invoice := &domain.Invoice{
		ID:                    uuid.New(),
		BuyerLegalName:        "Test Corp",
		Currency:              "VND",
		TotalAmountWithTax:    11000,
		TotalTaxAmount:        1000,
		TotalAmountWithoutTax: 10000,
		Items: []*domain.InvoiceItem{
			{ItemName: "Service", Quantity: 1, UnitPrice: 10000, TaxPercentage: 10, TaxAmount: 1000, ItemTotalAmountWithoutTax: 10000, ItemTotalAmountWithTax: 11000},
		},
	}

	externalID, err := publisher.CreateInvoice(context.Background(), invoice)
	if err != nil {
		t.Fatalf("CreateInvoice: %v", err)
	}
	if externalID != "INV001" {
		t.Errorf("externalID = %q, want %q", externalID, "INV001")
	}
}

func TestViettelPublisher_QueryStatus_Completed(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(AuthResponse{AccessToken: "tok", ExpiresIn: 3600})
	})

	mux.HandleFunc("/api/InvoiceAPI/InvoiceWS/searchInvoiceByTransactionUuid/TAX123/txn-123", func(w http.ResponseWriter, r *http.Request) {
		resp := ViettelSearchResponse{
			Result: []SearchResult{
				{InvoiceNo: "INV001", TransactionID: "txn-123"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	log := zerolog.Nop()
	cfg := config.ThirdPartyConfig{
		BaseURL:         server.URL + "/api",
		AuthURL:         server.URL + "/auth/login",
		QueryStatusPath: "/InvoiceAPI/InvoiceWS/searchInvoiceByTransactionUuid",
		SupplierCode:    "TAX123",
		Username:        "u",
		Password:        "p",
		Timeout:         5e9,
	}

	client := NewViettelClient(cfg, newMemTokenRepo(), &log)
	publisher := NewViettelPublisher(client, cfg, config.SellerConfig{}, &log)

	status, rawResp, err := publisher.QueryStatus(context.Background(), "txn-123")
	if err != nil {
		t.Fatalf("QueryStatus: %v", err)
	}
	if status != "completed" {
		t.Errorf("status = %q, want %q", status, "completed")
	}
	if rawResp == nil {
		t.Error("rawResp should not be nil")
	}
}

func TestViettelPublisher_QueryStatus_Pending(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(AuthResponse{AccessToken: "tok", ExpiresIn: 3600})
	})

	mux.HandleFunc("/api/InvoiceAPI/InvoiceWS/searchInvoiceByTransactionUuid/TAX123/txn-456", func(w http.ResponseWriter, r *http.Request) {
		resp := ViettelSearchResponse{
			Result: []SearchResult{},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	log := zerolog.Nop()
	cfg := config.ThirdPartyConfig{
		BaseURL:         server.URL + "/api",
		AuthURL:         server.URL + "/auth/login",
		QueryStatusPath: "/InvoiceAPI/InvoiceWS/searchInvoiceByTransactionUuid",
		SupplierCode:    "TAX123",
		Username:        "u",
		Password:        "p",
		Timeout:         5e9,
	}

	client := NewViettelClient(cfg, newMemTokenRepo(), &log)
	publisher := NewViettelPublisher(client, cfg, config.SellerConfig{}, &log)

	status, _, err := publisher.QueryStatus(context.Background(), "txn-456")
	if err != nil {
		t.Fatalf("QueryStatus: %v", err)
	}
	if status != "pending" {
		t.Errorf("status = %q, want %q", status, "pending")
	}
}

// --- ReportToAuthority tests ---

func TestViettelClient_ReportToAuthorityByTransactionUuid(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(AuthResponse{AccessToken: "tok", ExpiresIn: 3600})
	})

	mux.HandleFunc("/api/InvoiceAPI/InvoiceWS/sendInvoiceByTransactionUuid", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		ct := r.Header.Get("Content-Type")
		if ct != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q, want %q", ct, "application/x-www-form-urlencoded")
		}

		cookie := r.Header.Get("Cookie")
		if cookie != "access_token=tok" {
			t.Errorf("Cookie = %q, want %q", cookie, "access_token=tok")
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if r.FormValue("supplierTaxCode") != "TAX123" {
			t.Errorf("supplierTaxCode = %q, want %q", r.FormValue("supplierTaxCode"), "TAX123")
		}
		if r.FormValue("transactionUuid") != "txn-abc" {
			t.Errorf("transactionUuid = %q, want %q", r.FormValue("transactionUuid"), "txn-abc")
		}
		if r.FormValue("startDate") != "2025-01-01" {
			t.Errorf("startDate = %q, want %q", r.FormValue("startDate"), "2025-01-01")
		}
		if r.FormValue("endDate") != "2025-01-01" {
			t.Errorf("endDate = %q, want %q", r.FormValue("endDate"), "2025-01-01")
		}

		resp := ReportToAuthorityResponse{
			Total:     "1",
			Success:   "1",
			Fail:      "0",
			ErrorList: []ErrorDetail{},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	log := zerolog.Nop()
	cfg := config.ThirdPartyConfig{
		BaseURL:       server.URL + "/api",
		AuthURL:       server.URL + "/auth/login",
		ReportToAuthorityPath: "/InvoiceAPI/InvoiceWS/sendInvoiceByTransactionUuid",
		SupplierCode:  "TAX123",
		Username:      "u",
		Password:      "p",
		Timeout:       5e9,
	}

	client := NewViettelClient(cfg, newMemTokenRepo(), &log)

	req := &ReportToAuthorityRequest{
		SupplierTaxCode: "TAX123",
		TransactionUuid: "txn-abc",
		StartDate:       "2025-01-01",
		EndDate:         "2025-01-01",
	}
	resp, err := client.ReportToAuthorityByTransactionUuid(context.Background(), req)
	if err != nil {
		t.Fatalf("ReportToAuthorityByTransactionUuid: %v", err)
	}
	if resp.Success != "1" {
		t.Errorf("Success = %q, want %q", resp.Success, "1")
	}
	if resp.Fail != "0" {
		t.Errorf("Fail = %q, want %q", resp.Fail, "0")
	}
}

func TestViettelPublisher_ReportToAuthority_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(AuthResponse{AccessToken: "tok", ExpiresIn: 3600})
	})

	mux.HandleFunc("/api/InvoiceAPI/InvoiceWS/sendInvoiceByTransactionUuid", func(w http.ResponseWriter, r *http.Request) {
		resp := ReportToAuthorityResponse{
			Total:     "1",
			Success:   "1",
			Fail:      "0",
			ErrorList: []ErrorDetail{},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	log := zerolog.Nop()
	cfg := config.ThirdPartyConfig{
		BaseURL:       server.URL + "/api",
		AuthURL:       server.URL + "/auth/login",
		ReportToAuthorityPath: "/InvoiceAPI/InvoiceWS/sendInvoiceByTransactionUuid",
		SupplierCode:  "TAX123",
		Username:      "u",
		Password:      "p",
		Timeout:       5e9,
	}

	client := NewViettelClient(cfg, newMemTokenRepo(), &log)
	publisher := NewViettelPublisher(client, cfg, config.SellerConfig{}, &log)

	txnUuid := "txn-abc-123"

	successCount, errorCount, err := publisher.ReportToAuthority(context.Background(), txnUuid, "2026-03-01", "2026-03-31")
	if err != nil {
		t.Fatalf("ReportToAuthority: %v", err)
	}
	if successCount != 1 {
		t.Errorf("successCount = %d, want 1", successCount)
	}
	if errorCount != 0 {
		t.Errorf("errorCount = %d, want 0", errorCount)
	}
}

func TestViettelPublisher_ReportToAuthority_PartialFailure(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(AuthResponse{AccessToken: "tok", ExpiresIn: 3600})
	})

	mux.HandleFunc("/api/InvoiceAPI/InvoiceWS/sendInvoiceByTransactionUuid", func(w http.ResponseWriter, r *http.Request) {
		resp := ReportToAuthorityResponse{
			Total:   "2",
			Success: "1",
			Fail:    "1",
			ErrorList: []ErrorDetail{
				{
					TransactionUuid: "txn-fail",
					Detail:          "Không tìm thấy bản ghi cần gửi sang CQT",
					Message:         "INVOCIE_NOT_FOUND",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	log := zerolog.Nop()
	cfg := config.ThirdPartyConfig{
		BaseURL:       server.URL + "/api",
		AuthURL:       server.URL + "/auth/login",
		ReportToAuthorityPath: "/InvoiceAPI/InvoiceWS/sendInvoiceByTransactionUuid",
		SupplierCode:  "TAX123",
		Username:      "u",
		Password:      "p",
		Timeout:       5e9,
	}

	client := NewViettelClient(cfg, newMemTokenRepo(), &log)
	publisher := NewViettelPublisher(client, cfg, config.SellerConfig{}, &log)

	txnUuid := "txn-abc-123"

	successCount, errorCount, err := publisher.ReportToAuthority(context.Background(), txnUuid, "2026-03-01", "2026-03-31")
	if err == nil {
		t.Fatal("expected error for partial failure, got nil")
	}
	if successCount != 1 {
		t.Errorf("successCount = %d, want 1", successCount)
	}
	if errorCount != 1 {
		t.Errorf("errorCount = %d, want 1", errorCount)
	}
}

func TestViettelPublisher_ReportToAuthority_EmptyTransactionUuid(t *testing.T) {
	log := zerolog.Nop()
	cfg := config.ThirdPartyConfig{}
	client := NewViettelClient(cfg, newMemTokenRepo(), &log)
	publisher := NewViettelPublisher(client, cfg, config.SellerConfig{}, &log)

	// Calling with empty transactionUuid will fail at the HTTP call level
	_, _, err := publisher.ReportToAuthority(context.Background(), "", "2026-03-01", "2026-03-31")
	if err == nil {
		t.Fatal("expected error for empty transaction_uuid, got nil")
	}
}
