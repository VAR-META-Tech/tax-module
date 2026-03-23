package dto

// --- Invoice requests ---

type CreateInvoiceRequest struct {
	CustomerName    string  `json:"customer_name" binding:"required,max=255"`
	CustomerTaxID   string  `json:"customer_tax_id" binding:"max=50"`
	CustomerAddress string  `json:"customer_address"`
	Currency        string  `json:"currency" binding:"required,len=3"`
	Notes           string  `json:"notes"`
	IssuedAt        *string `json:"issued_at"`
	DueAt           *string `json:"due_at"`
}

type UpdateInvoiceRequest struct {
	CustomerName    string  `json:"customer_name" binding:"required,max=255"`
	CustomerTaxID   string  `json:"customer_tax_id" binding:"max=50"`
	CustomerAddress string  `json:"customer_address"`
	Currency        string  `json:"currency" binding:"required,len=3"`
	Notes           string  `json:"notes"`
	IssuedAt        *string `json:"issued_at"`
	DueAt           *string `json:"due_at"`
}

type CancelInvoiceRequest struct {
	Reason string `json:"reason"`
}

// --- Item requests ---

type AddItemRequest struct {
	Description string  `json:"description" binding:"required,max=500"`
	Quantity    float64 `json:"quantity" binding:"required,gt=0"`
	UnitPrice   float64 `json:"unit_price" binding:"required,gte=0"`
	TaxRate     float64 `json:"tax_rate" binding:"gte=0,lte=100"`
	SortOrder   int     `json:"sort_order"`
}

// --- Query params ---

type ListInvoicesQuery struct {
	Status string `form:"status"`
	From   string `form:"from"`
	To     string `form:"to"`
	Limit  int    `form:"limit,default=20"`
	Offset int    `form:"offset,default=0"`
}
