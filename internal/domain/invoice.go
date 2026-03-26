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
	CustomerName    string         `json:"customer_name"`
	CustomerTaxID   string         `json:"customer_tax_id,omitempty"`
	CustomerAddress string         `json:"customer_address,omitempty"`
	Currency            string         `json:"currency"`
	OriginalCurrency    string         `json:"original_currency"`
	ExchangeRate        float64        `json:"exchange_rate"`
	TotalAmount         float64        `json:"total_amount"`
	TaxAmount           float64        `json:"tax_amount"`
	NetAmount           float64        `json:"net_amount"`
	OriginalTotalAmount float64        `json:"original_total_amount"`
	OriginalTaxAmount   float64        `json:"original_tax_amount"`
	OriginalNetAmount   float64        `json:"original_net_amount"`
	TransactionHash     string         `json:"transaction_hash,omitempty"`
	Notes               string         `json:"notes,omitempty"`
	IssuedAt        *time.Time     `json:"issued_at,omitempty"`
	DueAt           *time.Time     `json:"due_at,omitempty"`
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
	ID          uuid.UUID `json:"id"`
	InvoiceID   uuid.UUID `json:"invoice_id"`
	Description string    `json:"description"`
	Quantity    float64   `json:"quantity"`
	UnitPrice   float64   `json:"unit_price"`
	TaxRate     float64   `json:"tax_rate"`
	TaxAmount         float64   `json:"tax_amount"`
	LineTotal         float64   `json:"line_total"`
	OriginalUnitPrice float64   `json:"original_unit_price"`
	OriginalTaxAmount float64   `json:"original_tax_amount"`
	OriginalLineTotal float64   `json:"original_line_total"`
	SortOrder         int       `json:"sort_order"`
	CreatedAt         time.Time `json:"created_at"`

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
}
