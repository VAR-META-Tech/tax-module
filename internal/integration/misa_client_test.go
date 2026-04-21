package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"

	"tax-module/internal/config"
)

func TestMISAClient_LoginWithCredentials(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/token", func(w http.ResponseWriter, r *http.Request) {
		var req MISAAuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if req.Username != "custom_user" || req.Password != "custom_pass" {
			t.Errorf("unexpected credentials: %s/%s", req.Username, req.Password)
		}
		if req.AppID != "APP001" || req.TaxCode != "0100109106" {
			t.Errorf("unexpected appID/taxCode: %s/%s", req.AppID, req.TaxCode)
		}
		resp := map[string]interface{}{
			"Success": true,
			"Data":    "misa_token_abc",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	log := zerolog.Nop()
	cfg := config.MISAConfig{
		BaseURL:  server.URL,
		Username: "cfg_user",
		Password: "cfg_pass",
		AppID:    "CFG_APP",
		TaxCode:  "0000000000",
		Timeout:  5e9,
	}

	tokenRepo := newMemTokenRepo()
	client := NewMISAClient(cfg, tokenRepo, &log)

	expiresAt, err := client.LoginWithCredentials(context.Background(), "custom_user", "custom_pass", "APP001", "0100109106")
	if err != nil {
		t.Fatalf("LoginWithCredentials: %v", err)
	}
	if expiresAt.IsZero() {
		t.Error("expiresAt should not be zero")
	}

	stored, err := tokenRepo.Get(context.Background(), misaProviderName)
	if err != nil {
		t.Fatalf("Get stored token: %v", err)
	}
	if stored.AccessToken != "misa_token_abc" {
		t.Errorf("stored token = %q, want %q", stored.AccessToken, "misa_token_abc")
	}
}

func TestMISAClient_LoginWithCredentials_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"Success":   false,
			"ErrorCode": "INVALID_CREDENTIALS",
			"Errors":    "bad username or password",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	log := zerolog.Nop()
	cfg := config.MISAConfig{
		BaseURL: server.URL,
		Timeout: 5e9,
	}

	client := NewMISAClient(cfg, newMemTokenRepo(), &log)

	_, err := client.LoginWithCredentials(context.Background(), "bad_user", "bad_pass", "APP001", "0100109106")
	if err == nil {
		t.Fatal("expected error for bad credentials, got nil")
	}
}
