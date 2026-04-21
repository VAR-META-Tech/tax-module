package dto

// --- Invoice requests ---

type CreateInvoiceRequest struct {
	// Buyer info
	BuyerName      string `json:"buyer_name" binding:"required,max=400"`
	BuyerLegalName string `json:"buyer_legal_name" binding:"max=400"`
	BuyerTaxCode   string `json:"buyer_tax_code" binding:"max=20"`
	BuyerAddress   string `json:"buyer_address" binding:"max=1200"`
	BuyerEmail     string `json:"buyer_email" binding:"max=2000"`
	BuyerPhone     string `json:"buyer_phone" binding:"max=15"`
	BuyerCode      string `json:"buyer_code" binding:"max=400"`

	// VND amounts (provided by ERP)
	Currency              string  `json:"currency" binding:"required,len=3"`
	TotalAmountWithTax    float64 `json:"total_amount_with_tax" binding:"required"`
	TotalTaxAmount        float64 `json:"total_tax_amount"`
	TotalAmountWithoutTax float64 `json:"total_amount_without_tax" binding:"required"`

	// Token/crypto info (stored in DB as evidence)
	TokenCurrency      string   `json:"token_currency" binding:"required,max=10"`
	ExchangeRate       *float64 `json:"exchange_rate"`
	ExchangeRateSource string   `json:"exchange_rate_source" binding:"max=100"`
	TokenTotalAmount   float64  `json:"token_total_amount"`
	TokenTaxAmount     float64  `json:"token_tax_amount"`
	TokenNetAmount     float64  `json:"token_net_amount"`

	// Payment & misc
	PaymentMethod   string  `json:"payment_method" binding:"max=50"`
	TransactionHash string  `json:"transaction_hash" binding:"max=255"`
	ErpOrderID      string  `json:"erp_order_id" binding:"max=255"`
	Notes           string  `json:"notes" binding:"max=500"`
	IssuedAt        *string `json:"issued_at"`

	// Items (required, at least 1)
	Items []ItemRequest `json:"items" binding:"required,min=1,dive"`
}

// ItemRequest represents a line item in a create/update invoice request.
type ItemRequest struct {
	ItemName  string  `json:"item_name" binding:"required,max=500"`
	Quantity  float64 `json:"quantity" binding:"required"`
	UnitPrice float64 `json:"unit_price" binding:"required"`

	// VND amounts (provided by ERP)
	TaxPercentage               float64  `json:"tax_percentage" binding:"gte=-2,lte=100"`
	TaxAmount                   float64  `json:"tax_amount"`
	ItemTotalAmountWithoutTax   float64  `json:"item_total_amount_without_tax" binding:"required"`
	ItemTotalAmountWithTax      float64  `json:"item_total_amount_with_tax"`
	ItemTotalAmountAfterDiscount *float64 `json:"item_total_amount_after_discount"`
	ItemDiscount                *float64 `json:"item_discount"`

	// Token amounts (stored in DB as evidence)
	TokenUnitPrice float64 `json:"token_unit_price"`
	TokenTaxAmount float64 `json:"token_tax_amount"`
	TokenLineTotal float64 `json:"token_line_total"`

	LineNumber       int               `json:"line_number"`
	Selection        *int              `json:"selection" binding:"omitempty,min=1,max=6"`
	ItemType         *int              `json:"item_type" binding:"omitempty,min=1,max=6"`
	ItemCode         string            `json:"item_code" binding:"max=50"`
	UnitCode         string            `json:"unit_code" binding:"max=100"`
	UnitName         string            `json:"unit_name" binding:"max=300"`
	Discount         float64           `json:"discount" binding:"gte=0"`
	Discount2        float64           `json:"discount2" binding:"gte=0"`
	ItemNote         string            `json:"item_note" binding:"max=300"`
	IsIncreaseItem   *bool             `json:"is_increase_item"`
	BatchNo          string            `json:"batch_no" binding:"max=300"`
	ExpDate          string            `json:"exp_date" binding:"max=50"`
	AdjustRatio      string            `json:"adjust_ratio" binding:"omitempty,oneof=1 2 3 5"`
	UnitPriceWithTax *float64          `json:"unit_price_with_tax"`
	SpecialInfo      []SpecialInfoItem `json:"special_info,omitempty"`
}

type UpdatePaymentRequest struct {
	TransactionHash string `json:"transaction_hash" binding:"required,max=255"`
}

// SpecialInfoItem represents a special goods attribute per ND70.
type SpecialInfoItem struct {
	Name  string `json:"name" binding:"required"`
	Value string `json:"value" binding:"required"`
}

// --- Auth requests ---

type LoginRequest struct {
	Provider string `json:"provider" binding:"required,oneof=viettel misa"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	AppID    string `json:"app_id"`   // required when provider == "misa"
	TaxCode  string `json:"tax_code"` // required when provider == "misa"
}

// --- Query params ---

type ListInvoicesQuery struct {
	Status string `form:"status"`
	From   string `form:"from"`
	To     string `form:"to"`
	Limit  int    `form:"limit,default=20"`
	Offset int    `form:"offset,default=0"`
}
