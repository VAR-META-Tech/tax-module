package integration

// --- Auth ---

// MISAAuthRequest is sent to POST /auth/token.
type MISAAuthRequest struct {
	AppID    string `json:"appid"`
	TaxCode  string `json:"taxcode"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// MISAAuthResponse is the response from POST /auth/token.
// The token string is in the Data field, not a nested object.
type MISAAuthResponse struct {
	Success   bool   `json:"Success"`
	Data      string `json:"Data"`
	ErrorCode string `json:"ErrorCode"`
	Errors    string `json:"Errors"`
}

// --- Templates ---

// MISATemplate represents one invoice template from GET /invoice/templates.
// Section 13.1 of the MISA Open API integration document.
type MISATemplate struct {
	IPTemplateID             string `json:"IPTemplateID"`
	CompanyID                int    `json:"CompanyID"`
	TemplateName             string `json:"TemplateName"`
	InvTemplateNo            string `json:"InvTemplateNo"`
	InvSeries                string `json:"InvSeries"`
	OrgInvSeries             string `json:"OrgInvSeries"`
	CreatedDate              string `json:"CreatedDate"`
	ModifiedDate             string `json:"ModifiedDate"`
	Inactive                 bool   `json:"Inactive"`
	IsInheritFromOldTemplate bool   `json:"IsInheritFromOldTemplate"`
	IsSendSummary            bool   `json:"IsSendSummary"`
	IsTemplatePetrol         bool   `json:"IsTemplatePetrol"`
	IsMoreVATRate            bool   `json:"IsMoreVATRate"`
}

// MISATemplateListResponse wraps the template list response.
type MISATemplateListResponse struct {
	Success   bool           `json:"Success"`
	Data      []MISATemplate `json:"Data"`
	ErrorCode string         `json:"ErrorCode"`
}

// --- Create / Publish Invoice ---

// MISAPublishRequest is sent to POST /invoice.
// PublishInvoiceData is null for SignType 2 (HSM with CKS) and 5 (HSM without CKS).
// For SignType 1 (USB/file), InvoiceData is null and PublishInvoiceData contains signed XML.
type MISAPublishRequest struct {
	SignType           int                     `json:"SignType"`
	InvoiceData        []MISAInvoiceData       `json:"InvoiceData"`
	PublishInvoiceData []MISAPublishInvoiceData `json:"PublishInvoiceData"`
}

// MISAInvoiceData is a single invoice in the publish request.
// Section 13.3 of the MISA Open API integration document.
type MISAInvoiceData struct {
	// Required identification fields
	RefID            string `json:"RefID"`          // our UUID, stored as transaction_uuid
	InvSeries        string `json:"InvSeries"`      // from template
	InvDate          string `json:"InvDate"`        // "yyyy-MM-dd"
	IsInvoiceSummary bool   `json:"IsInvoiceSummary"` // from template.IsSendSummary

	// Optional header fields
	CurrencyCode                string  `json:"CurrencyCode,omitempty"`
	ExchangeRate                float64 `json:"ExchangeRate,omitempty"`
	PaymentMethodName           string  `json:"PaymentMethodName,omitempty"`
	IsInvoiceCalculatingMachine bool    `json:"IsInvoiceCalculatingMachine,omitempty"` // true = hóa đơn từ máy tính tiền
	IsSendEmail                 bool    `json:"IsSendEmail,omitempty"`
	ReceiverName                string  `json:"ReceiverName,omitempty"`
	ReceiverEmail               string  `json:"ReceiverEmail,omitempty"` // multiple emails separated by ";"
	SellerShopCode              string  `json:"SellerShopCode,omitempty"`
	SellerShopName              string  `json:"SellerShopName,omitempty"`

	// Buyer — BuyerLegalName and BuyerAddress are required by MISA
	BuyerCode        string `json:"BuyerCode,omitempty"`
	BuyerLegalName   string `json:"BuyerLegalName"`
	BuyerTaxCode     string `json:"BuyerTaxCode,omitempty"`
	BuyerAddress     string `json:"BuyerAddress"`
	BuyerFullName    string `json:"BuyerFullName,omitempty"`
	BuyerPhoneNumber string `json:"BuyerPhoneNumber,omitempty"`
	BuyerEmail       string `json:"BuyerEmail,omitempty"` // only 1 email allowed
	BuyerBankAccount string `json:"BuyerBankAccount,omitempty"`
	BuyerBankName    string `json:"BuyerBankName,omitempty"`
	BuyerIDNumber    string `json:"BuyerIDNumber,omitempty"`   // số định danh cá nhân (12 ký tự số)
	BuyerPassport    string `json:"BuyerPassport,omitempty"`   // số hộ chiếu (tối đa 20 ký tự)
	BuyerBudgetCode  string `json:"BuyerBudgetCode,omitempty"` // mã số ĐVCQHVNS (7 ký tự)

	// Totals (all required by MISA)
	TotalSaleAmountOC       float64 `json:"TotalSaleAmountOC"`
	TotalSaleAmount         float64 `json:"TotalSaleAmount"`
	TotalDiscountAmountOC   float64 `json:"TotalDiscountAmountOC"`
	TotalDiscountAmount     float64 `json:"TotalDiscountAmount"`
	TotalAmountWithoutVATOC float64 `json:"TotalAmountWithoutVATOC"`
	TotalAmountWithoutVAT   float64 `json:"TotalAmountWithoutVAT"`
	TotalVATAmountOC        float64 `json:"TotalVATAmountOC"`
	TotalVATAmount          float64 `json:"TotalVATAmount"`
	TotalAmountOC           float64 `json:"TotalAmountOC"`
	TotalAmount             float64 `json:"TotalAmount"`
	// TotalAmountInWords is required — Vietnamese words for the total amount.
	// Example: "Năm triệu năm trăm nghìn đồng."
	TotalAmountInWords              string `json:"TotalAmountInWords"`
	TotalAmountInWordsByENG         string `json:"TotalAmountInWordsByENG,omitempty"`
	TotalAmountInWordsUnsignNormalVN string `json:"TotalAmountInWordsUnsignNormalVN,omitempty"`

	// Custom / extension fields
	CustomField1  string `json:"CustomField1,omitempty"`
	CustomField2  string `json:"CustomField2,omitempty"`
	CustomField3  string `json:"CustomField3,omitempty"`
	CustomField4  string `json:"CustomField4,omitempty"`
	CustomField5  string `json:"CustomField5,omitempty"`
	CustomField6  string `json:"CustomField6,omitempty"`
	CustomField7  string `json:"CustomField7,omitempty"`
	CustomField8  string `json:"CustomField8,omitempty"`
	CustomField9  string `json:"CustomField9,omitempty"`
	CustomField10 string `json:"CustomField10,omitempty"`

	// Adjustment / replacement invoice fields (only used when issuing adjusted/replaced invoices)
	ReferenceType    int    `json:"ReferenceType,omitempty"`    // 1=replacement, 2=adjustment
	OrgInvoiceType   int    `json:"OrgInvoiceType,omitempty"`   // 1=NĐ123, 3=NĐ51
	OrgInvTemplateNo string `json:"OrgInvTemplateNo,omitempty"` // first character of original InvSeries
	OrgInvSeries     string `json:"OrgInvSeries,omitempty"`     // last 6 characters of original InvSeries
	OrgInvNo         string `json:"OrgInvNo,omitempty"`         // original invoice number
	OrgInvDate       string `json:"OrgInvDate,omitempty"`       // original invoice date
	InvoiceNote      string `json:"InvoiceNote,omitempty"`      // reason for adjustment/replacement

	// Line items and tax summary
	OriginalInvoiceDetail []MISAInvoiceDetail  `json:"OriginalInvoiceDetail"`
	TaxRateInfo           []MISATaxRateInfo    `json:"TaxRateInfo"`
	OptionUserDefined     *MISAOptionUserDefined `json:"OptionUserDefined,omitempty"`
	OtherInfo             []MISAOtherInfo      `json:"OtherInfo,omitempty"`
	FeeInfo               []MISAFeeInfo        `json:"FeeInfo,omitempty"`
}

// MISAInvoiceDetail is a line item within MISAInvoiceData.
// Section 13.4 of the MISA Open API integration document.
type MISAInvoiceDetail struct {
	// Required
	ItemType   int     `json:"ItemType"`  // 1=normal, 2=promotion, 3=commercial discount, 4=note, 5=transport specialty
	SortOrder  *int    `json:"SortOrder"` // null for ItemType 3 and 4; starts from 1
	LineNumber int     `json:"LineNumber"`
	ItemName   string  `json:"ItemName"`
	Quantity   float64 `json:"Quantity"`
	UnitPrice  float64 `json:"UnitPrice"`
	AmountOC   float64 `json:"AmountOC"` // = Quantity * UnitPrice

	Amount             float64 `json:"Amount"`
	DiscountRate       float64 `json:"DiscountRate"`
	DiscountAmountOC   float64 `json:"DiscountAmountOC"`   // = AmountOC * DiscountRate / 100
	DiscountAmount     float64 `json:"DiscountAmount"`
	AmountWithoutVATOC float64 `json:"AmountWithoutVATOC"` // = AmountOC - DiscountAmountOC
	AmountWithoutVAT   float64 `json:"AmountWithoutVAT"`
	// VATRateName uses string format: "10%", "5%", "8%", "0%", "KCT", "KKKNT", "KHAC:x%"
	VATRateName string  `json:"VATRateName"`
	VATAmountOC float64 `json:"VATAmountOC"`
	VATAmount   float64 `json:"VATAmount"`

	// Optional standard fields
	ItemCode   string `json:"ItemCode,omitempty"`
	UnitName   string `json:"UnitName,omitempty"`
	ExpiryDate string `json:"ExpiryDate,omitempty"`   // hạn sử dụng
	ChassisNumber string `json:"ChassisNumber,omitempty"` // số khung
	EngineNumber  string `json:"EngineNumber,omitempty"`  // số máy
	LotNo         string `json:"LotNo,omitempty"`         // số lô

	// Wage fields (optional)
	WageAmount            float64 `json:"WageAmount,omitempty"`
	WageAmountOC          float64 `json:"WageAmountOC,omitempty"`
	WagePriceAmount       float64 `json:"WagePriceAmount,omitempty"`
	WagePriceDiscountAmount float64 `json:"WagePriceDiscountAmount,omitempty"`
	WageDiscountAmountOC  float64 `json:"WageDiscountAmountOC,omitempty"`
	InWard                float64 `json:"InWard,omitempty"` // thực nhập (dùng với PXK)
}

// MISATaxRateInfo aggregates tax amounts per VAT rate (required by MISA).
// Section 13.5 of the MISA Open API integration document.
// Formula: Sum(AmountWithoutVATOC, itemType=1) - Sum(AmountWithoutVATOC, itemType=3)
type MISATaxRateInfo struct {
	VATRateName        string  `json:"VATRateName"`
	AmountWithoutVATOC float64 `json:"AmountWithoutVATOC"`
	VATAmountOC        float64 `json:"VATAmountOC"`
}

// MISAOtherInfo holds additional key-value information on an invoice.
// Section 13.6 of the MISA Open API integration document.
type MISAOtherInfo struct {
	FieldName  string `json:"FieldName"`
	DataType   string `json:"DataType"`
	FieldValue string `json:"FieldValue"`
}

// MISAFeeInfo represents an additional fee on an invoice.
// Section 13.7 of the MISA Open API integration document.
type MISAFeeInfo struct {
	FeeName      string  `json:"FeeName"`
	FeeAmountOC  float64 `json:"FeeAmountOC"`
}

// MISAOptionUserDefined controls decimal display settings for amounts.
// Section 13.9 of the MISA Open API integration document.
type MISAOptionUserDefined struct {
	MainCurrency              string `json:"MainCurrency,omitempty"`              // same as CurrencyCode
	QuantityDecimalDigits     string `json:"QuantityDecimalDigits,omitempty"`
	UnitPriceOCDecimalDigits  string `json:"UnitPriceOCDecimalDigits,omitempty"`
	AmountOCDecimalDigits     string `json:"AmountOCDecimalDigits,omitempty"`
	AmountDecimalDigits       string `json:"AmountDecimalDigits,omitempty"`
	CoefficientDecimalDigits  string `json:"CoefficientDecimalDigits,omitempty"`
	ExchangRateDecimalDigits  string `json:"ExchangRateDecimalDigits,omitempty"` // note: MISA typo "ExchangRate" (no 'e')
}

// MISAPublishInvoiceData is used for SignType=1 (USB/file signing) to publish a pre-signed XML invoice.
// Section 13.10 of the MISA Open API integration document.
type MISAPublishInvoiceData struct {
	RefID                       string `json:"RefID"`
	TransactionID               string `json:"TransactionID"`
	InvSeries                   string `json:"InvSeries"`
	InvoiceData                 string `json:"InvoiceData"` // signed XML content
	IsInvoiceCalculatingMachine bool   `json:"IsInvoiceCalculatingMachine,omitempty"`
	IsSendEmail                 bool   `json:"IsSendEmail,omitempty"`
	ReceiverName                string `json:"ReceiverName,omitempty"`
	ReceiverEmail               string `json:"ReceiverEmail,omitempty"` // separated by ";"
}

// --- Publish Response ---

// MISAPublishResponse is the response from POST /invoice.
type MISAPublishResponse struct {
	Success              bool                `json:"success"`
	ErrorCode            string              `json:"errorCode"`
	DescriptionErrorCode string              `json:"descriptionErrorCode"`
	CreateInvoiceResult  []MISACreateResult  `json:"createInvoiceResult"`
	PublishInvoiceResult []MISAPublishResult `json:"publishInvoiceResult"`
}

// MISAPublishResult is one entry in the publishInvoiceResult array.
// TransactionID is MISA's tracking ID — stored as external_id.
type MISAPublishResult struct {
	RefID         string `json:"RefID"`
	TransactionID string `json:"TransactionID"` // MISA tracking ID — store as external_id
	InvSeries     string `json:"InvSeries"`
	InvNo         string `json:"InvNo"`      // invoice number assigned by MISA
	ErrorCode     string `json:"ErrorCode"`  // empty = success
	Description   string `json:"Description"`
}

// MISACreateResult is one entry in the createInvoiceResult array (used for SignType=1).
type MISACreateResult struct {
	RefID         string `json:"RefID"`
	TransactionID string `json:"TransactionID"`
	InvoiceData   string `json:"InvoiceData"` // XML content for SignType=1
	ErrorCode     string `json:"ErrorCode"`
	Description   string `json:"Description"`
}

// --- Query Status ---

// MISAStatusResponse is the response from POST /invoice/status.
type MISAStatusResponse struct {
	Success   bool                `json:"success"`
	ErrorCode string              `json:"errorCode"`
	Data      []MISAInvoiceStatus `json:"data"`
}

// MISAInvoiceStatus represents the status of one invoice in the status response.
// Section 13.11 of the MISA Open API integration document.
type MISAInvoiceStatus struct {
	RefID          string `json:"RefID"`
	TransactionID  string `json:"TransactionID"`
	InvNo          string `json:"InvNo"`
	InvSeries      string `json:"InvSeries"`
	InvTempl       string `json:"InvTempl"`       // mẫu số hóa đơn
	BuyerName      string `json:"BuyerName"`
	BuyerTaxCode   string `json:"BuyerTaxCode"`
	BuyerCode      string `json:"BuyerCode"`
	BuyerFullName  string `json:"BuyerFullName"`
	PublishStatus  int    `json:"PublishStatus"`  // 1 = published
	EInvoiceStatus int    `json:"EInvoiceStatus"` // 1=original,2=cancelled,3=replaced,5=adjusted,7=replaced by,8=adjusted by
	ReferenceType  int    `json:"ReferenceType"`  // 0=original,1=replacement,2=adjustment,5=commercial discount
	SendTaxStatus  int    `json:"SendTaxStatus"`  // 0=not sent,1=error,2=CQT accepted,3=CQT rejected
	InvoiceCode    string `json:"InvoiceCode"`    // CQT code
	SourceType     string `json:"SourceType"`     // source system
	PublishedTime  string `json:"PublishedTime"`
}

// --- Download ---

// MISADownloadResponse is the response from POST /invoice/download.
type MISADownloadResponse struct {
	Success   bool                 `json:"success"`
	ErrorCode string               `json:"errorCode"`
	Data      []MISADownloadResult `json:"data"`
}

// MISADownloadResult contains the base64-encoded file for one invoice.
type MISADownloadResult struct {
	TransactionID string `json:"TransactionID"`
	Data          string `json:"Data"` // base64 encoded PDF/XML
}

// --- Error codes ---

const (
	MISAErrInvoiceDuplicated          = "InvoiceDuplicated"
	MISAErrInvoiceNumberNotContinuous = "InvoiceNumberNotCotinuous" // note: MISA typo in "Cotinuous"
	MISAErrTokenExpired               = "TokenExpiredCode"
	MISAErrUnauthorize                = "UnAuthorize"
	MISAErrInvalidToken               = "InvalidTokenCode"
	MISAErrDuplicateInvoiceRefID      = "DuplicateInvoiceRefID"  // same RefID already published — query status to recover
	MISAErrLicenseOutOfInvoice        = "LicenseInfo_OutOfInvoice" // not enough quota
	MISAErrInvalidInvoiceDate         = "InvalidInvoiceDate"     // invoice date earlier than last issued
	MISAErrCreateInvoiceDataError     = "CreateInvoiceDataError" // unknown XML creation error
	MISAErrInvalidAppID               = "InvalidAppID"
	MISAErrInActiveAppID              = "InActiveAppID"
)

// misaNonRetryableErrors lists error codes that indicate a permanent failure.
var misaNonRetryableErrors = map[string]bool{
	"InvalidInvoiceData":      true,
	"InvoiceTemplateNotExist": true,
	"InvalidTaxCode":          true,
	"LicenseInfo_NotBuy":      true,
	"LicenseInfo_Expired":     true,
	MISAErrLicenseOutOfInvoice: true,
	MISAErrInvalidInvoiceDate:  true,
	MISAErrInvalidAppID:        true,
	MISAErrInActiveAppID:       true,
}

// misaRetryableErrors lists error codes that can be retried.
var misaRetryableErrors = map[string]bool{
	MISAErrInvoiceDuplicated:          true,
	MISAErrInvoiceNumberNotContinuous: true,
}

// MISAError wraps a MISA API error with retryability information.
type MISAError struct {
	ErrorCode   string
	Description string
	Retryable   bool
}

func (e *MISAError) Error() string {
	return "misa error " + e.ErrorCode + ": " + e.Description
}

// IsMISARetryable returns true for error codes that allow retrying the request.
func IsMISARetryable(errorCode string) bool {
	return misaRetryableErrors[errorCode]
}

// newMISAError creates a MISAError, detecting retryability from the error code.
func newMISAError(errorCode, description string) *MISAError {
	return &MISAError{
		ErrorCode:   errorCode,
		Description: description,
		Retryable:   IsMISARetryable(errorCode),
	}
}
