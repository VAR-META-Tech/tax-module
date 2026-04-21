package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"tax-module/internal/handler/dto"
	"tax-module/internal/integration"
)

// AuthHandler handles provider authentication requests.
type AuthHandler struct {
	viettelClient *integration.ViettelClient
	misaClient    *integration.MISAClient
	log           *zerolog.Logger
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(viettelClient *integration.ViettelClient, misaClient *integration.MISAClient, log *zerolog.Logger) *AuthHandler {
	return &AuthHandler{
		viettelClient: viettelClient,
		misaClient:    misaClient,
		log:           log,
	}
}

// Login authenticates with the specified provider using the provided credentials.
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	if req.Provider == "misa" {
		if req.AppID == "" || req.TaxCode == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "app_id and tax_code are required for MISA provider",
			})
			return
		}
	}

	ctx := c.Request.Context()

	switch req.Provider {
	case "viettel":
		expiresAt, err := h.viettelClient.LoginWithCredentials(ctx, req.Username, req.Password)
		if err != nil {
			h.log.Warn().Err(err).Str("provider", req.Provider).Msg("Login failed")
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"provider":   "viettel",
				"expires_at": expiresAt,
			},
		})

	case "misa":
		expiresAt, err := h.misaClient.LoginWithCredentials(ctx, req.Username, req.Password, req.AppID, req.TaxCode)
		if err != nil {
			h.log.Warn().Err(err).Str("provider", req.Provider).Msg("Login failed")
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"provider":   "misa",
				"expires_at": expiresAt,
			},
		})
	}
}
