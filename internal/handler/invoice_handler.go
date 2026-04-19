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
	svc             *service.InvoiceService
	log             *zerolog.Logger
	defaultProvider string
}

func NewInvoiceHandler(svc *service.InvoiceService, log *zerolog.Logger, defaultProvider string) *InvoiceHandler {
	return &InvoiceHandler{svc: svc, log: log, defaultProvider: defaultProvider}
}

// CreateInvoice godoc POST /api/v1/invoices
// Creates an invoice with items in draft status.
func (h *InvoiceHandler) CreateInvoice(c *gin.Context) {
	var req dto.CreateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	exchangeRate := 1.0
	if req.TokenCurrency != "" && req.TokenCurrency != "VND" {
		if req.ExchangeRate == nil || *req.ExchangeRate <= 0 {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", "exchange_rate is required and must be > 0 when token_currency is not VND"))
			return
		}
		exchangeRate = *req.ExchangeRate
	}

	invoice := &domain.Invoice{
		Provider:              h.defaultProvider,
		BuyerName:             req.BuyerName,
		BuyerLegalName:        req.BuyerLegalName,
		BuyerTaxCode:          req.BuyerTaxCode,
		BuyerAddress:          req.BuyerAddress,
		BuyerEmail:            req.BuyerEmail,
		BuyerPhone:            req.BuyerPhone,
		BuyerCode:             req.BuyerCode,
		Currency:              "VND",
		TotalAmountWithTax:    req.TotalAmountWithTax,
		TotalTaxAmount:        req.TotalTaxAmount,
		TotalAmountWithoutTax: req.TotalAmountWithoutTax,
		TokenCurrency:         req.TokenCurrency,
		ExchangeRate:          exchangeRate,
		ExchangeRateSource:    req.ExchangeRateSource,
		TokenTotalAmount:      req.TokenTotalAmount,
		TokenTaxAmount:        req.TokenTaxAmount,
		TokenNetAmount:        req.TokenNetAmount,
		PaymentMethod:         req.PaymentMethod,
		TransactionHash:       req.TransactionHash,
		ErpOrderID:            req.ErpOrderID,
		Notes:                 req.Notes,
	}
	if req.IssuedAt != nil {
		if t, err := time.Parse(time.RFC3339, *req.IssuedAt); err == nil {
			invoice.IssuedAt = &t
		}
	}

	// Map items
	items := make([]*domain.InvoiceItem, len(req.Items))
	for i, ri := range req.Items {
		items[i] = &domain.InvoiceItem{
			ItemName:                     ri.ItemName,
			Quantity:                     ri.Quantity,
			UnitPrice:                    ri.UnitPrice,
			TaxPercentage:               ri.TaxPercentage,
			TaxAmount:                   ri.TaxAmount,
			ItemTotalAmountWithoutTax:    ri.ItemTotalAmountWithoutTax,
			ItemTotalAmountWithTax:       ri.ItemTotalAmountWithTax,
			ItemTotalAmountAfterDiscount: ri.ItemTotalAmountAfterDiscount,
			ItemDiscount:                 ri.ItemDiscount,
			TokenUnitPrice:               ri.TokenUnitPrice,
			TokenTaxAmount:               ri.TokenTaxAmount,
			TokenLineTotal:               ri.TokenLineTotal,
			LineNumber:                   ri.LineNumber,
			Selection:                    ri.Selection,
			ItemType:                     ri.ItemType,
			ItemCode:                     ri.ItemCode,
			UnitCode:                     ri.UnitCode,
			UnitName:                     ri.UnitName,
			Discount:                     ri.Discount,
			Discount2:                    ri.Discount2,
			ItemNote:                     ri.ItemNote,
			IsIncreaseItem:               ri.IsIncreaseItem,
			BatchNo:                      ri.BatchNo,
			ExpDate:                      ri.ExpDate,
			AdjustRatio:                  ri.AdjustRatio,
			UnitPriceWithTax:             ri.UnitPriceWithTax,
			SpecialInfo:                  toSpecialInfo(ri.SpecialInfo),
		}
	}
	invoice.Items = items

	if err := h.svc.CreateInvoice(c.Request.Context(), invoice); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, dto.SuccessResponse(invoice))
}

// UpdatePayment godoc PATCH /api/v1/invoices/:id/payment
// Saves the blockchain transaction hash after payment is completed.
func (h *InvoiceHandler) UpdatePayment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", "invalid invoice id"))
		return
	}

	var req dto.UpdatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	if err := h.svc.UpdateTransactionHash(c.Request.Context(), id, req.TransactionHash); err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{"transaction_hash": req.TransactionHash}))
}

// SubmitInvoice godoc POST /api/v1/invoices/:id/submit
// Transitions a draft invoice to submitted and enqueues for Viettel publishing.
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

// ReportToAuthority godoc POST /api/v1/invoices/send-to-tax
func (h *InvoiceHandler) ReportToAuthority(c *gin.Context) {
	var req struct {
		TransactionUuid string `json:"transaction_uuid" binding:"required"`
		StartDate       string `json:"start_date" binding:"required"`
		EndDate         string `json:"end_date" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", "transaction_uuid, start_date, end_date are required"))
		return
	}

	successCount, errorCount, err := h.svc.ReportToAuthority(c.Request.Context(), req.TransactionUuid, req.StartDate, req.EndDate)
	if err != nil {
		if errorCount > 0 {
			c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{
				"success_count": successCount,
				"error_count":   errorCount,
			}))
			return
		}
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{
		"success_count": successCount,
		"error_count":   errorCount,
	}))
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

// DownloadPDF godoc GET /api/v1/invoices/:id/pdf
// Downloads the invoice PDF from Viettel and returns it as a Data URL.
func (h *InvoiceHandler) DownloadPDF(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse("VALIDATION_ERROR", "invalid invoice id"))
		return
	}

	fileBase64, invoiceNo, err := h.svc.DownloadInvoiceFile(c.Request.Context(), id)
	if err != nil {
		handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse(gin.H{
		"url":      "data:application/pdf;base64," + fileBase64,
		"filename": "invoice_" + invoiceNo + ".pdf",
	}))
}

func toSpecialInfo(items []dto.SpecialInfoItem) []domain.SpecialInfoItem {
	if len(items) == 0 {
		return nil
	}
	result := make([]domain.SpecialInfoItem, len(items))
	for i, item := range items {
		result[i] = domain.SpecialInfoItem{Name: item.Name, Value: item.Value}
	}
	return result
}

func handleError(c *gin.Context, err error) {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.HTTPStatus, dto.ErrorResponse(string(appErr.Code), appErr.Message))
		return
	}
	c.JSON(http.StatusInternalServerError, dto.ErrorResponse("INTERNAL_ERROR", "an unexpected error occurred"))
}
