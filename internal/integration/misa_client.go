package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"tax-module/internal/config"
	"tax-module/internal/domain"
)

const (
	misaProviderName   = "misa_meinvoice"
	misaTokenBuffer    = 24 * time.Hour // refresh 1 day before expiry (TTL is 14 days)
	misaTokenTTL       = 14 * 24 * time.Hour
)

// MISAClient handles HTTP communication with the MISA MeInvoice Integration API.
type MISAClient struct {
	cfg          config.MISAConfig
	http         *http.Client
	tokenRepo    domain.AccessTokenRepository
	log          *zerolog.Logger
	mu           sync.Mutex    // protects token refresh
	template     *MISATemplate // cached active template
	templateOnce sync.Once
}

// NewMISAClient creates a new MISA API client.
func NewMISAClient(cfg config.MISAConfig, tokenRepo domain.AccessTokenRepository, log *zerolog.Logger) *MISAClient {
	return &MISAClient{
		cfg:       cfg,
		http:      &http.Client{Timeout: cfg.Timeout},
		tokenRepo: tokenRepo,
		log:       log,
	}
}

// login authenticates with MISA and persists the token.
func (c *MISAClient) login(ctx context.Context) error {
	body, err := json.Marshal(MISAAuthRequest{
		AppID:    c.cfg.AppID,
		TaxCode:  c.cfg.TaxCode,
		Username: c.cfg.Username,
		Password: c.cfg.Password,
	})
	if err != nil {
		return fmt.Errorf("misa login marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/auth/token", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("misa login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("misa login http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("misa login read body: %w", err)
	}

	var authResp MISAAuthResponse
	if err := json.Unmarshal(raw, &authResp); err != nil {
		return fmt.Errorf("misa login decode: %w", err)
	}

	if !authResp.Success || authResp.Data == "" {
		return fmt.Errorf("misa login failed: %s %s", authResp.ErrorCode, authResp.Errors)
	}

	expiresAt := time.Now().Add(misaTokenTTL)
	token := &domain.AccessToken{
		Provider:    misaProviderName,
		AccessToken: authResp.Data,
		ExpiresAt:   expiresAt,
		RawResponse: raw,
	}
	if err := c.tokenRepo.Set(ctx, token); err != nil {
		return fmt.Errorf("misa save token: %w", err)
	}

	c.log.Info().
		Str("provider", misaProviderName).
		Time("expires_at", expiresAt).
		Msg("MISA token refreshed")

	return nil
}

// getToken returns a valid Bearer token, refreshing if needed.
func (c *MISAClient) getToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	token, err := c.tokenRepo.Get(ctx, misaProviderName)
	if err == nil && token != nil && time.Now().Before(token.ExpiresAt.Add(-misaTokenBuffer)) {
		return token.AccessToken, nil
	}

	if err := c.login(ctx); err != nil {
		return "", err
	}

	token, err = c.tokenRepo.Get(ctx, misaProviderName)
	if err != nil || token == nil {
		return "", fmt.Errorf("misa get token after login: %w", err)
	}
	return token.AccessToken, nil
}

// doRequest executes an authenticated request with Bearer token.
func (c *MISAClient) doRequest(ctx context.Context, method, url string, body []byte) ([]byte, error) {
	token, err := c.getToken(ctx)
	if err != nil {
		return nil, err
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("misa build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("misa http %s %s: %w", method, url, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("misa read response: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		// Token may have expired — invalidate and retry once
		c.mu.Lock()
		loginErr := c.login(ctx)
		c.mu.Unlock()
		if loginErr != nil {
			return nil, loginErr
		}
		return c.doRequest(ctx, method, url, body)
	}

	return raw, nil
}

// FetchTemplate retrieves the active invoice template from MISA.
// The result is cached in memory — call ResetTemplate to force a refresh.
func (c *MISAClient) FetchTemplate(ctx context.Context) (*MISATemplate, error) {
	var fetchErr error
	c.templateOnce.Do(func() {
		url := fmt.Sprintf("%s/invoice/templates?invoiceWithCode=%v&ticket=false",
			c.cfg.BaseURL, c.cfg.InvoiceWithCode)

		raw, err := c.doRequest(ctx, http.MethodGet, url, nil)
		if err != nil {
			fetchErr = err
			return
		}

		var listResp MISATemplateListResponse
		if err := json.Unmarshal(raw, &listResp); err != nil {
			fetchErr = fmt.Errorf("misa template decode: %w", err)
			return
		}
		if !listResp.Success {
			fetchErr = fmt.Errorf("misa template fetch failed: %s", listResp.ErrorCode)
			return
		}

		for i := range listResp.Data {
			t := &listResp.Data[i]
			if !t.Inactive {
				c.template = t
				return
			}
		}
		fetchErr = fmt.Errorf("misa: no active invoice template found")
	})

	if fetchErr != nil {
		return nil, fetchErr
	}
	if c.template == nil {
		return nil, fmt.Errorf("misa: template not loaded")
	}
	return c.template, nil
}

// ResetTemplate clears the cached template so the next call to FetchTemplate re-fetches.
func (c *MISAClient) ResetTemplate() {
	c.template = nil
	c.templateOnce = sync.Once{}
}

// PublishInvoice sends invoices to MISA for signing and publishing.
func (c *MISAClient) PublishInvoice(ctx context.Context, req *MISAPublishRequest) (*MISAPublishResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("misa publish marshal: %w", err)
	}

	raw, err := c.doRequest(ctx, http.MethodPost, c.cfg.BaseURL+"/invoice", body)
	if err != nil {
		return nil, err
	}

	var resp MISAPublishResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("misa publish decode: %w", err)
	}
	return &resp, nil
}

// GetInvoiceStatus queries invoice status by RefID (inputType=2).
func (c *MISAClient) GetInvoiceStatus(ctx context.Context, refIDs []string) (*MISAStatusResponse, error) {
	body, err := json.Marshal(refIDs)
	if err != nil {
		return nil, fmt.Errorf("misa status marshal: %w", err)
	}

	url := fmt.Sprintf("%s/invoice/status?inputType=2&invoiceWithCode=%v&invoiceCalcu=%v",
		c.cfg.BaseURL, c.cfg.InvoiceWithCode, c.cfg.InvoiceCalcu)

	raw, err := c.doRequest(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	var resp MISAStatusResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("misa status decode: %w", err)
	}
	return &resp, nil
}

// DownloadInvoice downloads the invoice file (PDF or XML) by TransactionID.
func (c *MISAClient) DownloadInvoice(ctx context.Context, transactionIDs []string, fileType string) (*MISADownloadResponse, error) {
	body, err := json.Marshal(transactionIDs)
	if err != nil {
		return nil, fmt.Errorf("misa download marshal: %w", err)
	}

	url := fmt.Sprintf("%s/invoice/download?downloadDataType=%s&invoiceWithCode=%v&invoiceCalcu=%v",
		c.cfg.BaseURL, fileType, c.cfg.InvoiceWithCode, c.cfg.InvoiceCalcu)

	raw, err := c.doRequest(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	var resp MISADownloadResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("misa download decode: %w", err)
	}
	return &resp, nil
}
