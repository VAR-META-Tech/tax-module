package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"tax-module/internal/domain"
)

// AccessTokenRepo implements domain.AccessTokenRepository using PostgreSQL.
type AccessTokenRepo struct {
	pool *pgxpool.Pool
	log  *zerolog.Logger
}

func NewAccessTokenRepo(pool *pgxpool.Pool, log *zerolog.Logger) *AccessTokenRepo {
	return &AccessTokenRepo{pool: pool, log: log}
}

// Get retrieves the current access token for the given provider.
func (r *AccessTokenRepo) Get(ctx context.Context, provider string) (*domain.AccessToken, error) {
	query := `
		SELECT id, provider, access_token, token_type, expires_at,
		       refresh_token, scope, raw_response, created_at, updated_at
		FROM access_tokens
		WHERE provider = $1`

	row := r.pool.QueryRow(ctx, query, provider)

	var t domain.AccessToken
	err := row.Scan(
		&t.ID,
		&t.Provider,
		&t.AccessToken,
		&t.TokenType,
		&t.ExpiresAt,
		&t.RefreshToken,
		&t.Scope,
		&t.RawResponse,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewNotFoundError("access token not found for provider: " + provider)
		}
		return nil, domain.NewInternalError("failed to get access token", err)
	}

	return &t, nil
}

// Set upserts the access token for the given provider.
// Uses ON CONFLICT to replace existing token for the same provider.
func (r *AccessTokenRepo) Set(ctx context.Context, token *domain.AccessToken) error {
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	now := time.Now()
	token.UpdatedAt = now
	if token.CreatedAt.IsZero() {
		token.CreatedAt = now
	}

	query := `
		INSERT INTO access_tokens (id, provider, access_token, token_type, expires_at,
		                           refresh_token, scope, raw_response, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (provider) DO UPDATE SET
			access_token  = EXCLUDED.access_token,
			token_type    = EXCLUDED.token_type,
			expires_at    = EXCLUDED.expires_at,
			refresh_token = EXCLUDED.refresh_token,
			scope         = EXCLUDED.scope,
			raw_response  = EXCLUDED.raw_response,
			updated_at    = EXCLUDED.updated_at`

	_, err := r.pool.Exec(ctx, query,
		token.ID,
		token.Provider,
		token.AccessToken,
		token.TokenType,
		token.ExpiresAt,
		token.RefreshToken,
		token.Scope,
		token.RawResponse,
		token.CreatedAt,
		token.UpdatedAt,
	)
	if err != nil {
		return domain.NewInternalError("failed to set access token", err)
	}

	r.log.Info().
		Str("provider", token.Provider).
		Time("expires_at", token.ExpiresAt).
		Msg("Access token saved")

	return nil
}
