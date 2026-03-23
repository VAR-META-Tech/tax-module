package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"tax-module/internal/domain"
	"tax-module/internal/handler/dto"
	"tax-module/internal/service"
)

type InvoiceHandler struct {
	svc *service.InvoiceService
	log *zerolog.Logger
}

func NewInvoiceHandler(svc *service.InvoiceService, log *zerolog.Logger) *InvoiceHandler {
	return &InvoiceHandler{svc: svc, log: log}
}

// CreateInvoice godoc POST /api/v1/invoices
func (h *InvoiceHandler) CreateInvoice(c *gin.Context) {
	var req dto.CreateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	if req.OriginalCurrency != "VND" && req.OriginalCurrency != "HBAR" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", "original_currency must be VND or HBAR"))
		return
	}

	exchangeRate := 1.0
	if req.OriginalCurrency == "HBAR" {
		if req.ExchangeRate == nil || *req.ExchangeRate <= 0 {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", "exchange_rate is required and must be > 0 when original_currency is HBAR"))
			return
		}
		exchangeRate = *req.ExchangeRate
	}

	invoice := &domain.Invoice{
		CustomerName:     req.CustomerName,
		CustomerTaxID:    req.CustomerTaxID,
		CustomerAddress:  req.CustomerAddress,
		Currency:         "VND",
		OriginalCurrency: req.OriginalCurrency,
		ExchangeRate:     exchangeRate,
		TransactionHash:  req.TransactionHash,
		Notes:            req.Notes,
	}
	if req.IssuedAt != nil {
		if t, err := time.Parse(time.RFC3339, *req.IssuedAt); err == nil {
			invoice.IssuedAt = &t
		}
	}
	if req.DueAt != nil {
		if t, err := time.Parse(time.RFC3339, *req.DueAt); err == nil {
			invoice.DueAt = &t
		}
	}

	if err := h.svc.CreateDraft(c.Request.Context(), invoice); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse(invoice))
}

// ListInvoices godoc GET /api/v1/invoices
func (h *InvoiceHandler) ListInvoices(c *gin.Context) {
	var q dto.ListInvoicesQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	filter := domain.InvoiceFilter{
		Limit:  q.Limit,
		Offset: q.Offset,
	}
	if q.Status != "" {
		s := domain.InvoiceStatus(q.Status)
		filter.Status = &s
	}
	if q.From != "" {
		if t, err := time.Parse(time.RFC3339, q.From); err == nil {
			filter.FromDate = &t
		}
	}
	if q.To != "" {
		if t, err := time.Parse(time.RFC3339, q.To); err == nil {
			filter.ToDate = &t
		}
	}

	invoices, total, err := h.svc.ListInvoices(c.Request.Context(), filter)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessListResponse(invoices, total, q.Limit, q.Offset))
}

// GetInvoice godoc GET /api/v1/invoices/:id
func (h *InvoiceHandler) GetInvoice(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", "invalid invoice id"))
		return
	}

	invoice, err := h.svc.GetInvoice(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(invoice))
}

// UpdateInvoice godoc PUT /api/v1/invoices/:id
func (h *InvoiceHandler) UpdateInvoice(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", "invalid invoice id"))
		return
	}

	var req dto.UpdateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	if req.OriginalCurrency != "VND" && req.OriginalCurrency != "HBAR" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", "original_currency must be VND or HBAR"))
		return
	}

	exchangeRate := 1.0
	if req.OriginalCurrency == "HBAR" {
		if req.ExchangeRate == nil || *req.ExchangeRate <= 0 {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", "exchange_rate is required and must be > 0 when original_currency is HBAR"))
			return
		}
		exchangeRate = *req.ExchangeRate
	}

	invoice := &domain.Invoice{
		CustomerName:     req.CustomerName,
		CustomerTaxID:    req.CustomerTaxID,
		CustomerAddress:  req.CustomerAddress,
		Currency:         "VND",
		OriginalCurrency: req.OriginalCurrency,
		ExchangeRate:     exchangeRate,
		TransactionHash:  req.TransactionHash,
		Notes:            req.Notes,
	}
	if req.IssuedAt != nil {
		if t, err := time.Parse(time.RFC3339, *req.IssuedAt); err == nil {
			invoice.IssuedAt = &t
		}
	}
	if req.DueAt != nil {
		if t, err := time.Parse(time.RFC3339, *req.DueAt); err == nil {
			invoice.DueAt = &t
		}
	}

	if err := h.svc.UpdateInvoice(c.Request.Context(), id, invoice); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(invoice))
}

// CancelInvoice godoc DELETE /api/v1/invoices/:id
func (h *InvoiceHandler) CancelInvoice(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", "invalid invoice id"))
		return
	}

	var req dto.CancelInvoiceRequest
	_ = c.ShouldBindJSON(&req) // reason is optional

	if err := h.svc.CancelInvoice(c.Request.Context(), id, req.Reason); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{"status": "cancelled"}))
}

// AddItem godoc POST /api/v1/invoices/:id/items
func (h *InvoiceHandler) AddItem(c *gin.Context) {
	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", "invalid invoice id"))
		return
	}

	var req dto.AddItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	item := &domain.InvoiceItem{
		Description: req.Description,
		Quantity:    req.Quantity,
		UnitPrice:   req.UnitPrice,
		TaxRate:     req.TaxRate,
		SortOrder:   req.SortOrder,
	}

	if err := h.svc.AddItem(c.Request.Context(), invoiceID, item); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse(item))
}

// RemoveItem godoc DELETE /api/v1/invoices/:id/items/:itemId
func (h *InvoiceHandler) RemoveItem(c *gin.Context) {
	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", "invalid invoice id"))
		return
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", "invalid item id"))
		return
	}

	if err := h.svc.RemoveItem(c.Request.Context(), invoiceID, itemID); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{"status": "removed"}))
}

// SubmitInvoice godoc POST /api/v1/invoices/:id/submit
func (h *InvoiceHandler) SubmitInvoice(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", "invalid invoice id"))
		return
	}

	if err := h.svc.SubmitInvoice(c.Request.Context(), id); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{"status": "submitted"}))
}

// GetStatus godoc GET /api/v1/invoices/:id/status
func (h *InvoiceHandler) GetStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", "invalid invoice id"))
		return
	}

	invoice, err := h.svc.GetInvoice(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{
		"id":     invoice.ID,
		"status": invoice.Status,
	}))
}

// GetHistory godoc GET /api/v1/invoices/:id/history
func (h *InvoiceHandler) GetHistory(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", "invalid invoice id"))
		return
	}

	history, err := h.svc.GetStatusHistory(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(history))
}

// handleError maps domain.AppError to HTTP response.
func handleError(c *gin.Context, err error) {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.HTTPStatus, dto.ErrorResponse(string(appErr.Code), appErr.Message))
		return
	}
	c.JSON(http.StatusInternalServerError, dto.ErrorResponse("INTERNAL_ERROR", "an unexpected error occurred"))
}
