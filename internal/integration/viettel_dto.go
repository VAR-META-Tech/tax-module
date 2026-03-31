package integration

import (
	"encoding/json"
	"fmt"
)

// ---------------------------------------------------------------------------
// Authentication (§5.5)
// ---------------------------------------------------------------------------

// AuthRequest is the body sent to POST /auth/login.
type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse is the JSON returned by /auth/login.
type AuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type,omitempty"`
	ExpiresIn   int64  `json:"expires_in,omitempty"`
}

// ---------------------------------------------------------------------------
// Create Invoice request (§7.2) — also used for adjustment / replacement
// ---------------------------------------------------------------------------

// ViettelInvoiceRequest is the top-level body for createInvoice,
// createOrUpdateInvoiceDraft, and related endpoints.
type ViettelInvoiceRequest struct {
	GeneralInvoiceInfo GeneralInvoiceInfo `json:"generalInvoiceInfo" validate:"required"`
	BuyerInfo          BuyerInfo          `json:"buyerInfo"`
	SellerInfo         *SellerInfo        `json:"sellerInfo,omitempty" validate:"omitempty"`
	Payments           []Payment          `json:"payments" validate:"required,min=1,dive"`
	ItemInfo           []ItemInfo         `json:"itemInfo" validate:"required,min=1,dive"`
	TaxBreakdowns      []TaxBreakdown     `json:"taxBreakdowns"`
	SummarizeInfo      SummarizeInfo      `json:"summarizeInfo"`
	Metadata           []MetadataEntry    `json:"metadata,omitempty"`
}

// GeneralInvoiceInfo contains the common invoice fields (§6.2).
type GeneralInvoiceInfo struct {
	InvoiceType     string   `json:"invoiceType"`
	TemplateCode    string   `json:"templateCode" validate:"required,max=20"`
	InvoiceSeries   string   `json:"invoiceSeries" validate:"required,max=25"`
	TransactionUuid string   `json:"transactionUuid" validate:"required,min=10,max=36"`
	CurrencyCode    string   `json:"currencyCode" validate:"required,len=3"`
	ExchangeRate    *float64 `json:"exchangeRate,omitempty"`

	// Adjustment / replacement fields
	AdjustmentType        string `json:"adjustmentType,omitempty" validate:"omitempty,oneof=1 3 5"`
	AdjustmentInvoiceType string `json:"adjustmentInvoiceType,omitempty" validate:"omitempty,oneof=1 2"`
	AdjustedNote          string `json:"adjustedNote,omitempty" validate:"max=255"`

	// Original invoice reference (required for adjustment/replacement)
	OriginalInvoiceId        string `json:"originalInvoiceId,omitempty" validate:"omitempty,min=7,max=15"`
	OriginalInvoiceIssueDate *int64 `json:"originalInvoiceIssueDate,omitempty"` // unix millis
	OriginalInvoiceType      string `json:"originalInvoiceType,omitempty" validate:"omitempty,oneof=0 1 2 3 4"`
	OriginalTemplateCode     string `json:"originalTemplateCode,omitempty" validate:"max=20"`

	AdditionalReferenceDesc string `json:"additionalReferenceDesc,omitempty" validate:"max=225"`
	AdditionalReferenceDate *int64 `json:"additionalReferenceDate,omitempty"` // unix millis

	InvoiceIssuedDate  *int64 `json:"invoiceIssuedDate,omitempty"` // unix millis
	InvoiceNote        string `json:"invoiceNote,omitempty" validate:"max=500"`
	PaymentStatus      bool   `json:"paymentStatus"`
	CusGetInvoiceRight *bool  `json:"cusGetInvoiceRight,omitempty"`
	ReservationCode    string `json:"reservationCode,omitempty" validate:"max=100"`
	CertificateSerial  string `json:"certificateSerial,omitempty" validate:"max=100"`
	AdjustAmount20     string `json:"adjustAmount20,omitempty"`
	Validation         *int   `json:"validation,omitempty"` // 0 = skip validation
	DetailedListNo     string `json:"DetailedListNo,omitempty" validate:"max=50"`
	DetailedListDate   string `json:"DetailedListDate,omitempty" validate:"max=50"`
	QrCode             string `json:"qrCode,omitempty" validate:"max=500"`
	OtherTax           string `json:"otherTax,omitempty"`
}

// BuyerInfo contains buyer/customer data (§6.4).
type BuyerInfo struct {
	BuyerName          string `json:"buyerName,omitempty" validate:"max=100"`
	BuyerCode          string `json:"buyerCode,omitempty" validate:"max=400"`
	BuyerLegalName     string `json:"buyerLegalName,omitempty" validate:"max=400"`
	BuyerTaxCode       string `json:"buyerTaxCode,omitempty" validate:"max=20"`
	BuyerBudgetCode    string `json:"buyerBudgetCode,omitempty" validate:"max=7"`
	BuyerAddressLine   string `json:"buyerAddressLine,omitempty" validate:"max=1200"`
	BuyerPhoneNumber   string `json:"buyerPhoneNumber,omitempty" validate:"max=15"`
	BuyerFaxNumber     string `json:"buyerFaxNumber,omitempty"`
	BuyerEmail         string `json:"buyerEmail,omitempty" validate:"max=2000"`
	BuyerBankName      string `json:"buyerBankName,omitempty" validate:"max=200"`
	BuyerBankAccount   string `json:"buyerBankAccount,omitempty" validate:"max=30"`
	BuyerDistrictName  string `json:"buyerDistrictName,omitempty"`
	BuyerCityName      string `json:"buyerCityName,omitempty"`
	BuyerCountryCode   string `json:"buyerCountryCode,omitempty"`
	BuyerIdType        string `json:"buyerIdType,omitempty" validate:"omitempty,oneof=1 3"`
	BuyerIdNo          string `json:"buyerIdNo,omitempty" validate:"max=200"`
	BuyerBirthDay      string `json:"buyerBirthDay,omitempty"`
	BuyerNotGetInvoice *int   `json:"buyerNotGetInvoice,omitempty" validate:"omitempty,oneof=0 1"`
}

// SellerInfo contains seller/vendor data (§6.3).
// Optional — if sellerTaxCode is omitted, Viettel uses the portal config.
type SellerInfo struct {
	SellerLegalName    string `json:"sellerLegalName,omitempty" validate:"required,max=400"`
	SellerTaxCode      string `json:"sellerTaxCode,omitempty" validate:"required,max=20"`
	SellerAddressLine  string `json:"sellerAddressLine,omitempty" validate:"required,max=255"`
	SellerPhoneNumber  string `json:"sellerPhoneNumber,omitempty" validate:"max=50"`
	SellerFaxNumber    string `json:"sellerFaxNumber,omitempty" validate:"max=50"`
	SellerEmail        string `json:"sellerEmail,omitempty" validate:"max=50"`
	SellerBankName     string `json:"sellerBankName,omitempty" validate:"max=400"`
	SellerBankAccount  string `json:"sellerBankAccount,omitempty" validate:"max=30"`
	SellerDistrictName string `json:"sellerDistrictName,omitempty" validate:"max=50"`
	SellerCityName     string `json:"sellerCityName,omitempty" validate:"max=600"`
	SellerCountryCode  string `json:"sellerCountryCode,omitempty" validate:"max=15"`
	SellerWebsite      string `json:"sellerWebsite,omitempty" validate:"max=200"`
	StoreCode          string `json:"storeCode,omitempty" validate:"max=50"`
	StoreName          string `json:"storeName,omitempty" validate:"max=400"`
	MerchantCode       string `json:"merchantCode,omitempty" validate:"max=4"`
	MerchantName       string `json:"merchantName,omitempty" validate:"max=25"`
	MerchantCity       string `json:"merchantCity,omitempty" validate:"max=15"`
}

// Payment describes a payment method entry (§6.5).
type Payment struct {
	PaymentMethod     string `json:"paymentMethod,omitempty" validate:"max=50"`
	PaymentMethodName string `json:"paymentMethodName" validate:"required,max=50"`
}

// ItemInfo represents a line item on the invoice (§6.6).
type ItemInfo struct {
	LineNumber                   *int     `json:"lineNumber,omitempty"`
	Selection                    *int     `json:"selection,omitempty" validate:"omitempty,min=1,max=6"`
	ItemType                     *int     `json:"itemType,omitempty" validate:"omitempty,min=1,max=6"`
	ItemCode                     string   `json:"itemCode,omitempty" validate:"max=50"`
	ItemName                     string   `json:"itemName,omitempty" validate:"max=500"`
	UnitCode                     string   `json:"unitCode,omitempty" validate:"max=100"`
	UnitName                     string   `json:"unitName,omitempty" validate:"max=300"`
	Quantity                     *float64 `json:"quantity,omitempty"`
	UnitPrice                    *float64 `json:"unitPrice,omitempty"`
	UnitPriceWithTax             *float64 `json:"unitPriceWithTax,omitempty"`
	ItemTotalAmountWithoutTax    *float64 `json:"itemTotalAmountWithoutTax" validate:"required"`
	ItemTotalAmountAfterDiscount *float64 `json:"itemTotalAmountAfterDiscount,omitempty"`
	ItemTotalAmountWithTax       *float64 `json:"itemTotalAmountWithTax,omitempty"`
	TaxPercentage                *float64 `json:"taxPercentage,omitempty" validate:"omitempty,gte=-2"`
	TaxAmount                    *float64 `json:"taxAmount,omitempty"`
	Discount                     *float64 `json:"discount,omitempty" validate:"omitempty,gte=0"`
	Discount2                    *float64 `json:"discount2,omitempty" validate:"omitempty,gte=0"`
	ItemDiscount                 *float64 `json:"itemDiscount,omitempty"`
	ItemNote                     string   `json:"itemNote,omitempty" validate:"max=300"`
	IsIncreaseItem               *bool    `json:"isIncreaseItem,omitempty"`
	BatchNo                      string   `json:"batchNo,omitempty" validate:"max=300"`
	ExpDate                      string   `json:"expDate,omitempty" validate:"max=50"`
	AdjustRatio                  string   `json:"adjustRatio,omitempty" validate:"omitempty,oneof=1 2 3 5"`
	SpecialInfo                  []SpecialInfoItem `json:"specialInfo,omitempty" validate:"omitempty,dive"`
}

// SpecialInfoItem represents a special goods attribute per ND70.
type SpecialInfoItem struct {
	Name  string `json:"name,omitempty" validate:"required"`
	Value string `json:"value,omitempty" validate:"required"`
}

// TaxBreakdown groups tax amounts by tax rate (§6.7).
type TaxBreakdown struct {
	TaxPercentage      *float64 `json:"taxPercentage,omitempty"`
	TaxableAmount      *float64 `json:"taxableAmount,omitempty"`
	TaxAmount          *float64 `json:"taxAmount,omitempty"`
	TaxableAmountPos   *bool    `json:"taxableAmountPos,omitempty"` // nil/true=positive, false=negative
	TaxAmountPos       *bool    `json:"taxAmountPos,omitempty"`
	TaxExemptionReason string   `json:"taxExemptionReason,omitempty"`
}

// SummarizeInfo contains invoice totals (§6.8).
type SummarizeInfo struct {
	SumOfTotalLineAmountWithoutTax *float64 `json:"sumOfTotalLineAmountWithoutTax,omitempty"`
	TotalAmountWithoutTax          *float64 `json:"totalAmountWithoutTax,omitempty"`
	TotalTaxAmount                 *float64 `json:"totalTaxAmount,omitempty"`
	TotalAmountWithTax             *float64 `json:"totalAmountWithTax,omitempty"`
	TotalAmountWithTaxInWords      string   `json:"totalAmountWithTaxInWords,omitempty"`
	TotalAmountAfterDiscount       *float64 `json:"totalAmountAfterDiscount,omitempty"`
	DiscountAmount                 *float64 `json:"discountAmount,omitempty"`
	IsTotalAmountPos               *bool    `json:"isTotalAmountPos,omitempty"`
	IsTotalTaxAmountPos            *bool    `json:"isTotalTaxAmountPos,omitempty"`
	IsTotalAmtWithoutTaxPos        *bool    `json:"isTotalAmtWithoutTaxPos,omitempty"`
}

// MetadataEntry represents a dynamic field on the invoice (§6.9).
type MetadataEntry struct {
	KeyTag      string `json:"keyTag,omitempty"`
	ValueType   string `json:"valueType,omitempty"` // text, number, date
	StringValue string `json:"stringValue,omitempty"`
	NumberValue *int   `json:"numberValue,omitempty"`
	DateValue   *int64 `json:"dateValue,omitempty"` // unix millis
	KeyLabel    string `json:"keyLabel,omitempty"`
	IsRequired  *bool  `json:"isRequired,omitempty"`
	IsSeller    *bool  `json:"isSeller,omitempty"`
}

// ---------------------------------------------------------------------------
// Create Invoice response (§7.2)
// ---------------------------------------------------------------------------

// ViettelInvoiceResponse is the JSON returned by createInvoice.
type ViettelInvoiceResponse struct {
	ErrorCode   *string              `json:"errorCode"` // nil on success
	Description *string              `json:"description"`
	Result      *InvoiceCreateResult `json:"result,omitempty"`
}

// InvoiceCreateResult holds the data inside a successful create response.
type InvoiceCreateResult struct {
	SupplierTaxCode string `json:"supplierTaxCode"`
	InvoiceNo       string `json:"invoiceNo"`
	TransactionID   string `json:"transactionID"`
	ReservationCode string `json:"reservationCode"`
	CodeOfTax       string `json:"codeOfTax,omitempty"`
}

// ---------------------------------------------------------------------------
// Search by transactionUuid response (§7.21)
// ---------------------------------------------------------------------------

// ViettelSearchResponse is the JSON returned by searchInvoiceByTransactionUuid.
type ViettelSearchResponse struct {
	ErrorCode   *string        `json:"errorCode"`
	Description *string        `json:"description"`
	Result      []SearchResult `json:"result,omitempty"`
}

// SearchResult holds one invoice record from a search response.
type SearchResult struct {
	InvoiceNo       string `json:"invoiceNo"`
	TransactionID   string `json:"transactionID"`
	SupplierTaxCode string `json:"supplierTaxCode"`
	InvoiceType     string `json:"invoiceType"`
	TemplateCode    string `json:"templateCode"`
	InvoiceSeries   string `json:"invoiceSeries"`
	PaymentStatus   *int   `json:"paymentStatus"`
	IssueDate       *int64 `json:"issueDate"` // unix millis
	CreateTime      *int64 `json:"createTime"`
}

// ---------------------------------------------------------------------------
// Get Invoice File request/response (§7.3)
// ---------------------------------------------------------------------------

// GetInvoiceFileRequest is the body for getInvoiceRepresentationFile.
type GetInvoiceFileRequest struct {
	SupplierTaxCode string `json:"supplierTaxCode"`
	InvoiceNo       string `json:"invoiceNo"`
	TemplateCode    string `json:"templateCode"`
	FileType        string `json:"fileType,omitempty"` // ZIP or PDF
}

// GetInvoiceFileResponse is the JSON returned by getInvoiceRepresentationFile.
// Note: this endpoint returns errorCode as int (e.g. 200), unlike other endpoints that use string.
type GetInvoiceFileResponse struct {
	ErrorCode   int     `json:"errorCode"`
	Description *string `json:"description"`
	FileName    string  `json:"fileName,omitempty"`
	FileToBytes string  `json:"fileToBytes,omitempty"` // base64-encoded file content
}

// ---------------------------------------------------------------------------
// Search Invoices request/response (§7.6)
// ---------------------------------------------------------------------------

// SearchInvoicesRequest is the body for getInvoices/{supplierTaxCode}.
type SearchInvoicesRequest struct {
	InvoiceNo      string `json:"invoiceNo,omitempty"`
	StartDate      string `json:"startDate"` // "2020-05-12"
	EndDate        string `json:"endDate"`   // "2020-05-12"
	InvoiceType    string `json:"invoiceType,omitempty"`
	RowPerPage     int    `json:"rowPerPage"`
	PageNum        int    `json:"pageNum"`
	BuyerTaxCode   string `json:"buyerTaxCode,omitempty"`
	TemplateCode   string `json:"templateCode,omitempty"`
	InvoiceSeri    string `json:"invoiceSeri,omitempty"`
	AdjustmentType string `json:"adjustmentType,omitempty"`
}

// SearchInvoicesResponse is the JSON returned by getInvoices.
type SearchInvoicesResponse struct {
	ErrorCode   *string          `json:"errorCode"`
	Description *string          `json:"description"`
	TotalRow    int64            `json:"totalRow"`
	Invoices    []InvoiceListRow `json:"invoices,omitempty"`
}

// InvoiceListRow is a single row in the search invoices list.
type InvoiceListRow struct {
	InvoiceID       int64    `json:"invoiceId"`
	InvoiceType     string   `json:"invoiceType"`
	AdjustmentType  string   `json:"adjustmentType"`
	TemplateCode    string   `json:"templateCode"`
	InvoiceSeri     string   `json:"invoiceSeri"`
	InvoiceNumber   string   `json:"invoiceNumber"`
	InvoiceNo       string   `json:"invoiceNo"`
	Currency        string   `json:"currency"`
	Total           *float64 `json:"total"`
	IssueDate       *int64   `json:"issueDate"`
	PaymentStatus   *int     `json:"paymentStatus"`
	CreateTime      *int64   `json:"createTime"`
	SupplierTaxCode string   `json:"supplierTaxCode"`
	BuyerTaxCode    string   `json:"buyerTaxCode"`
	TotalBeforeTax  *float64 `json:"totalBeforeTax"`
}

// ---------------------------------------------------------------------------
// Draft Invoice request (§7.8/7.9) — same body as create, different endpoint
// ---------------------------------------------------------------------------

// ViettelDraftRequest is identical to ViettelInvoiceRequest.
// Uses endpoint: createOrUpdateInvoiceDraft/{supplierTaxCode}
type ViettelDraftRequest = ViettelInvoiceRequest

// ---------------------------------------------------------------------------
// Send Invoice to Tax Authority request/response (§7.36)
// ---------------------------------------------------------------------------

// ReportToAuthorityRequest is the form data for sendInvoiceByTransactionUuid.
type ReportToAuthorityRequest struct {
	SupplierTaxCode string `json:"supplierTaxCode"` // required; seller tax code
	TransactionUuid string `json:"transactionUuid"` // required; comma-separated UUIDs (10-36 chars each)
	StartDate       string `json:"startDate"`       // required; format "2019-05-12"
	EndDate         string `json:"endDate"`         // required; format "2019-05-12"
}

// ErrorDetail represents a single error entry in ReportToAuthorityResponse.ErrorList.
type ErrorDetail struct {
	TransactionUuid string `json:"transactionUuid"` // comma-separated UUIDs that share this error
	Detail          string `json:"detail"`           // error description
	Message         string `json:"message"`          // error code (e.g. "INVOCIE_NOT_FOUND")
}

// ReportToAuthorityResponse is the JSON returned by sendInvoiceByTransactionUuid (§7.36).
type ReportToAuthorityResponse struct {
	ErrorCode   *string       `json:"errorCode"`
	Description *string       `json:"description"`
	Total       string        `json:"total"`   // total invoice count
	Success     string        `json:"success"` // successful count
	Fail        string        `json:"fail"`    // failed count
	ErrorList   []ErrorDetail `json:"errorlist"`
}

// ---------------------------------------------------------------------------
// Viettel API error codes (BAD_REQUEST / HTTP 400)
// ---------------------------------------------------------------------------

// ViettelErrCode identifies a specific Viettel API error.
type ViettelErrCode string

const (
	ViettelErrTaxCodeInvalid       ViettelErrCode = "TAX_CODE_INVALID"
	ViettelErrTxnUuidRequired      ViettelErrCode = "TRANSACTION_UUID_REQUIRED"
	ViettelErrTaxCodeRequired      ViettelErrCode = "TAX_CODE_REQUIRED"
	ViettelErrBuyerEmailRequired   ViettelErrCode = "BUYER_EMAIL_REQUIRED"
	ViettelErrNotFoundData         ViettelErrCode = "NOT_FOUND_DATA"
	ViettelErrBuyerEmailFormat     ViettelErrCode = "BUYER_EMAIL_ADDRESS_FORMAT"
	ViettelErrEmailConfigNotActive ViettelErrCode = "EMAIL_CONFIG_NOT_ACTIVE"
	ViettelErrEmailNotConfig       ViettelErrCode = "EMAIL_NOT_CONFIG"
	ViettelErrUnknown              ViettelErrCode = "UNKNOWN"
)

// viettelErrCodeMap maps Viettel message strings to typed error codes.
var viettelErrCodeMap = map[string]ViettelErrCode{
	"TAX_CODE_INVALID":                           ViettelErrTaxCodeInvalid,
	"INVOICE_VALID_INPUT_INVALID_TAX_CODE":       ViettelErrTaxCodeInvalid,
	"INVOICE_VALID_INPUT_INVALID_BUYER_TAX_CODE": ViettelErrTaxCodeInvalid,
	"TRANSACTION_UUID_REQUIRED":                  ViettelErrTxnUuidRequired,
	"TAX_CODE_REQUIRED":                          ViettelErrTaxCodeRequired,
	"BUYER_EMAIL_REQUIRED":                       ViettelErrBuyerEmailRequired,
	"NOT_FOUND_DATA":                             ViettelErrNotFoundData,
	"BUYER_EMAIL_ADDRESS_FORMAT":                 ViettelErrBuyerEmailFormat,
	"EMAIL_CONFIG_NOT_ACTIVE":                    ViettelErrEmailConfigNotActive,
	"EMAIL_NOT_CONFIG":                           ViettelErrEmailNotConfig,
}

// nonRetryableViettelErrors lists error codes that should never be retried.
var nonRetryableViettelErrors = map[ViettelErrCode]bool{
	ViettelErrTaxCodeInvalid:     true,
	ViettelErrTxnUuidRequired:    true,
	ViettelErrTaxCodeRequired:    true,
	ViettelErrBuyerEmailRequired: true,
	ViettelErrBuyerEmailFormat:   true,
	ViettelErrNotFoundData:       true,
}

// ---------------------------------------------------------------------------
// Viettel API error response (generic error format)
// ---------------------------------------------------------------------------

// ViettelErrorResponse represents the error format returned for HTTP 400.
type ViettelErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// ViettelError is a structured error wrapping a Viettel API error response.
type ViettelError struct {
	ErrCode     ViettelErrCode
	RawMessage  string // original message from Viettel API
	Description string // user-facing description
	Retryable   bool
}

func (e *ViettelError) Error() string {
	return fmt.Sprintf("viettel error [%s]: %s", e.ErrCode, e.Description)
}

// ParseViettelError parses a raw 400 response body into a structured ViettelError.
func ParseViettelError(rawBody []byte) *ViettelError {
	var resp ViettelErrorResponse
	if err := json.Unmarshal(rawBody, &resp); err != nil {
		return &ViettelError{
			ErrCode:     ViettelErrUnknown,
			RawMessage:  string(rawBody),
			Description: string(rawBody),
			Retryable:   false,
		}
	}

	code, ok := viettelErrCodeMap[resp.Message]
	if !ok {
		code = ViettelErrUnknown
	}

	desc := resp.Data
	if desc == "" {
		desc = resp.Message
	}

	return &ViettelError{
		ErrCode:     code,
		RawMessage:  resp.Message,
		Description: desc,
		Retryable:   !nonRetryableViettelErrors[code],
	}
}
