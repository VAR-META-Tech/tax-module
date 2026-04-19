# Multi-Provider Plan: Viettel SInvoice + MISA MeInvoice

> **Mục tiêu:** Hỗ trợ đồng thời cả 2 nhà cung cấp hóa đơn điện tử. Invoice được tạo với field `provider` xác định nhà cung cấp nào xử lý. `DispatchingPublisher` route tự động — service/worker/handler không cần biết đang dùng provider nào.
>
> **Tài liệu tham chiếu:** `docs/Misa_API_Integrate_Doc.md` (MISA Open API v2 – cập nhật 18/11/2025)

---

## Kiến trúc tổng quan

```
                   domain.InvoicePublisher (interface)
                              │ implements
                   DispatchingPublisher         ← routes by invoice.Provider
                   ┌──────────┴──────────┐
         ViettelPublisher          MISAPublisher
         (hiện tại)                (mới)
```

**Quy tắc routing:**
- `invoice.Provider = "viettel"` → ViettelPublisher
- `invoice.Provider = "misa"` → MISAPublisher
- Không set → fallback theo `DEFAULT_PROVIDER` trong config

---

## So sánh API Viettel vs MISA (sau khi đọc doc thực tế)

| Khía cạnh | Viettel | MISA |
|-----------|---------|------|
| **Base URL** | `https://api-vinvoice.viettel.vn/...` | `https://api.meinvoice.vn/api/integration` |
| **Auth endpoint** | `POST /auth/login` JSON (`username`, `password`) | `POST /auth/token` JSON (`appid`, `taxcode`, `username`, `password`) |
| **Auth header** | Cookie `access_token=...` | `Authorization: Bearer {token}` |
| **Token TTL** | ~1h | **14 ngày** (nên gọi 7 ngày/lần) |
| **Token extras** | Không có | Token nằm trong `Data` field của response |
| **Template** | Static config (`templateCode`, `invoiceSeries`) | `GET /invoice/templates?invoiceWithCode=true&ticket=false` → lấy `InvSeries`, `IsSendSummary` |
| **Create invoice** | POST với `transactionUuid`, complex JSON | POST `{SignType, InvoiceData[], PublishInvoiceData}` |
| **Invoice ref ID** | `transactionUuid` (ta sinh) | `RefID` (ta sinh, ta lưu làm `transaction_uuid`) |
| **MISA tracking ID** | Không có | `TransactionID` — MISA trả về, **cần lưu làm `external_id`** |
| **Số hóa đơn** | `invoiceNo` Viettel trả về | `InvNo` MISA gán (lấy từ `publishInvoiceResult`) |
| **Flow** | Async — poller cần poll | **Synchronous** (SignType=2/5) — kết quả trả về ngay |
| **VAT format** | `taxPercentage float64` | `VATRateName string` ("10%", "KCT", "0%"...) |
| **Buyer fields** | `buyerName`, `buyerLegalName`... | `BuyerLegalName` (bắt buộc), `BuyerAddress` (bắt buộc), `BuyerFullName`... |
| **Total amounts** | `summarizeInfo` | Flat fields: `TotalSaleAmountOC`, `TotalAmountInWords` (bắt buộc) |
| **Tax breakdown** | `taxBreakdowns[]` | `TaxRateInfo[]` (bắt buộc) |
| **Query status** | GET theo `transactionUuid` | `POST /invoice/status` với `inputType=2` (theo `RefID`) hoặc `inputType=1` (theo `TransactionID`) |
| **Download PDF** | POST `/getInvoiceRepresentationFile` | `POST /invoice/download?downloadDataType=pdf` với body `["{TransactionID}"]` |
| **Retry errors** | Theo `ViettelErrCode` | `InvoiceDuplicated`, `InvoiceNumberNotCotinuous` |

---

## Phân tích Database

### Schema hiện tại (đã tái dựng sau 11 migrations)

#### Bảng `invoices`
| Cột | Type | Ghi chú |
|-----|------|---------|
| `id` | UUID PK | |
| `external_id` | VARCHAR(255) | Viettel: `invoiceNo` / **MISA: `TransactionID`** (dùng để download/query) |
| `transaction_uuid` | VARCHAR(36) UNIQUE | Viettel & MISA: UUID ta sinh, gửi đi làm `transactionUuid`/`RefID` |
| `status` | VARCHAR(50) | draft/submitted/processing/completed/failed |
| `buyer_name` | VARCHAR(255) NOT NULL | MISA: `BuyerFullName` |
| `buyer_legal_name` | VARCHAR(400) | MISA: `BuyerLegalName` (bắt buộc) |
| `buyer_tax_code` | VARCHAR(50) | MISA: `BuyerTaxCode` |
| `buyer_address` | TEXT | MISA: `BuyerAddress` (bắt buộc) |
| `buyer_email` | VARCHAR(2000) | MISA: `BuyerEmail` (chỉ 1 email) |
| `buyer_phone` | VARCHAR(15) | MISA: `BuyerPhoneNumber` |
| `buyer_code` | VARCHAR(400) | MISA: `BuyerCode` |
| `currency` | VARCHAR(3) | MISA: `CurrencyCode` |
| `total_amount_with_tax` | NUMERIC(18,2) | MISA: `TotalAmountOC`, `TotalAmount` |
| `total_tax_amount` | NUMERIC(18,2) | MISA: `TotalVATAmountOC`, `TotalVATAmount` |
| `total_amount_without_tax` | NUMERIC(18,2) | MISA: `TotalAmountWithoutVATOC`, `TotalAmountWithoutVAT` |
| `payment_method` | VARCHAR(50) | MISA: `PaymentMethodName` |
| `notes` | TEXT | MISA: `InvoiceNote` (điều chỉnh/thay thế) |
| ... | | |

#### Bảng `invoice_items`
| Cột | Type | Tương thích MISA? |
|-----|------|------------------|
| `item_name` | VARCHAR(500) | → MISA `ItemName` ✓ |
| `item_code` | VARCHAR(50) | → MISA `ItemCode` ✓ |
| `item_type` | INT | → MISA `ItemType` (1=thường, 2=KM, 3=CK, 4=ghi chú) ✓ |
| `unit_name` | VARCHAR(300) | → MISA `UnitName` ✓ |
| `quantity`, `unit_price` | NUMERIC | ✓ |
| `tax_percentage` | NUMERIC | → **MISA `VATRateName` là string** ("10%") — cần convert khi mapping |
| `tax_amount` | NUMERIC | → MISA `VATAmountOC`, `VATAmount` ✓ |
| `item_total_amount_without_tax` | NUMERIC | → MISA `AmountWithoutVATOC`, `AmountWithoutVAT` ✓ |
| `discount` | NUMERIC | → MISA `DiscountRate` ✓ |
| `item_total_amount_with_tax` | NUMERIC | ✓ |

#### Bảng `access_tokens`
| Cột | Ghi chú |
|-----|---------|
| `provider` VARCHAR(100) UNIQUE | `"viettel_sinvoice"` và `"misa_meinvoice"` tách biệt — **đủ** |
| `access_token` TEXT | MISA token lấy từ `response.Data` — **đủ** |
| `expires_at` TIMESTAMPTZ | MISA TTL 14 ngày → **đủ** |
| `raw_response` JSONB | Lưu full response — **đủ** |

---

## Database Changes Cần Thiết

### Migration 000012 — **BẮT BUỘC**
```sql
ALTER TABLE invoices
    ADD COLUMN provider VARCHAR(20) NOT NULL DEFAULT 'viettel';

CREATE INDEX idx_invoices_provider ON invoices(provider);
```

### Không cần migration thêm vì:
- `access_tokens.access_token` lưu được MISA token (từ `response.Data`) ✓
- `invoices.external_id` lưu được MISA `TransactionID` (VARCHAR(255)) ✓
- `invoices.transaction_uuid` dùng làm MISA `RefID` ✓
- `invoice_items` fields tương thích đủ ✓

---

## Interface Changes (`domain.InvoicePublisher`)

```go
type InvoicePublisher interface {
    CreateInvoice(ctx context.Context, invoice *Invoice) (invoiceNo string, err error)

    // Đổi từ (ctx, transactionUuid string) → (ctx, invoice *Invoice)
    // Viettel: đọc *invoice.TransactionUuid
    // MISA: đọc invoice.TransactionUuid làm RefID cho inputType=2
    QueryStatus(ctx context.Context, invoice *Invoice) (status, invoiceNo string, raw []byte, err error)

    // Thêm provider param để Dispatcher route
    ReportToAuthority(ctx context.Context, provider, transactionUuid, startDate, endDate string) (int, int, error)

    // Đổi: MISA dùng TransactionID (= invoice.ExternalID) thay vì invoiceNo
    // → pass full invoice thay vì invoiceNo string
    DownloadInvoiceFile(ctx context.Context, provider string, invoice *Invoice, fileType string) (string, error)
}
```

---

## Todo List

### Phase 1 — Database & Domain Model ✅

- [x] **[DB]** Viết `000012_add_provider_to_invoices.up.sql`
- [x] **[DB]** Viết `000012_add_provider_to_invoices.down.sql`
- [x] **[Domain]** Thêm field `Provider string` vào `domain.Invoice` struct
- [x] **[Domain]** Thêm constants `ProviderViettel`, `ProviderMISA`
- [x] **[Domain]** Cập nhật `InvoicePublisher` interface (QueryStatus, ReportToAuthority, DownloadInvoiceFile)

### Phase 2 — Config ✅

- [x] **[Config]** Đổi tên type `ThirdPartyConfig` → `ViettelConfig` (giữ nguyên mapstructure keys `THIRD_PARTY_*`)
- [x] **[Config]** Thêm struct `MISAConfig`:
  ```go
  type MISAConfig struct {
      BaseURL        string        `mapstructure:"MISA_BASE_URL"`
      AppID          string        `mapstructure:"MISA_APP_ID"`          // bắt buộc — MISA cấp khi đăng ký
      TaxCode        string        `mapstructure:"MISA_TAX_CODE"`
      Username       string        `mapstructure:"MISA_USERNAME"`
      Password       string        `mapstructure:"MISA_PASSWORD"`
      InvoiceWithCode bool         `mapstructure:"MISA_INVOICE_WITH_CODE"` // true=có mã, false=không mã
      InvoiceCalcu   bool          `mapstructure:"MISA_INVOICE_CALCU"`     // true=hóa đơn từ máy tính tiền
      SignType        int          `mapstructure:"MISA_SIGN_TYPE"`         // 2=HSM có CKS, 5=HSM không CKS
      Timeout        time.Duration `mapstructure:"MISA_TIMEOUT"`
  }
  ```
- [x] **[Config]** Thêm `ProviderConfig { Default string }` với env `DEFAULT_PROVIDER` (default: `"viettel"`)
- [x] **[Config]** Cập nhật struct `Config` và hàm `Load()` với defaults:
  ```
  MISA_BASE_URL          = https://api.meinvoice.vn/api/integration
  MISA_APP_ID            = (lấy từ MISA)
  MISA_TAX_CODE          = (mã số thuế)
  MISA_INVOICE_WITH_CODE = true
  MISA_INVOICE_CALCU     = false
  MISA_SIGN_TYPE         = 2
  MISA_TIMEOUT           = 30s
  DEFAULT_PROVIDER       = viettel
  ```

### Phase 3 — Repository ✅

- [x] **[Repo]** Thêm `provider` vào SQL `INSERT` trong `CreateWithItems`
- [x] **[Repo]** Thêm `provider` vào `scanInvoice` helper
- [x] **[Repo]** Thêm `provider` vào SQL `UPDATE` trong `Update`

### Phase 4 — Viettel (cập nhật signature mới) ✅

- [x] **[Viettel]** Sửa `ViettelPublisher.QueryStatus` nhận `invoice *domain.Invoice`
- [x] **[Viettel]** Sửa `ViettelPublisher.ReportToAuthority` thêm param `provider string`
- [x] **[Viettel]** Sửa `ViettelPublisher.DownloadInvoiceFile` nhận `invoice *domain.Invoice`

### Phase 5 — MISA DTOs (`internal/integration/misa_dto.go`) ✅

> **Lưu ý:** API này (`/api/integration`) khác hoàn toàn với API cũ (`/api/v2`). Không dùng các struct đã plan trước.

- [x] **[MISA]** Tạo `internal/integration/misa_dto.go` với các structs:

```go
// --- Auth ---
type MISAAuthRequest struct {
    AppID    string `json:"appid"`
    TaxCode  string `json:"taxcode"`
    Username string `json:"username"`
    Password string `json:"password"`
}

type MISAAuthResponse struct {
    Success   bool   `json:"Success"`
    Data      string `json:"Data"`      // token string nằm ở đây
    ErrorCode string `json:"ErrorCode"`
    Errors    string `json:"Errors"`
}

// --- Template (section 13.1) ---
// GET /invoice/templates?invoiceWithCode=true&ticket=false
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

type MISATemplateListResponse struct {
    Success   bool           `json:"Success"`
    Data      []MISATemplate `json:"Data"`
    ErrorCode string         `json:"ErrorCode"`
}

// --- Create/Publish Invoice (section 13.3) ---
// POST /invoice với SignType=2 (HSM sync) hoặc SignType=5 (không CKS)
// SignType=1 (USB/file): InvoiceData=null, dùng PublishInvoiceData
type MISAPublishRequest struct {
    SignType           int                     `json:"SignType"`
    InvoiceData        []MISAInvoiceData       `json:"InvoiceData"`
    PublishInvoiceData []MISAPublishInvoiceData `json:"PublishInvoiceData"`
}

type MISAInvoiceData struct {
    // Bắt buộc
    RefID            string `json:"RefID"`            // GUID của ta
    InvSeries        string `json:"InvSeries"`        // từ template
    InvDate          string `json:"InvDate"`          // "yyyy-MM-dd"
    IsInvoiceSummary bool   `json:"IsInvoiceSummary"` // = template.IsSendSummary

    // Header optional
    CurrencyCode                string  `json:"CurrencyCode,omitempty"`
    ExchangeRate                float64 `json:"ExchangeRate,omitempty"`
    PaymentMethodName           string  `json:"PaymentMethodName,omitempty"`
    IsInvoiceCalculatingMachine bool    `json:"IsInvoiceCalculatingMachine,omitempty"` // true = hóa đơn từ máy tính tiền
    IsSendEmail                 bool    `json:"IsSendEmail,omitempty"`
    ReceiverName                string  `json:"ReceiverName,omitempty"`
    ReceiverEmail               string  `json:"ReceiverEmail,omitempty"` // nhiều email phân tách bằng ";"
    SellerShopCode              string  `json:"SellerShopCode,omitempty"`
    SellerShopName              string  `json:"SellerShopName,omitempty"`

    // Buyer (BuyerLegalName + BuyerAddress bắt buộc)
    BuyerCode        string `json:"BuyerCode,omitempty"`
    BuyerLegalName   string `json:"BuyerLegalName"`
    BuyerTaxCode     string `json:"BuyerTaxCode,omitempty"`
    BuyerAddress     string `json:"BuyerAddress"`
    BuyerFullName    string `json:"BuyerFullName,omitempty"`
    BuyerPhoneNumber string `json:"BuyerPhoneNumber,omitempty"`
    BuyerEmail       string `json:"BuyerEmail,omitempty"` // chỉ 1 email
    BuyerBankAccount string `json:"BuyerBankAccount,omitempty"`
    BuyerBankName    string `json:"BuyerBankName,omitempty"`
    BuyerIDNumber    string `json:"BuyerIDNumber,omitempty"`   // số định danh cá nhân (12 ký tự số)
    BuyerPassport    string `json:"BuyerPassport,omitempty"`   // số hộ chiếu (tối đa 20 ký tự)
    BuyerBudgetCode  string `json:"BuyerBudgetCode,omitempty"` // mã ĐVCQHVNS (7 ký tự)

    // Totals (tất cả bắt buộc)
    TotalSaleAmountOC               float64 `json:"TotalSaleAmountOC"`
    TotalSaleAmount                 float64 `json:"TotalSaleAmount"`
    TotalDiscountAmountOC           float64 `json:"TotalDiscountAmountOC"`
    TotalDiscountAmount             float64 `json:"TotalDiscountAmount"`
    TotalAmountWithoutVATOC         float64 `json:"TotalAmountWithoutVATOC"`
    TotalAmountWithoutVAT           float64 `json:"TotalAmountWithoutVAT"`
    TotalVATAmountOC                float64 `json:"TotalVATAmountOC"`
    TotalVATAmount                  float64 `json:"TotalVATAmount"`
    TotalAmountOC                   float64 `json:"TotalAmountOC"`
    TotalAmount                     float64 `json:"TotalAmount"`
    TotalAmountInWords              string  `json:"TotalAmountInWords"`               // bắt buộc — VD: "Năm triệu đồng."
    TotalAmountInWordsByENG         string  `json:"TotalAmountInWordsByENG,omitempty"`
    TotalAmountInWordsUnsignNormalVN string `json:"TotalAmountInWordsUnsignNormalVN,omitempty"`

    // Trường mở rộng (optional)
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

    // Điều chỉnh/Thay thế (chỉ dùng khi phát hành hóa đơn điều chỉnh/thay thế)
    ReferenceType    int    `json:"ReferenceType,omitempty"`    // 1=thay thế, 2=điều chỉnh
    OrgInvoiceType   int    `json:"OrgInvoiceType,omitempty"`   // 1=NĐ123, 3=NĐ51
    OrgInvTemplateNo string `json:"OrgInvTemplateNo,omitempty"` // ký tự đầu của InvSeries gốc
    OrgInvSeries     string `json:"OrgInvSeries,omitempty"`     // 6 ký tự cuối của InvSeries gốc
    OrgInvNo         string `json:"OrgInvNo,omitempty"`         // số hóa đơn gốc
    OrgInvDate       string `json:"OrgInvDate,omitempty"`       // ngày hóa đơn gốc
    InvoiceNote      string `json:"InvoiceNote,omitempty"`      // lý do điều chỉnh/thay thế

    // Items & Tax
    OriginalInvoiceDetail []MISAInvoiceDetail    `json:"OriginalInvoiceDetail"`
    TaxRateInfo           []MISATaxRateInfo      `json:"TaxRateInfo"`
    OptionUserDefined     *MISAOptionUserDefined `json:"OptionUserDefined,omitempty"`
    OtherInfo             []MISAOtherInfo        `json:"OtherInfo,omitempty"`
    FeeInfo               []MISAFeeInfo          `json:"FeeInfo,omitempty"`
}

// OriginalInvoiceDetail (section 13.4)
type MISAInvoiceDetail struct {
    // Bắt buộc
    ItemType   int     `json:"ItemType"`  // 1=thường, 2=KM, 3=CK thương mại, 4=ghi chú, 5=vận tải
    SortOrder  *int    `json:"SortOrder"` // null với ItemType 3,4
    LineNumber int     `json:"LineNumber"`
    ItemName   string  `json:"ItemName"`
    Quantity   float64 `json:"Quantity"`
    UnitPrice  float64 `json:"UnitPrice"`
    AmountOC   float64 `json:"AmountOC"` // = Quantity * UnitPrice
    Amount     float64 `json:"Amount"`
    DiscountRate       float64 `json:"DiscountRate"`
    DiscountAmountOC   float64 `json:"DiscountAmountOC"`   // = AmountOC * DiscountRate / 100
    DiscountAmount     float64 `json:"DiscountAmount"`
    AmountWithoutVATOC float64 `json:"AmountWithoutVATOC"` // = AmountOC - DiscountAmountOC
    AmountWithoutVAT   float64 `json:"AmountWithoutVAT"`
    VATRateName        string  `json:"VATRateName"` // "10%","5%","8%","0%","KCT","KKKNT","KHAC:x%"
    VATAmountOC        float64 `json:"VATAmountOC"`
    VATAmount          float64 `json:"VATAmount"`

    // Optional
    ItemCode      string `json:"ItemCode,omitempty"`
    UnitName      string `json:"UnitName,omitempty"`
    ExpiryDate    string `json:"ExpiryDate,omitempty"`    // hạn sử dụng
    ChassisNumber string `json:"ChassisNumber,omitempty"` // số khung
    EngineNumber  string `json:"EngineNumber,omitempty"`  // số máy
    LotNo         string `json:"LotNo,omitempty"`         // số lô

    // Tiền công (optional)
    WageAmount              float64 `json:"WageAmount,omitempty"`
    WageAmountOC            float64 `json:"WageAmountOC,omitempty"`
    WagePriceAmount         float64 `json:"WagePriceAmount,omitempty"`
    WagePriceDiscountAmount float64 `json:"WagePriceDiscountAmount,omitempty"`
    WageDiscountAmountOC    float64 `json:"WageDiscountAmountOC,omitempty"`
    InWard                  float64 `json:"InWard,omitempty"` // thực nhập (dùng với PXK)
}

// TaxRateInfo — tổng hợp thuế theo từng loại thuế suất (section 13.5, bắt buộc)
// Formula: Sum(AmountWithoutVATOC, itemType=1) - Sum(AmountWithoutVATOC, itemType=3)
type MISATaxRateInfo struct {
    VATRateName        string  `json:"VATRateName"`
    AmountWithoutVATOC float64 `json:"AmountWithoutVATOC"`
    VATAmountOC        float64 `json:"VATAmountOC"`
}

// OtherInfo — thông tin khác (section 13.6)
type MISAOtherInfo struct {
    FieldName  string `json:"FieldName"`
    DataType   string `json:"DataType"`
    FieldValue string `json:"FieldValue"`
}

// FeeInfo — phí khác (section 13.7)
type MISAFeeInfo struct {
    FeeName     string  `json:"FeeName"`
    FeeAmountOC float64 `json:"FeeAmountOC"`
}

// OptionUserDefined — thiết lập số thập phân (section 13.9)
type MISAOptionUserDefined struct {
    MainCurrency             string `json:"MainCurrency,omitempty"`
    QuantityDecimalDigits    string `json:"QuantityDecimalDigits,omitempty"`
    UnitPriceOCDecimalDigits string `json:"UnitPriceOCDecimalDigits,omitempty"`
    AmountOCDecimalDigits    string `json:"AmountOCDecimalDigits,omitempty"`
    AmountDecimalDigits      string `json:"AmountDecimalDigits,omitempty"`
    CoefficientDecimalDigits string `json:"CoefficientDecimalDigits,omitempty"`
    ExchangRateDecimalDigits string `json:"ExchangRateDecimalDigits,omitempty"` // typo MISA: "ExchangRate"
}

// PublishInvoiceData — dùng cho SignType=1 (USB/file) để phát hành XML đã ký (section 13.10)
type MISAPublishInvoiceData struct {
    RefID                       string `json:"RefID"`
    TransactionID               string `json:"TransactionID"`
    InvSeries                   string `json:"InvSeries"`
    InvoiceData                 string `json:"InvoiceData"` // XML đã ký số
    IsInvoiceCalculatingMachine bool   `json:"IsInvoiceCalculatingMachine,omitempty"`
    IsSendEmail                 bool   `json:"IsSendEmail,omitempty"`
    ReceiverName                string `json:"ReceiverName,omitempty"`
    ReceiverEmail               string `json:"ReceiverEmail,omitempty"` // phân tách bằng ";"
}

// --- Publish Response ---
type MISAPublishResponse struct {
    Success              bool                `json:"success"`
    ErrorCode            string              `json:"errorCode"`
    DescriptionErrorCode string              `json:"descriptionErrorCode"`
    CreateInvoiceResult  []MISACreateResult  `json:"createInvoiceResult"`
    PublishInvoiceResult []MISAPublishResult `json:"publishInvoiceResult"`
}

type MISAPublishResult struct {
    RefID         string `json:"RefID"`
    TransactionID string `json:"TransactionID"` // MISA tracking ID — lưu làm external_id
    InvSeries     string `json:"InvSeries"`
    InvNo         string `json:"InvNo"`     // số hóa đơn MISA gán
    ErrorCode     string `json:"ErrorCode"` // rỗng = thành công
    Description   string `json:"Description"`
}

type MISACreateResult struct {
    RefID         string `json:"RefID"`
    TransactionID string `json:"TransactionID"`
    InvoiceData   string `json:"InvoiceData"` // XML content (SignType=1)
    ErrorCode     string `json:"ErrorCode"`
    Description   string `json:"Description"`
}

// --- Query Status ---
// POST /invoice/status?invoiceWithCode=true&invoiceCalcu=false&inputType=2
// Body: ["refID1", ...]
type MISAStatusResponse struct {
    Success   bool                `json:"success"`
    ErrorCode string              `json:"errorCode"`
    Data      []MISAInvoiceStatus `json:"data"`
}

// InvoiceStatus (section 13.11)
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
    PublishStatus  int    `json:"PublishStatus"`  // 1 = đã phát hành
    EInvoiceStatus int    `json:"EInvoiceStatus"` // 1=gốc,2=bị hủy,3=thay thế,5=điều chỉnh,7=bị thay thế,8=bị điều chỉnh
    ReferenceType  int    `json:"ReferenceType"`  // 0=gốc,1=thay thế,2=điều chỉnh,5=CKTM
    SendTaxStatus  int    `json:"SendTaxStatus"`  // 0=chưa,1=lỗi,2=CQT chấp nhận,3=CQT từ chối
    InvoiceCode    string `json:"InvoiceCode"`    // mã CQT
    SourceType     string `json:"SourceType"`     // nguồn dữ liệu
    PublishedTime  string `json:"PublishedTime"`
}

// --- Download ---
// POST /invoice/download?invoiceWithCode=true&invoiceCalcu=false&downloadDataType=pdf
// Body: ["TransactionID1", ...]
type MISADownloadResponse struct {
    Success   bool                 `json:"success"`
    ErrorCode string               `json:"errorCode"`
    Data      []MISADownloadResult `json:"data"`
}

type MISADownloadResult struct {
    TransactionID string `json:"TransactionID"`
    Data          string `json:"Data"` // base64 encoded
}

// --- Error Codes ---
const (
    MISAErrInvoiceDuplicated          = "InvoiceDuplicated"           // retry
    MISAErrInvoiceNumberNotContinuous = "InvoiceNumberNotCotinuous"   // retry (typo của MISA)
    MISAErrTokenExpired               = "TokenExpiredCode"
    MISAErrUnauthorize                = "UnAuthorize"
    MISAErrInvalidToken               = "InvalidTokenCode"
    MISAErrDuplicateInvoiceRefID      = "DuplicateInvoiceRefID"       // gọi QueryStatus để recover
    MISAErrLicenseOutOfInvoice        = "LicenseInfo_OutOfInvoice"    // non-retryable
    MISAErrInvalidInvoiceDate         = "InvalidInvoiceDate"          // non-retryable
    MISAErrCreateInvoiceDataError     = "CreateInvoiceDataError"
    MISAErrInvalidAppID               = "InvalidAppID"                // non-retryable
    MISAErrInActiveAppID              = "InActiveAppID"               // non-retryable
)

var misaNonRetryableErrors = map[string]bool{
    "InvalidInvoiceData":         true,
    "InvoiceTemplateNotExist":    true,
    "InvalidTaxCode":             true,
    "LicenseInfo_NotBuy":         true,
    "LicenseInfo_Expired":        true,
    MISAErrLicenseOutOfInvoice:   true,
    MISAErrInvalidInvoiceDate:    true,
    MISAErrInvalidAppID:          true,
    MISAErrInActiveAppID:         true,
}

var misaRetryableErrors = map[string]bool{
    MISAErrInvoiceDuplicated:          true,
    MISAErrInvoiceNumberNotContinuous: true,
}
```

### Phase 6 — MISA Client (`internal/integration/misa_client.go`) ✅

- [x] **[MISA]** Tạo `internal/integration/misa_client.go`:
  - `MISAClient` struct (`cfg MISAConfig`, `http *http.Client`, `tokenRepo`, `log`, `mu sync.Mutex`, `template *MISATemplate`, `templateOnce sync.Once`)
  - `login(ctx)`:
    - POST JSON `{appid, taxcode, username, password}` tới `{BaseURL}/auth/token`
    - Header: `Content-Type: application/json` (KHÔNG dùng `taxcode` header)
    - Parse token từ `response.Data` (string), không phải `access_token` field
    - Store token với TTL 14 ngày
  - `getToken(ctx)`: check DB cache với buffer **1 ngày** (token TTL 14 ngày, MISA khuyến nghị 7 ngày/lần)
  - `doAuthenticatedRequest(ctx)`: header `Authorization: Bearer {token}` (không có TaxCode header)
  - `FetchTemplate(ctx)`:
    - `GET {BaseURL}/invoice/templates?invoiceWithCode={cfg.InvoiceWithCode}&ticket=false`
    - Cache trong `templateOnce` — chỉ reload khi ký hiệu thay đổi
    - Lọc template `!Inactive`, lấy template đầu tiên phù hợp
  - `PublishInvoice(ctx, req *MISAPublishRequest)`: `POST {BaseURL}/invoice`
  - `GetInvoiceStatus(ctx, refIDs []string)`: `POST {BaseURL}/invoice/status?inputType=2&invoiceWithCode={}&invoiceCalcu={}`
  - `DownloadInvoice(ctx, transactionIDs []string, fileType string)`: `POST {BaseURL}/invoice/download?downloadDataType={fileType}&invoiceWithCode={}&invoiceCalcu={}`

### Phase 7 — MISA Mapper (`internal/integration/misa_mapper.go`) ✅

- [x] **[MISA]** Tạo `internal/integration/misa_mapper.go`:

```go
func MapInvoiceToMISA(
    invoice *domain.Invoice,
    template *MISATemplate,
) *MISAInvoiceData
```

**Field mapping:**

| Domain | MISA | Ghi chú |
|--------|------|---------|
| `invoice.TransactionUuid` | `RefID` | UUID ta sinh |
| `template.InvSeries` | `InvSeries` | từ template |
| `template.IsSendSummary` | `IsInvoiceSummary` | bắt buộc |
| `invoice.IssuedAt` (hoặc now) | `InvDate` | format "yyyy-MM-dd" |
| `invoice.BuyerLegalName` | `BuyerLegalName` | bắt buộc |
| `invoice.BuyerAddress` | `BuyerAddress` | bắt buộc |
| `invoice.BuyerName` | `BuyerFullName` | |
| `invoice.BuyerTaxCode` | `BuyerTaxCode` | |
| `invoice.BuyerEmail` | `BuyerEmail`, `ReceiverEmail` | |
| `invoice.BuyerPhone` | `BuyerPhoneNumber` | |
| `invoice.BuyerCode` | `BuyerCode` | |
| `invoice.Currency` | `CurrencyCode` | |
| `invoice.ExchangeRate` | `ExchangeRate` | |
| `invoice.PaymentMethod` (default "TM/CK") | `PaymentMethodName` | |
| `invoice.TotalAmountWithoutTax` | `TotalSaleAmountOC`, `TotalSaleAmount`, `TotalAmountWithoutVATOC`, `TotalAmountWithoutVAT` | |
| `invoice.TotalTaxAmount` | `TotalVATAmountOC`, `TotalVATAmount` | |
| `invoice.TotalAmountWithTax` | `TotalAmountOC`, `TotalAmount` | |
| 0 | `TotalDiscountAmountOC`, `TotalDiscountAmount` | mặc định 0 |
| **Hàm convert số → chữ VND** | `TotalAmountInWords` | **BẮT BUỘC**, cần viết utility |

**Item mapping:**

| Domain `InvoiceItem` | MISA `OriginalInvoiceDetail` |
|---------------------|------------------------------|
| `item.ItemType` (default 1) | `ItemType` |
| `i+1` | `SortOrder`, `LineNumber` |
| `item.ItemCode` | `ItemCode` |
| `item.ItemName` | `ItemName` |
| `item.UnitName` | `UnitName` |
| `item.Quantity` | `Quantity` |
| `item.UnitPrice` | `UnitPrice` |
| `item.ItemTotalAmountWithoutTax` | `AmountOC`, `Amount` |
| `item.Discount` (%) | `DiscountRate` |
| `item.ItemTotalAmountWithoutTax * item.Discount / 100` | `DiscountAmountOC`, `DiscountAmount` |
| `item.ItemTotalAmountWithoutTax - DiscountAmount` | `AmountWithoutVATOC`, `AmountWithoutVAT` |
| `formatVATRate(item.TaxPercentage)` | `VATRateName` | "10%", "KCT", "0%"... |
| `item.TaxAmount` | `VATAmountOC`, `VATAmount` |

**Helper cần viết:**
```go
// formatVATRate chuyển float64 → VATRateName string cho MISA
// 10.0 → "10%", 0.0 → "0%", -1.0 → "KCT" (không chịu thuế), -2.0 → "KKKNT"
func formatVATRate(taxPercentage float64) string

// amountToWordsVND chuyển số tiền → chuỗi chữ tiếng Việt
// 5500000.0 → "Năm triệu năm trăm nghìn đồng."
func amountToWordsVND(amount float64) string
```

**TaxRateInfo (aggregate):**
```go
func buildTaxRateInfo(items []*domain.InvoiceItem) []MISATaxRateInfo
// Group by VATRateName, sum AmountWithoutVATOC và VATAmountOC
```

### Phase 8 — MISA Publisher (`internal/integration/misa_publisher.go`) ✅

- [x] **[MISA]** Tạo `internal/integration/misa_publisher.go` implements `domain.InvoicePublisher`:

```go
// CreateInvoice — SYNCHRONOUS (SignType=2/5)
func (p *MISAPublisher) CreateInvoice(ctx, invoice) (string, error):
    1. FetchTemplate() — lấy từ cache sync.Once
    2. MapInvoiceToMISA(invoice, template)
    3. POST /invoice với SignType=cfg.SignType
    4. Kiểm tra response.Success và publishInvoiceResult[0].ErrorCode
    5. Nếu ErrorCode rỗng → thành công ngay:
        - external_id = publishInvoiceResult[0].TransactionID
        - invoiceNo = publishInvoiceResult[0].InvNo
        - Return TransactionID (ta sẽ store làm external_id)
    6. Nếu ErrorCode là retryable (InvoiceDuplicated, InvoiceNumberNotCotinuous) → return retryable error
    7. Nếu ErrorCode là non-retryable → return non-retryable error
    // Lưu ý: MISA trả về kết quả ngay → không cần async poll như Viettel

// QueryStatus — dùng RefID của ta (inputType=2)
func (p *MISAPublisher) QueryStatus(ctx, invoice) (status, invoiceNo string, raw []byte, err error):
    1. POST /invoice/status?inputType=2&invoiceWithCode={}&invoiceCalcu={}
       Body: [invoice.TransactionUuid]
    2. Parse MISAInvoiceStatus từ Data[0]
    3. Nếu PublishStatus == 1 (đã phát hành):
        - Return "completed", status.InvNo
    4. Nếu không:
        - Return "processing", ""

// ReportToAuthority — stub (MISA không có endpoint tương đương trong API tích hợp)
func (p *MISAPublisher) ReportToAuthority(ctx, provider, ...) (int, int, error):
    log.Warn("ReportToAuthority not supported for MISA — handled automatically by MISA platform")
    return 0, 0, nil // không lỗi, MISA tự xử lý CQT

// DownloadInvoiceFile — dùng TransactionID (= invoice.ExternalID)
func (p *MISAPublisher) DownloadInvoiceFile(ctx, provider string, invoice *domain.Invoice, fileType string) (string, error):
    1. Cần invoice.ExternalID (= MISA TransactionID)
    2. POST /invoice/download?downloadDataType={fileType}&invoiceWithCode={}&invoiceCalcu={}
       Body: [*invoice.ExternalID]
    3. Return base64 từ Data[0].Data
```

### Phase 9 — MISA Error Type ✅

- [x] **[MISA]** Tạo `MISAError` struct (trong `misa_dto.go`) với `IsMISARetryable`:
  ```go
  type MISAError struct {
      ErrorCode   string
      Description string
      Retryable   bool
  }
  func (e *MISAError) Error() string
  func IsMISARetryable(errorCode string) bool
  ```

### Phase 10 — DispatchingPublisher (`internal/integration/dispatching_publisher.go`) ✅

- [x] **[Dispatch]** Tạo `internal/integration/dispatching_publisher.go`:
  ```go
  type DispatchingPublisher struct {
      providers       map[string]domain.InvoicePublisher
      defaultProvider string
  }

  func NewDispatchingPublisher(defaultProvider string, viettel, misa domain.InvoicePublisher) *DispatchingPublisher

  func (d *DispatchingPublisher) resolve(provider string) domain.InvoicePublisher

  // Implement tất cả methods của InvoicePublisher
  // CreateInvoice: route by invoice.Provider
  // QueryStatus:   route by invoice.Provider
  // ReportToAuthority: route by provider param
  // DownloadInvoiceFile: route by provider param
  ```

### Phase 11 — Worker Pool (`internal/worker/pool.go`) ✅

- [x] **[Worker]** Sửa `handlePoll` dùng `invoice` thay vì `*invoice.TransactionUuid`
- [ ] **[Worker]** Cập nhật log messages bỏ hardcode "viettel" → dùng `invoice.Provider`
- [ ] **[Worker]** Sửa `handlePublishError`: thêm case cho `MISAError` retryable

### Phase 12 — Handler & DTO

- [x] **[DTO]** Thêm `Provider string` vào `CreateInvoiceRequest`
- [x] **[Handler]** `CreateInvoice`: set `invoice.Provider` từ request, fallback `defaultProvider`
- [x] **[Handler]** `ReportToAuthority`: service lookup invoice → lấy `invoice.Provider`
- [x] **[Handler]** `DownloadInvoiceFile`: pass `invoice.Provider` và full `invoice`

### Phase 13 — Wiring (`cmd/server/main.go`) ✅

- [x] **[Main]** Cập nhật wiring:
  ```go
  viettelClient := integration.NewViettelClient(cfg.Viettel, tokenRepo, log)
  viettelPub    := integration.NewViettelPublisher(viettelClient, cfg.Viettel, cfg.Seller, log)

  misaClient := integration.NewMISAClient(cfg.MISA, tokenRepo, log)
  misaPub    := integration.NewMISAPublisher(misaClient, cfg.MISA, log)

  publisher := integration.NewDispatchingPublisher(cfg.Provider.Default, viettelPub, misaPub)
  ```

### Phase 14 — Testing ✅

- [x] **[Test]** Unit test `formatVATRate`: 10.0→"10%", 0.0→"0%", -1.0→"KCT", -2.0→"KKKNT"
- [x] **[Test]** Unit test `amountToWordsVND`: số → chữ tiếng Việt đúng
- [x] **[Test]** Unit test `buildTaxRateInfo`: group đúng theo VATRateName
- [x] **[Test]** Unit test `MapInvoiceToMISA`: field mapping đầy đủ
- [x] **[Test]** Unit test `DispatchingPublisher.resolve()`: routing đúng provider + fallback + unknown
- [x] **[Test]** Unit test retry logic: `InvoiceDuplicated` → retryable, `InvalidTaxCode` → non-retryable
- [x] **[Test]** `go build ./...` pass không lỗi, toàn bộ test xanh

### Phase 15 — Documentation & Cleanup

- [ ] **[Docs]** Cập nhật `CLAUDE.md` — mô tả multi-provider architecture
- [ ] **[Docs]** Cập nhật `docs/API.md` — thêm field `provider` vào `POST /invoices`
- [ ] **[Cleanup]** Xóa hardcode string "viettel" trong log messages của worker

---

## Các vấn đề đã giải đáp từ tài liệu

| Câu hỏi | Đáp án từ doc |
|---------|--------------|
| Base URL | `https://api.meinvoice.vn/api/integration` (prod), `https://testapi.meinvoice.vn/api/integration` (test) |
| Auth | POST JSON với `appid` + `taxcode` + `username` + `password`. Token trong `response.Data` |
| Có mã / không mã | Xác định bằng ký tự thứ 2 của `InvSeries` (C=có mã, K=không mã). Config `MISA_INVOICE_WITH_CODE` |
| Template | `GET /invoice/templates?invoiceWithCode=true` — lưu `InvSeries` và `IsSendSummary` |
| EInvoiceStatus values | 1=gốc, 2=bị hủy, 3=thay thế, 5=điều chỉnh, 7=bị thay thế, 8=bị điều chỉnh |
| Query status | POST `/invoice/status?inputType=2` — dùng `RefID` ta sinh |
| Download PDF | POST `/invoice/download?downloadDataType=pdf` với body `[TransactionID]` |
| Send to CQT | **MISA tự xử lý** — không có endpoint riêng trong tích hợp |
| Retry errors | `InvoiceDuplicated`, `InvoiceNumberNotCotinuous` |

## Câu hỏi còn cần confirm

| # | Câu hỏi | Ảnh hưởng |
|---|---------|-----------|
| 1 | `MISA_APP_ID` là gì? | Bắt buộc trong auth request — cần lấy từ MISA |
| 2 | `MISA_SIGN_TYPE` dùng loại nào? 2 (có CKS) hay 5 (không CKS)? | Ảnh hưởng quy trình phát hành |
| 3 | `DEFAULT_PROVIDER` khi deploy production? | Config default |
| 4 | Format số tiền bằng chữ (`TotalAmountInWords`)? | **Đã implement** — `amountToWordsVND()` trong `misa_mapper.go` |

---

## Ghi chú kỹ thuật quan trọng

### MISA là synchronous (khác Viettel)
Với SignType=2/5, MISA trả về kết quả ngay trong response:
- `publishInvoiceResult[0].ErrorCode` rỗng → hóa đơn phát hành thành công
- Không cần poller — `CreateInvoice` sẽ return `TransactionID` ngay, worker mark `completed`
- Nếu lỗi retryable → worker retry theo cơ chế hiện có

### `external_id` cho MISA là `TransactionID`
- Viettel: `external_id` = `invoiceNo` (ví dụ: "B25T0000001")
- MISA: `external_id` = MISA's `TransactionID` (GUID, cần cho download API)
- `InvNo` (số hóa đơn dạng "0000001") lấy từ `publishInvoiceResult` hoặc status query
- Nếu muốn lưu `InvNo` riêng: cần thêm migration (optional, không bắt buộc)

### VATRateName — string format
MISA không dùng số float mà dùng string:
- `10.0` → `"10%"`
- `0.0` → `"0%"`
- `-1` (Viettel: không chịu thuế) → `"KCT"`
- `-2` (Viettel: không kê khai) → `"KKKNT"`
- Thuế suất khác → `"KHAC:x%"` (VD: `"KHAC:3.5%"`)

### DuplicateInvoiceRefID — xử lý đặc biệt
Khi gặp lỗi `DuplicateInvoiceRefID` trong `publishInvoiceResult.ErrorCode`:
- MISA trả về thông tin của hóa đơn đã phát hành trước đó
- **Không retry** — thay vào đó gọi `QueryStatus` với `RefID` để lấy `TransactionID` + `InvNo` và cập nhật DB
- Implement riêng nếu cần: hiện tại publisher xử lý như non-retryable error

### SignType — các loại ký số
| SignType | Mô tả | Flow |
|----------|-------|------|
| 1 | USB/File — ký ngoài, cần tool MISA SignService | Async: tạo XML → ký → publish |
| 2 | HSM, **có hiển thị CKS** | **Synchronous** — kết quả ngay |
| 3 | HSM, có CKS, **bất đồng bộ** | Async — cần poll |
| 4 | HSM, Vé không mã, không CKS | Synchronous |
| 5 | HSM, Hóa đơn MTT, không CKS | **Synchronous** — kết quả ngay |
| 6 | HSM, HĐ/Vé MTT, không CKS, **bất đồng bộ** | Async — cần poll |

**Khuyến nghị dùng SignType=2 hoặc 5** (synchronous, không cần poller cho MISA).

### TotalAmountInWords — bắt buộc
MISA yêu cầu trường này. Cần viết utility hoặc dùng thư viện Go chuyển số tiền VND sang chữ tiếng Việt.
Ví dụ: `5500000.0` → `"Năm triệu năm trăm nghìn đồng."`

### Template cache
- MISA khuyến nghị: "chỉ gọi lại khi ký hiệu hóa đơn thay đổi (trừ thay đổi năm)"
- Ký hiệu thay đổi năm tự động (MISA xử lý theo `InvDate`)
- → Cache template thoải mái trong memory với `sync.Once`, invalidate thủ công khi cần

### Request phải tuần tự
Tài liệu MISA: "Bắt buộc PHẢI xử lý request tuần tự với mỗi ký hiệu hóa đơn"
→ Worker pool hiện tại đã đủ — mỗi invoice xử lý độc lập, không batch

### Interface change: `DownloadInvoiceFile`
Chữ ký cũ: `DownloadInvoiceFile(ctx, invoiceNo, fileType string)`  
Chữ ký mới: `DownloadInvoiceFile(ctx, provider string, invoice *Invoice, fileType string)`  
- Viettel: dùng `*invoice.ExternalID` làm `invoiceNo`
- MISA: dùng `*invoice.ExternalID` làm `TransactionID`
- Nhất quán — cả 2 đều dùng `ExternalID`, chỉ khác endpoint
