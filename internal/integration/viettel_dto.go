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
	GeneralInvoiceInfo GeneralInvoiceInfo `json:"generalInvoiceInfo"`
	BuyerInfo          BuyerInfo          `json:"buyerInfo"`
	SellerInfo         *SellerInfo        `json:"sellerInfo,omitempty"`
	Payments           []Payment          `json:"payments"`
	ItemInfo           []ItemInfo         `json:"itemInfo"`
	TaxBreakdowns      []TaxBreakdown     `json:"taxBreakdowns"`
	SummarizeInfo      SummarizeInfo      `json:"summarizeInfo"`
	Metadata           []MetadataEntry    `json:"metadata,omitempty"`
}

// GeneralInvoiceInfo contains the common invoice fields (§6.2).
type GeneralInvoiceInfo struct {
	InvoiceType     string `json:"invoiceType"`
	TemplateCode    string `json:"templateCode"`
	InvoiceSeries   string `json:"invoiceSeries"`
	TransactionUuid string `json:"transactionUuid"` // required; min 10, max 36; UUID v4 recommended
	CurrencyCode    string `json:"currencyCode"`
	ExchangeRate    *float64 `json:"exchangeRate,omitempty"` // default 1; BigDecimal (max 11 integer + 2 decimal digits)

	// Adjustment / replacement fields
	AdjustmentType        string `json:"adjustmentType,omitempty"`        // 1=original, 3=replacement, 5=adjustment
	AdjustmentInvoiceType string `json:"adjustmentInvoiceType,omitempty"` // 1=amount, 2=info (when adjustmentType=5)
	AdjustedNote          string `json:"adjustedNote,omitempty"`

	// Original invoice reference (required for adjustment/replacement)
	OriginalInvoiceId        string `json:"originalInvoiceId,omitempty"`
	OriginalInvoiceIssueDate *int64 `json:"originalInvoiceIssueDate,omitempty"` // unix millis
	OriginalInvoiceType      string `json:"originalInvoiceType,omitempty"`      // 0-4
	OriginalTemplateCode     string `json:"originalTemplateCode,omitempty"`

	AdditionalReferenceDesc string `json:"additionalReferenceDesc,omitempty"`
	AdditionalReferenceDate *int64 `json:"additionalReferenceDate,omitempty"` // unix millis

	InvoiceIssuedDate  *int64 `json:"invoiceIssuedDate,omitempty"` // unix millis
	InvoiceNote        string `json:"invoiceNote,omitempty"`
	PaymentStatus      bool   `json:"paymentStatus"`
	CusGetInvoiceRight *bool  `json:"cusGetInvoiceRight,omitempty"`
	ReservationCode    string `json:"reservationCode,omitempty"`
	CertificateSerial  string `json:"certificateSerial,omitempty"`
	AdjustAmount20     string `json:"adjustAmount20,omitempty"`
	Validation         *int   `json:"validation,omitempty"` // 0 = skip validation
	DetailedListNo     string `json:"DetailedListNo,omitempty"`
	DetailedListDate   string `json:"DetailedListDate,omitempty"`
	QrCode             string `json:"qrCode,omitempty"`
	OtherTax           string `json:"otherTax,omitempty"`
}

// BuyerInfo contains buyer/customer data (§6.4).
type BuyerInfo struct {
	BuyerName          string `json:"buyerName,omitempty"`          // max 100
	BuyerCode          string `json:"buyerCode,omitempty"`          // max 400
	BuyerLegalName     string `json:"buyerLegalName,omitempty"`     // max 400; required if buyerTaxCode is set
	BuyerTaxCode       string `json:"buyerTaxCode,omitempty"`       // max 20
	BuyerBudgetCode    string `json:"buyerBudgetCode,omitempty"`    // max 7
	BuyerAddressLine   string `json:"buyerAddressLine,omitempty"`   // max 1200; required when buyerNotGetInvoice=0
	BuyerPhoneNumber   string `json:"buyerPhoneNumber,omitempty"`   // max 15
	BuyerFaxNumber     string `json:"buyerFaxNumber,omitempty"`
	BuyerEmail         string `json:"buyerEmail,omitempty"`         // max 2000; semicolon-separated for multiple
	BuyerBankName      string `json:"buyerBankName,omitempty"`      // max 200
	BuyerBankAccount   string `json:"buyerBankAccount,omitempty"`   // max 30
	BuyerDistrictName  string `json:"buyerDistrictName,omitempty"`
	BuyerCityName      string `json:"buyerCityName,omitempty"`
	BuyerCountryCode   string `json:"buyerCountryCode,omitempty"`
	BuyerIdType        string `json:"buyerIdType,omitempty"`        // "1"=CCCD, "3"=Passport
	BuyerIdNo          string `json:"buyerIdNo,omitempty"`          // max 200; required when buyerIdType is set
	BuyerBirthDay      string `json:"buyerBirthDay,omitempty"`
	BuyerNotGetInvoice *int   `json:"buyerNotGetInvoice,omitempty"` // 0=gets invoice, 1=does not; default 0
}

// SellerInfo contains seller/vendor data (§6.3).
// Optional — if sellerTaxCode is omitted, Viettel uses the portal config.
type SellerInfo struct {
	SellerLegalName   string `json:"sellerLegalName,omitempty"`   // required when sellerTaxCode is set; max 400
	SellerTaxCode     string `json:"sellerTaxCode,omitempty"`     // required; max 20
	SellerAddressLine string `json:"sellerAddressLine,omitempty"` // required when sellerTaxCode is set; max 255
	SellerPhoneNumber string `json:"sellerPhoneNumber,omitempty"` // max 50
	SellerFaxNumber   string `json:"sellerFaxNumber,omitempty"`   // max 50
	SellerEmail       string `json:"sellerEmail,omitempty"`       // max 50
	SellerBankName    string `json:"sellerBankName,omitempty"`    // max 400
	SellerBankAccount string `json:"sellerBankAccount,omitempty"` // max 30
	SellerDistrictName string `json:"sellerDistrictName,omitempty"` // max 50
	SellerCityName     string `json:"sellerCityName,omitempty"`     // max 600
	SellerCountryCode  string `json:"sellerCountryCode,omitempty"`  // max 15
	SellerWebsite      string `json:"sellerWebsite,omitempty"`      // max 200
	StoreCode          string `json:"storeCode,omitempty"`          // max 50
	StoreName          string `json:"storeName,omitempty"`          // max 400
	MerchantCode       string `json:"merchantCode,omitempty"`       // max 4; required for qrcode78
	MerchantName       string `json:"merchantName,omitempty"`       // max 25; required for qrcode78
	MerchantCity       string `json:"merchantCity,omitempty"`       // max 15; required for qrcode78
}

// Payment describes a payment method entry (§6.5).
type Payment struct {
	PaymentMethod     string `json:"paymentMethod,omitempty"`     // 1-8
	PaymentMethodName string `json:"paymentMethodName"` // required; TM, CK, TM/CK, DTCN, KHAC, etc.
}

// ItemInfo represents a line item on the invoice (§6.6).
type ItemInfo struct {
	LineNumber                   *int     `json:"lineNumber,omitempty"`
	Selection                    *int     `json:"selection,omitempty"`    // 1=goods,2=note,3=discount,4=table/fee,5=promo(TT78),6=special(ND70)
	ItemType                     *int     `json:"itemType,omitempty"`    // required when selection=6; 1-6 per ND70
	ItemCode                     string   `json:"itemCode,omitempty"`    // max 50
	ItemName                     string   `json:"itemName,omitempty"`    // max 500
	UnitCode                     string   `json:"unitCode,omitempty"`    // max 100
	UnitName                     string   `json:"unitName,omitempty"`    // max 300
	Quantity                     *float64 `json:"quantity,omitempty"`
	UnitPrice                    *float64 `json:"unitPrice,omitempty"`
	UnitPriceWithTax             *float64 `json:"unitPriceWithTax,omitempty"` // unit price including tax (draft fuel invoices)
	ItemTotalAmountWithoutTax    *float64 `json:"itemTotalAmountWithoutTax"` // required; max 13 digits; = quantity * unitPrice
	ItemTotalAmountAfterDiscount *float64 `json:"itemTotalAmountAfterDiscount,omitempty"`
	ItemTotalAmountWithTax       *float64 `json:"itemTotalAmountWithTax,omitempty"`
	TaxPercentage                *float64 `json:"taxPercentage,omitempty"` // -2=no tax, -1=not declared, 0,5,8,10
	TaxAmount                    *float64 `json:"taxAmount,omitempty"`
	Discount                     *float64 `json:"discount,omitempty"`     // % discount on unit price
	Discount2                    *float64 `json:"discount2,omitempty"`    // second % discount on unit price
	ItemDiscount                 *float64 `json:"itemDiscount,omitempty"` // discount amount (system-calculated)
	ItemNote                     string   `json:"itemNote,omitempty"`     // max 300
	IsIncreaseItem               *bool    `json:"isIncreaseItem,omitempty"` // nil=normal, false=decrease, true=increase
	BatchNo                      string   `json:"batchNo,omitempty"`      // max 300
	ExpDate                      string   `json:"expDate,omitempty"`      // max 50
	AdjustRatio                  string   `json:"adjustRatio,omitempty"`  // "1","2","3","5"; used with adjustAmount20="0"
	SpecialInfo                  []SpecialInfoItem `json:"specialInfo,omitempty"` // required when selection=6 (ND70)
}

// SpecialInfoItem represents a special goods attribute per ND70.
type SpecialInfoItem struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
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
type GetInvoiceFileResponse struct {
	ErrorCode   *string `json:"errorCode"`
	Description *string `json:"description"`
	FileName    string  `json:"fileName,omitempty"`
	FileToBytes []byte  `json:"fileToBytes,omitempty"`
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
