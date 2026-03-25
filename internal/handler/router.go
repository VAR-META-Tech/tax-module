package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"tax-module/internal/service"
)

// NewRouter creates the Gin engine with all routes registered.
func NewRouter(log *zerolog.Logger, dbPool *pgxpool.Pool, invoiceSvc *service.InvoiceService) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	inv := NewInvoiceHandler(invoiceSvc, log)

	// System endpoints
	router.GET("/health", healthCheck)
	router.GET("/ready", readinessCheck(dbPool))

	// API v1
	v1 := router.Group("/api/v1")
	{
		invoices := v1.Group("/invoices")
		{
			invoices.POST("", inv.CreateInvoice)
			invoices.GET("", inv.ListInvoices)
			invoices.GET("/:id", inv.GetInvoice)
			invoices.PUT("/:id", inv.UpdateInvoice)
			invoices.DELETE("/:id", inv.CancelInvoice)

			invoices.POST("/:id/items", inv.AddItem)
			invoices.DELETE("/:id/items/:itemId", inv.RemoveItem)

			invoices.POST("/:id/submit", inv.SubmitInvoice)
			invoices.POST("/send-to-tax", inv.SendInvoiceToTax)

			invoices.GET("/:id/status", inv.GetStatus)
			invoices.GET("/:id/history", inv.GetHistory)
		}
	}

	return router
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func readinessCheck(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := pool.Ping(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not ready", "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}
