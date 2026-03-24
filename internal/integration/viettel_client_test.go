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

	body, err := client.doAuthenticatedRequest(context.Background(), http.MethodGet, server.URL+"/api/test", nil)
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
	publisher := NewViettelPublisher(client, cfg, &log)

	invoice := &domain.Invoice{
		ID:           uuid.New(),
		CustomerName: "Test Corp",
		Currency:     "VND",
		TotalAmount:  11000,
		TaxAmount:    1000,
		NetAmount:    10000,
		Items: []*domain.InvoiceItem{
			{Description: "Service", Quantity: 1, UnitPrice: 10000, TaxRate: 10, TaxAmount: 1000, LineTotal: 11000},
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
	publisher := NewViettelPublisher(client, cfg, &log)

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
	publisher := NewViettelPublisher(client, cfg, &log)

	status, _, err := publisher.QueryStatus(context.Background(), "txn-456")
	if err != nil {
		t.Fatalf("QueryStatus: %v", err)
	}
	if status != "pending" {
		t.Errorf("status = %q, want %q", status, "pending")
	}
}
