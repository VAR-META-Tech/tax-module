package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// InvoiceStatus represents the lifecycle state of an invoice.
type InvoiceStatus string

const (
	StatusDraft      InvoiceStatus = "draft"
	StatusSubmitted  InvoiceStatus = "submitted"
	StatusProcessing InvoiceStatus = "processing"
	StatusCompleted  InvoiceStatus = "completed"
	StatusFailed     InvoiceStatus = "failed"
	StatusCancelled  InvoiceStatus = "cancelled"
)

// ValidTransitions defines which status transitions are allowed.
var ValidTransitions = map[InvoiceStatus][]InvoiceStatus{
	StatusDraft:      {StatusSubmitted, StatusCancelled},
	StatusSubmitted:  {StatusProcessing, StatusFailed, StatusCancelled},
	StatusProcessing: {StatusCompleted, StatusFailed, StatusCancelled},
	StatusFailed:     {StatusSubmitted, StatusCancelled}, // retry goes back to submitted
}

// CanTransitionTo checks if the status transition is valid.
func (s InvoiceStatus) CanTransitionTo(target InvoiceStatus) bool {
	allowed, ok := ValidTransitions[s]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == target {
			return true
		}
	}
	return false
}

// Invoice is the main domain entity.
type Invoice struct {
	ID              uuid.UUID      `json:"id"`
	ExternalID      *string        `json:"external_id,omitempty"`
	TransactionUuid *string        `json:"transaction_uuid,omitempty"`
	Status          InvoiceStatus  `json:"status"`

	// Buyer info — maps to Viettel buyerInfo
	BuyerName       string         `json:"buyer_name"`
	BuyerLegalName  string         `json:"buyer_legal_name,omitempty"`
	BuyerTaxCode    string         `json:"buyer_tax_code,omitempty"`
	BuyerAddress    string         `json:"buyer_address,omitempty"`
	BuyerEmail      string         `json:"buyer_email,omitempty"`
	BuyerPhone      string         `json:"buyer_phone,omitempty"`
	BuyerCode       string         `json:"buyer_code,omitempty"`

	// VND amounts — sent to Viettel API
	Currency                string  `json:"currency"`
	TotalAmountWithTax      float64 `json:"total_amount_with_tax"`
	TotalTaxAmount          float64 `json:"total_tax_amount"`
	TotalAmountWithoutTax   float64 `json:"total_amount_without_tax"`

	// Token/crypto amounts — stored in DB as evidence only, NOT sent to Viettel
	TokenCurrency       string  `json:"token_currency"`
	ExchangeRate        float64 `json:"exchange_rate"`
	ExchangeRateSource  string  `json:"exchange_rate_source,omitempty"`
	HbarAmount          float64 `json:"hbar_amount,omitempty"`
	TokenTotalAmount    float64 `json:"token_total_amount"`
	TokenTaxAmount      float64 `json:"token_tax_amount"`
	TokenNetAmount      float64 `json:"token_net_amount"`

	// Payment & blockchain
	PaymentMethod   string         `json:"payment_method,omitempty"`
	TransactionHash string         `json:"transaction_hash,omitempty"`
	ErpOrderID      string         `json:"erp_order_id,omitempty"`

	Notes           string         `json:"notes,omitempty"`
	IssuedAt        *time.Time     `json:"issued_at,omitempty"`
	SubmittedAt     *time.Time     `json:"submitted_at,omitempty"`
	CompletedAt     *time.Time     `json:"completed_at,omitempty"`
	RetryCount      int            `json:"retry_count"`
	LastError       *string        `json:"last_error,omitempty"`
	Metadata        []byte         `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	Items           []*InvoiceItem `json:"items,omitempty"`
}

// InvoiceItem is a line item in an invoice.
type InvoiceItem struct {
	ID        uuid.UUID `json:"id"`
	InvoiceID uuid.UUID `json:"invoice_id"`

	// Core fields — maps to Viettel itemInfo
	ItemName   string  `json:"item_name"`
	Quantity   float64 `json:"quantity"`
	UnitPrice  float64 `json:"unit_price"`

	// VND amounts — sent to Viettel
	TaxPercentage               float64 `json:"tax_percentage"`
	TaxAmount                   float64 `json:"tax_amount"`
	ItemTotalAmountWithoutTax   float64 `json:"item_total_amount_without_tax"`
	ItemTotalAmountWithTax      float64 `json:"item_total_amount_with_tax"`
	ItemTotalAmountAfterDiscount *float64 `json:"item_total_amount_after_discount,omitempty"`
	ItemDiscount                *float64 `json:"item_discount,omitempty"`

	// Token amounts — DB evidence only
	TokenUnitPrice float64 `json:"token_unit_price"`
	TokenTaxAmount float64 `json:"token_tax_amount"`
	TokenLineTotal float64 `json:"token_line_total"`

	LineNumber int       `json:"line_number"`
	CreatedAt  time.Time `json:"created_at"`

	// Viettel ItemInfo fields
	Selection       *int     `json:"selection,omitempty"`         // 1=goods,2=note,3=discount,4=table/fee,5=promo,6=special
	ItemType        *int     `json:"item_type,omitempty"`         // required when selection=6 (ND70)
	ItemCode        string   `json:"item_code,omitempty"`
	UnitCode        string   `json:"unit_code,omitempty"`
	UnitName        string   `json:"unit_name,omitempty"`
	Discount        float64  `json:"discount"`                    // % discount on unit price
	Discount2       float64  `json:"discount2"`                   // second % discount
	ItemNote        string   `json:"item_note,omitempty"`
	IsIncreaseItem  *bool    `json:"is_increase_item,omitempty"`  // nil=normal, false=decrease, true=increase
	BatchNo         string   `json:"batch_no,omitempty"`
	ExpDate         string   `json:"exp_date,omitempty"`
	AdjustRatio     string   `json:"adjust_ratio,omitempty"`
	UnitPriceWithTax *float64          `json:"unit_price_with_tax,omitempty"`
	SpecialInfo      []SpecialInfoItem `json:"special_info,omitempty"` // ND70 special goods attributes
}

// SpecialInfoItem represents a special goods attribute per ND70.
type SpecialInfoItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// InvoiceStatusHistory tracks status transitions.
type InvoiceStatusHistory struct {
	ID         uuid.UUID `json:"id"`
	InvoiceID  uuid.UUID `json:"invoice_id"`
	FromStatus string    `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Reason     string    `json:"reason,omitempty"`
	ChangedBy  string    `json:"changed_by,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// AuditLog records changes to entities for auditing.
type AuditLog struct {
	ID         uuid.UUID `json:"id"`
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	Action     string    `json:"action"`
	Actor      string    `json:"actor,omitempty"`
	OldData    []byte    `json:"old_data,omitempty"`
	NewData    []byte    `json:"new_data,omitempty"`
	RequestID  string    `json:"request_id,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// InvoicePublisher is the port for 3rd party invoice creation (Part 4.2).
type InvoicePublisher interface {
	CreateInvoice(ctx context.Context, invoice *Invoice) (externalID string, err error)
	QueryStatus(ctx context.Context, externalID string) (status string, rawResponse []byte, err error)
	ReportToAuthority(ctx context.Context, transactionUuid, startDate, endDate string) (successCount, errorCount int, err error)
	DownloadInvoiceFile(ctx context.Context, invoiceNo, fileType string) ([]byte, error)
}
