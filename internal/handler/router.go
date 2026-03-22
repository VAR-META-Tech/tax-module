package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// NewRouter creates the Gin engine with all routes registered.
func NewRouter(log *zerolog.Logger) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	// System endpoints
	router.GET("/health", healthCheck)
	// router.GET("/ready", readinessCheck) // TODO: Part 3 — needs DB pool

	// API v1
	// v1 := router.Group("/api/v1")
	// {
	// 	invoices := v1.Group("/invoices")
	// 	{
	// 		invoices.POST("",           createInvoice)     // TODO: Part 5
	// 		invoices.GET("",            listInvoices)      // TODO: Part 5
	// 		invoices.GET("/:id",        getInvoice)        // TODO: Part 5
	// 		invoices.PUT("/:id",        updateInvoice)     // TODO: Part 5
	// 		invoices.DELETE("/:id",     cancelInvoice)     // TODO: Part 5
	//
	// 		invoices.POST("/:id/items",           addItem)        // TODO: Part 5
	// 		invoices.DELETE("/:id/items/:itemId",  removeItem)     // TODO: Part 5
	//
	// 		invoices.POST("/:id/submit",  submitInvoice)   // TODO: Part 5
	//
	// 		invoices.GET("/:id/status",   getStatus)        // TODO: Part 5
	// 		invoices.GET("/:id/history",  getHistory)       // TODO: Part 5
	// 	}
	// }

	return router
}

// healthCheck returns 200 if the process is alive.
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
