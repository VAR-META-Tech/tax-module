package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AccessToken stores a token obtained from a 3rd party service.
type AccessToken struct {
	ID           uuid.UUID
	Provider     string // identifies which 3rd party (e.g. "invoice_service")
	AccessToken  string
	TokenType    string // e.g. "Bearer"
	ExpiresAt    time.Time
	RefreshToken string
	Scope        string
	RawResponse  []byte // full JSON response from the token endpoint
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IsExpired returns true if the token has expired (with optional buffer).
func (t *AccessToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsExpiredWithBuffer returns true if the token will expire within the given buffer duration.
func (t *AccessToken) IsExpiredWithBuffer(buffer time.Duration) bool {
	return time.Now().Add(buffer).After(t.ExpiresAt)
}

// AccessTokenRepository defines getter/setter for 3rd party access tokens.
type AccessTokenRepository interface {
	// Get retrieves the current token for the given provider.
	// Returns ErrCodeNotFound if no token exists.
	Get(ctx context.Context, provider string) (*AccessToken, error)

	// Set upserts the access token for the given provider.
	// If a token already exists for this provider, it is replaced.
	Set(ctx context.Context, token *AccessToken) error
}
