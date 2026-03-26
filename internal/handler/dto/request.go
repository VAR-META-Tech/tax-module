package dto

// --- Invoice requests ---

type CreateInvoiceRequest struct {
	CustomerName     string   `json:"customer_name" binding:"required,max=400"`
	CustomerTaxID    string   `json:"customer_tax_id" binding:"max=20"`
	CustomerAddress  string   `json:"customer_address" binding:"max=1200"`
	Currency         string   `json:"currency" binding:"required,len=3"`
	OriginalCurrency string   `json:"original_currency" binding:"required,max=10"`
	ExchangeRate     *float64 `json:"exchange_rate"`
	TransactionHash  string   `json:"transaction_hash" binding:"max=255"`
	Notes            string   `json:"notes" binding:"max=500"`
	IssuedAt         *string  `json:"issued_at"`
	DueAt            *string  `json:"due_at"`
}

type UpdateInvoiceRequest struct {
	CustomerName     string   `json:"customer_name" binding:"required,max=400"`
	CustomerTaxID    string   `json:"customer_tax_id" binding:"max=20"`
	CustomerAddress  string   `json:"customer_address" binding:"max=1200"`
	Currency         string   `json:"currency" binding:"required,len=3"`
	OriginalCurrency string   `json:"original_currency" binding:"required,max=10"`
	ExchangeRate     *float64 `json:"exchange_rate"`
	TransactionHash  string   `json:"transaction_hash" binding:"max=255"`
	Notes            string   `json:"notes" binding:"max=500"`
	IssuedAt         *string  `json:"issued_at"`
	DueAt            *string  `json:"due_at"`
}

type CancelInvoiceRequest struct {
	Reason string `json:"reason"`
}

// --- Item requests ---

type AddItemRequest struct {
	Description      string            `json:"description" binding:"required,max=500"`
	Quantity         float64           `json:"quantity" binding:"required,gt=0"`
	UnitPrice        float64           `json:"unit_price" binding:"required,gte=0"`
	TaxRate          float64           `json:"tax_rate" binding:"gte=-2,lte=100"`
	SortOrder        int               `json:"sort_order"`
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

// SpecialInfoItem represents a special goods attribute per ND70.
type SpecialInfoItem struct {
	Name  string `json:"name" binding:"required"`
	Value string `json:"value" binding:"required"`
}

// --- Query params ---

type ListInvoicesQuery struct {
	Status string `form:"status"`
	From   string `form:"from"`
	To     string `form:"to"`
	Limit  int    `form:"limit,default=20"`
	Offset int    `form:"offset,default=0"`
}
