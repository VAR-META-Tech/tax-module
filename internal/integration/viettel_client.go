package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"tax-module/internal/config"
	"tax-module/internal/domain"
)

const (
	providerName     = "viettel_sinvoice"
	tokenBufferTime  = 5 * time.Minute
	defaultExpiresIn = 3600 // seconds — fallback if API doesn't return expires_in
)

// ViettelClient handles HTTP communication with the Viettel SInvoice API.
type ViettelClient struct {
	cfg       config.ThirdPartyConfig
	http      *http.Client
	tokenRepo domain.AccessTokenRepository
	log       *zerolog.Logger
	mu        sync.Mutex // protects token refresh
}

// NewViettelClient creates a new Viettel HTTP client.
func NewViettelClient(cfg config.ThirdPartyConfig, tokenRepo domain.AccessTokenRepository, log *zerolog.Logger) *ViettelClient {
	return &ViettelClient{
		cfg: cfg,
		http: &http.Client{
			Timeout: cfg.Timeout,
		},
		tokenRepo: tokenRepo,
		log:       log,
	}
}

// getToken returns a valid access token, refreshing if necessary.
func (c *ViettelClient) getToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Try to get cached token
	token, err := c.tokenRepo.Get(ctx, providerName)
	if err == nil && !token.IsExpiredWithBuffer(tokenBufferTime) {
		return token.AccessToken, nil
	}

	// Token missing or expired — login again
	return c.login(ctx)
}

// login authenticates with Viettel and stores the token.
func (c *ViettelClient) login(ctx context.Context) (string, error) {
	body, err := json.Marshal(AuthRequest{
		Username: c.cfg.Username,
		Password: c.cfg.Password,
	})
	if err != nil {
		return "", domain.NewInternalError("marshal auth request", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.AuthURL, bytes.NewReader(body))
	if err != nil {
		return "", domain.NewInternalError("create auth request", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", domain.NewThirdPartyError("viettel auth request failed", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", domain.NewThirdPartyError("read auth response body", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", domain.NewThirdPartyError(
			fmt.Sprintf("viettel auth returned HTTP %d: %s", resp.StatusCode, string(rawBody)), nil)
	}

	var authResp AuthResponse
	if err := json.Unmarshal(rawBody, &authResp); err != nil {
		return "", domain.NewThirdPartyError("decode auth response", err)
	}

	if authResp.AccessToken == "" {
		return "", domain.NewThirdPartyError("viettel auth returned empty access_token", nil)
	}

	expiresIn := authResp.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = defaultExpiresIn
	}

	token := &domain.AccessToken{
		Provider:    providerName,
		AccessToken: authResp.AccessToken,
		TokenType:   authResp.TokenType,
		ExpiresAt:   time.Now().Add(time.Duration(expiresIn) * time.Second),
		RawResponse: rawBody,
	}

	if err := c.tokenRepo.Set(ctx, token); err != nil {
		c.log.Warn().Err(err).Msg("Failed to persist viettel token — will still use in-memory")
	}

	c.log.Info().
		Time("expires_at", token.ExpiresAt).
		Msg("Viettel SInvoice token obtained")

	return authResp.AccessToken, nil
}

// CreateInvoice sends a create-invoice request to Viettel.
func (c *ViettelClient) CreateInvoice(ctx context.Context, invoiceReq *ViettelInvoiceRequest) (*ViettelInvoiceResponse, error) {
	url := fmt.Sprintf("%s%s/%s", c.cfg.BaseURL, c.cfg.CreateInvoicePath, c.cfg.SupplierCode)

	body, err := json.Marshal(invoiceReq)
	if err != nil {
		return nil, domain.NewInternalError("marshal invoice request", err)
	}

	rawBody, err := c.doAuthenticatedRequest(ctx, http.MethodPost, url, "application/json", body)
	if err != nil {
		return nil, err
	}

	var resp ViettelInvoiceResponse
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, domain.NewThirdPartyError("decode create invoice response", err)
	}

	return &resp, nil
}

// SearchByTransactionUuid queries invoice status by transactionUuid.
func (c *ViettelClient) SearchByTransactionUuid(ctx context.Context, transactionUuid, supplierTaxCode string) (*ViettelSearchResponse, error) {
	url := fmt.Sprintf("%s%s/%s/%s", c.cfg.BaseURL, c.cfg.QueryStatusPath, supplierTaxCode, transactionUuid)

	rawBody, err := c.doAuthenticatedRequest(ctx, http.MethodGet, url, "application/json", nil)
	if err != nil {
		return nil, err
	}

	var resp ViettelSearchResponse
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, domain.NewThirdPartyError("decode search response", err)
	}

	return &resp, nil
}

// doAuthenticatedRequest sends an HTTP request with Viettel token cookie.
// On 401, it re-authenticates once and retries.
func (c *ViettelClient) doAuthenticatedRequest(ctx context.Context, method, url, contentType string, body []byte) ([]byte, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, err
	}

	rawBody, statusCode, err := c.doRequest(ctx, method, url, contentType, body, token)
	if err != nil {
		return nil, err
	}

	// If 401, refresh token and retry once
	if statusCode == http.StatusUnauthorized {
		c.log.Warn().Msg("Viettel returned 401, re-authenticating...")
		c.mu.Lock()
		token, err = c.login(ctx)
		c.mu.Unlock()
		if err != nil {
			return nil, err
		}

		rawBody, statusCode, err = c.doRequest(ctx, method, url, contentType, body, token)
		if err != nil {
			return nil, err
		}
	}

	if statusCode == http.StatusBadRequest {
		vErr := ParseViettelError(rawBody)
		return nil, domain.NewThirdPartyError(vErr.Error(), vErr)
	}

	if statusCode < 200 || statusCode >= 300 {
		return nil, domain.NewThirdPartyError(
			fmt.Sprintf("viettel API returned HTTP %d: %s", statusCode, string(rawBody)), nil)
	}

	return rawBody, nil
}

// ReportToAuthorityByTransactionUuid sends an invoice to the tax authority (CQT) via Viettel (§7.36).
func (c *ViettelClient) ReportToAuthorityByTransactionUuid(ctx context.Context, req *ReportToAuthorityRequest) (*ReportToAuthorityResponse, error) {
	url := fmt.Sprintf("%s%s", c.cfg.BaseURL, c.cfg.ReportToAuthorityPath)

	formData := neturl.Values{}
	formData.Set("supplierTaxCode", req.SupplierTaxCode)
	formData.Set("transactionUuid", req.TransactionUuid)
	formData.Set("startDate", req.StartDate)
	formData.Set("endDate", req.EndDate)

	rawBody, err := c.doAuthenticatedRequest(ctx, http.MethodPost, url, "application/x-www-form-urlencoded", []byte(formData.Encode()))
	if err != nil {
		return nil, err
	}

	var resp ReportToAuthorityResponse
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return nil, domain.NewThirdPartyError("decode send to tax response", err)
	}

	return &resp, nil
}

// doRequest executes a single HTTP request.
func (c *ViettelClient) doRequest(ctx context.Context, method, url, contentType string, body []byte, token string) ([]byte, int, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, 0, domain.NewInternalError("create http request", err)
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Cookie", "access_token="+token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, domain.NewThirdPartyError("viettel API request failed", err)
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, domain.NewThirdPartyError("read response body", err)
	}

	return rawBody, resp.StatusCode, nil
}
