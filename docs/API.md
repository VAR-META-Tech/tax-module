# Tax Module — API Reference

Internal service for managing e-invoices powered by [Viettel SInvoice](https://vinvoice.viettel.vn/).
All APIs are internal-only — no authentication is required.

---

## Table of Contents

- [Invoice Lifecycle](#invoice-lifecycle)
- [Base URL](#base-url)
- [Response Envelope](#response-envelope)
- [Error Codes](#error-codes)
- [Endpoints](#endpoints)
  - [System](#system)
  - [Invoices](#invoices)
  - [Payment](#payment)
  - [Tax Authority](#tax-authority)
- [Environment Variables](#environment-variables)

---

## Invoice Lifecycle

```
draft ──► submitted ──► processing ──► completed
                            │
                            ▼
                          failed ──► submitted  (retry via POST /:id/submit)

Any non-completed status ──► cancelled
```

| Status | Description |
|---|---|
| `draft` | Created locally; not yet sent to Viettel |
| `submitted` | Enqueued for background publishing |
| `processing` | Being processed by Viettel SInvoice |
| `completed` | Successfully published and confirmed |
| `failed` | Publishing failed; eligible for retry |
| `cancelled` | Cancelled by request |

---

## Base URL

```
http://localhost:8080
```

---

## Response Envelope

Every endpoint (except `/health` and `/ready`) returns a standard JSON wrapper:

```json
{
  "success": true,
  "data": { ... },
  "meta": { ... },
  "error": null
}
```

| Field | Type | Description |
|---|---|---|
| `success` | boolean | `true` for success, `false` for error |
| `data` | object / array | Response payload (omitted on error) |
| `meta` | object | Pagination info (list endpoints only) |
| `error` | object | Error detail (omitted on success) |

**Error object:**

```json
{
  "success": false,
  "error": {
    "code": "NOT_FOUND",
    "message": "invoice not found"
  }
}
```

**Pagination meta (list endpoints):**

```json
{
  "total": 42,
  "limit": 20,
  "offset": 0
}
```

---

## Error Codes

| Code | HTTP Status | Meaning |
|---|---|---|
| `VALIDATION_ERROR` | 400 | Invalid or missing request fields |
| `NOT_FOUND` | 404 | Resource does not exist |
| `CONFLICT` | 409 | Duplicate or conflicting state |
| `INVALID_STATUS_TRANSITION` | 422 | Action not allowed for current invoice status |
| `THIRD_PARTY_ERROR` | 502 | Viettel SInvoice API failure |
| `TIMEOUT` | 504 | Operation timed out |
| `QUEUE_FULL` | 503 | Background worker queue is at capacity |
| `INTERNAL_ERROR` | 500 | Unexpected server error |

---

## Endpoints

### System

#### `GET /health`

Simple liveness check.

**Response 200**

```json
{ "status": "ok" }
```

---

#### `GET /ready`

Readiness check — pings the database.

**Response 200**

```json
{ "status": "ready" }
```

**Response 503**

```json
{ "status": "not ready", "error": "connection refused" }
```

---

### Invoices

#### `POST /api/v1/invoices`

Creates a new invoice in `draft` status. The invoice is stored locally and is **not** sent to Viettel yet. Use [`POST /:id/submit`](#post-apiv1invoicesidsubmit) to publish it.

**Request Body**

```json
{
  "buyer_name": "Nguyen Van A",
  "buyer_legal_name": "Cong ty TNHH ABC",
  "buyer_tax_code": "0123456789",
  "buyer_address": "123 Le Loi, Quan 1, TP.HCM",
  "buyer_email": "buyer@example.com",
  "buyer_phone": "0901234567",
  "buyer_code": "CUST-001",

  "currency": "VND",
  "total_amount_with_tax": 1100000,
  "total_tax_amount": 100000,
  "total_amount_without_tax": 1000000,

  "token_currency": "HBAR",
  "exchange_rate": 5000.0,
  "exchange_rate_source": "CoinGecko",
  "hbar_amount": 220.0,
  "token_total_amount": 220.0,
  "token_tax_amount": 20.0,
  "token_net_amount": 200.0,

  "payment_method": "TM/CK",
  "transaction_hash": "0x1234abcd...",
  "erp_order_id": "ERP-2025-001",
  "notes": "Thank you for your business",
  "issued_at": "2025-06-15T10:00:00Z",

  "items": [
    {
      "item_name": "Dich vu tu van blockchain",
      "quantity": 2,
      "unit_price": 500000,
      "tax_percentage": 10,
      "tax_amount": 100000,
      "item_total_amount_without_tax": 1000000,
      "item_total_amount_with_tax": 1100000,
      "token_unit_price": 100.0,
      "token_tax_amount": 20.0,
      "token_line_total": 220.0,
      "unit_name": "goi",
      "line_number": 1
    }
  ]
}
```

**Request Fields**

| Field | Type | Required | Constraints | Description |
|---|---|---|---|---|
| `buyer_name` | string | Yes | max 400 | Buyer display name |
| `buyer_legal_name` | string | No | max 400 | Legal / company name |
| `buyer_tax_code` | string | No | max 20 | Tax identification number |
| `buyer_address` | string | No | max 1200 | Full address |
| `buyer_email` | string | No | max 2000 | Email address |
| `buyer_phone` | string | No | max 15 | Phone number |
| `buyer_code` | string | No | max 400 | Internal buyer reference code |
| `currency` | string | Yes | exactly 3 chars | ISO 4217 code for VND amounts, e.g. `VND` |
| `total_amount_with_tax` | float | Yes | | Invoice total including tax (VND) |
| `total_tax_amount` | float | No | | Total tax portion (VND) |
| `total_amount_without_tax` | float | Yes | | Invoice total before tax (VND) |
| `token_currency` | string | Yes | `VND` or `HBAR` | Token used for payment |
| `exchange_rate` | float | Conditional | > 0 | Required when `token_currency` is `HBAR` |
| `exchange_rate_source` | string | No | max 100 | Source of the exchange rate |
| `hbar_amount` | float | No | | Amount in HBAR |
| `token_total_amount` | float | No | | Total in token currency |
| `token_tax_amount` | float | No | | Tax in token currency |
| `token_net_amount` | float | No | | Net amount in token currency |
| `payment_method` | string | No | max 50 | e.g. `TM/CK` |
| `transaction_hash` | string | No | max 255 | Blockchain tx hash (can be set later) |
| `erp_order_id` | string | No | max 255 | External ERP order reference |
| `notes` | string | No | max 500 | Free-text notes |
| `issued_at` | string (RFC 3339) | No | | Invoice issue date; defaults to server time |
| `items` | array | Yes | min 1 item | Line items (see below) |

**Item Fields (`items[]`)**

| Field | Type | Required | Constraints | Description |
|---|---|---|---|---|
| `item_name` | string | Yes | max 500 | Product / service name |
| `quantity` | float | Yes | | Quantity |
| `unit_price` | float | Yes | | Unit price in VND |
| `tax_percentage` | float | No | -2 to 100 | Tax rate. Use `-2` = non-taxable, `-1` = exempt |
| `tax_amount` | float | No | | Tax amount in VND |
| `item_total_amount_without_tax` | float | Yes | | Line total before tax (VND) |
| `item_total_amount_with_tax` | float | No | | Line total after tax (VND) |
| `item_total_amount_after_discount` | float | No | | Line total after discount (VND) |
| `item_discount` | float | No | | Discount amount |
| `token_unit_price` | float | No | | Unit price in token currency |
| `token_tax_amount` | float | No | | Tax in token currency |
| `token_line_total` | float | No | | Line total in token currency |
| `line_number` | integer | No | | Display order |
| `selection` | integer | No | 1–6 | Viettel item type: 1=goods, 2=note, 3=discount, 4=table/fee, 5=promo, 6=special (ND70) |
| `item_type` | integer | No | 1–6 | Required when `selection=6` |
| `item_code` | string | No | max 50 | Internal product code |
| `unit_code` | string | No | max 100 | Unit of measure code |
| `unit_name` | string | No | max 300 | Unit of measure label, e.g. `goi` |
| `discount` | float | No | >= 0 | Discount percentage on unit price |
| `discount2` | float | No | >= 0 | Second discount percentage |
| `item_note` | string | No | max 300 | Note on this line |
| `is_increase_item` | boolean | No | | `null`=normal, `true`=increase adjustment, `false`=decrease adjustment |
| `batch_no` | string | No | max 300 | Batch number |
| `exp_date` | string | No | max 50 | Expiry date |
| `adjust_ratio` | string | No | `"1"`,`"2"`,`"3"`,`"5"` | Adjustment ratio code |
| `unit_price_with_tax` | float | No | | Unit price including tax |
| `special_info` | array | No | | ND70 key-value attributes (see below) |

**`special_info[]` fields:**

| Field | Type | Required |
|---|---|---|
| `name` | string | Yes |
| `value` | string | Yes |

**Response 201**

```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "draft",
    "buyer_name": "Nguyen Van A",
    "currency": "VND",
    "total_amount_with_tax": 1100000,
    "token_currency": "HBAR",
    "exchange_rate": 5000,
    "items": [ { ... } ],
    "created_at": "2025-06-15T10:00:00Z",
    "updated_at": "2025-06-15T10:00:00Z"
  }
}
```

---

#### `GET /api/v1/invoices`

Returns a paginated list of invoices with optional filters.

**Query Parameters**

| Parameter | Type | Default | Description |
|---|---|---|---|
| `status` | string | — | Filter by status: `draft`, `submitted`, `processing`, `completed`, `failed`, `cancelled` |
| `from` | string (RFC 3339) | — | Created after this datetime, e.g. `2025-01-01T00:00:00Z` |
| `to` | string (RFC 3339) | — | Created before this datetime |
| `limit` | integer | `20` | Max results per page |
| `offset` | integer | `0` | Number of results to skip |

**Example**

```
GET /api/v1/invoices?status=completed&from=2025-01-01T00:00:00Z&limit=10&offset=0
```

**Response 200**

```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "status": "completed",
      "buyer_name": "Nguyen Van A",
      "total_amount_with_tax": 1100000,
      "created_at": "2025-06-15T10:00:00Z"
    }
  ],
  "meta": {
    "total": 42,
    "limit": 10,
    "offset": 0
  }
}
```

---

#### `GET /api/v1/invoices/:id`

Returns a single invoice with all line items.

**Path Parameters**

| Parameter | Type | Description |
|---|---|---|
| `id` | UUID | Invoice ID |

**Response 200**

```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "external_id": "C25TAA-0001",
    "transaction_uuid": "AB123-XYZ",
    "status": "completed",
    "buyer_name": "Nguyen Van A",
    "total_amount_with_tax": 1100000,
    "total_tax_amount": 100000,
    "total_amount_without_tax": 1000000,
    "token_currency": "HBAR",
    "exchange_rate": 5000,
    "completed_at": "2025-06-15T10:05:00Z",
    "items": [ { ... } ],
    "created_at": "2025-06-15T10:00:00Z",
    "updated_at": "2025-06-15T10:05:00Z"
  }
}
```

**Error Responses**

| Status | Code | When |
|---|---|---|
| 400 | `VALIDATION_ERROR` | `id` is not a valid UUID |
| 404 | `NOT_FOUND` | Invoice does not exist |

---

#### `DELETE /api/v1/invoices/:id`

Cancels an invoice. Any status except `completed` can be cancelled.

**Path Parameters**

| Parameter | Type | Description |
|---|---|---|
| `id` | UUID | Invoice ID |

**Request Body** _(optional)_

```json
{
  "reason": "Customer requested cancellation"
}
```

**Response 200**

```json
{
  "success": true,
  "data": { "status": "cancelled" }
}
```

**Error Responses**

| Status | Code | When |
|---|---|---|
| 404 | `NOT_FOUND` | Invoice does not exist |
| 422 | `INVALID_STATUS_TRANSITION` | Invoice is already `completed` |

---

#### `POST /api/v1/invoices/:id/submit`

Transitions a `draft` (or `failed`) invoice to `submitted` and enqueues it for asynchronous publishing to Viettel SInvoice. The background worker handles `submitted → processing → completed`.

**Path Parameters**

| Parameter | Type | Description |
|---|---|---|
| `id` | UUID | Invoice ID |

**Request Body** — none

**Response 200**

```json
{
  "success": true,
  "data": { "status": "submitted" }
}
```

**Error Responses**

| Status | Code | When |
|---|---|---|
| 404 | `NOT_FOUND` | Invoice does not exist |
| 422 | `INVALID_STATUS_TRANSITION` | Invoice is not in `draft` or `failed` status |
| 503 | `QUEUE_FULL` | Worker queue is full; try again later |

---

#### `GET /api/v1/invoices/:id/status`

Lightweight endpoint — returns only the current status of an invoice.

**Path Parameters**

| Parameter | Type | Description |
|---|---|---|
| `id` | UUID | Invoice ID |

**Response 200**

```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "processing"
  }
}
```

---

#### `GET /api/v1/invoices/:id/history`

Returns the full status transition history for an invoice, ordered chronologically.

**Path Parameters**

| Parameter | Type | Description |
|---|---|---|
| `id` | UUID | Invoice ID |

**Response 200**

```json
{
  "success": true,
  "data": [
    {
      "id": "661f9511-...",
      "invoice_id": "550e8400-...",
      "from_status": "draft",
      "to_status": "submitted",
      "reason": "",
      "changed_by": "api",
      "created_at": "2025-06-15T10:01:00Z"
    },
    {
      "id": "772a0622-...",
      "invoice_id": "550e8400-...",
      "from_status": "submitted",
      "to_status": "processing",
      "reason": "",
      "changed_by": "worker",
      "created_at": "2025-06-15T10:02:00Z"
    }
  ]
}
```

---

#### `GET /api/v1/invoices/:id/pdf`

Downloads the invoice PDF from Viettel SInvoice and returns it as a base64-encoded data URL. The invoice must be in `completed` status.

**Path Parameters**

| Parameter | Type | Description |
|---|---|---|
| `id` | UUID | Invoice ID |

**Response 200**

```json
{
  "success": true,
  "data": {
    "url": "data:application/pdf;base64,JVBERi0xLjQK...",
    "filename": "invoice_C25TAA-0001.pdf"
  }
}
```

**Error Responses**

| Status | Code | When |
|---|---|---|
| 404 | `NOT_FOUND` | Invoice does not exist |
| 502 | `THIRD_PARTY_ERROR` | Viettel SInvoice returned an error |

---

### Payment

#### `PATCH /api/v1/invoices/:id/payment`

Saves the blockchain transaction hash after on-chain payment completes. This links the local invoice record to the Hedera ledger transaction.

**Path Parameters**

| Parameter | Type | Description |
|---|---|---|
| `id` | UUID | Invoice ID |

**Request Body**

```json
{
  "transaction_hash": "0.0.1234@1718445600.000000000"
}
```

| Field | Type | Required | Constraints | Description |
|---|---|---|---|---|
| `transaction_hash` | string | Yes | max 255 | Hedera / blockchain transaction hash |

**Response 200**

```json
{
  "success": true,
  "data": { "transaction_hash": "0.0.1234@1718445600.000000000" }
}
```

---

### Tax Authority

#### `POST /api/v1/invoices/report-to-authority`

Sends an invoice batch to the Vietnamese tax authority (CQT) via Viettel SInvoice, identified by a Viettel transaction UUID and date range.

**Request Body**

```json
{
  "transaction_uuid": "AB123-XYZ",
  "start_date": "01/06/2025",
  "end_date": "30/06/2025"
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `transaction_uuid` | string | Yes | Viettel transaction UUID identifying the invoice batch |
| `start_date` | string | Yes | Batch start date in `DD/MM/YYYY` format |
| `end_date` | string | Yes | Batch end date in `DD/MM/YYYY` format |

**Response 200**

```json
{
  "success": true,
  "data": {
    "success_count": 10,
    "error_count": 0
  }
}
```

**Error Responses**

| Status | Code | When |
|---|---|---|
| 400 | `VALIDATION_ERROR` | Missing required fields |
| 502 | `THIRD_PARTY_ERROR` | Viettel API returned an error |

---

## Environment Variables

Copy `.env.example` to `.env` and fill in the values marked as required.

```bash
cp .env.example .env
```

### Server

| Variable | Default | Required | Description |
|---|---|---|---|
| `SERVER_PORT` | `8080` | No | TCP port the HTTP server listens on |
| `SERVER_READ_TIMEOUT` | `15s` | No | Maximum duration to read a full request |
| `SERVER_WRITE_TIMEOUT` | `15s` | No | Maximum duration to write the full response |

```env
SERVER_PORT=8080
SERVER_READ_TIMEOUT=15s
SERVER_WRITE_TIMEOUT=15s
```

---

### Database

| Variable | Default | Required | Description |
|---|---|---|---|
| `DB_HOST` | `localhost` | **Yes** | PostgreSQL hostname |
| `DB_PORT` | `5432` | No | PostgreSQL port |
| `DB_USER` | `taxmodule` | No | Database user |
| `DB_PASSWORD` | `secret` | No | Database password |
| `DB_NAME` | `tax_module` | **Yes** | Database name |
| `DB_SSLMODE` | `disable` | No | SSL mode: `disable`, `require`, `verify-full` |
| `DB_MAX_OPEN_CONNS` | `25` | No | Max open connections in the pool |
| `DB_MAX_IDLE_CONNS` | `5` | No | Max idle connections in the pool |

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=taxmodule
DB_PASSWORD=secret
DB_NAME=tax_module
DB_SSLMODE=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
```

> The service builds a DSN in the form:
> `postgres://USER:PASSWORD@HOST:PORT/DBNAME?sslmode=SSLMODE`

---

### Viettel SInvoice (Third Party)

| Variable | Default | Required | Description |
|---|---|---|---|
| `THIRD_PARTY_BASE_URL` | `https://api-vinvoice.viettel.vn/services/einvoiceapplication/api` | No | Base URL of the Viettel SInvoice API |
| `THIRD_PARTY_AUTH_URL` | `https://api-vinvoice.viettel.vn/auth/login` | No | Authentication endpoint URL |
| `THIRD_PARTY_CREATE_PATH` | `/InvoiceAPI/InvoiceWS/createInvoice` | No | Path for creating an invoice |
| `THIRD_PARTY_QUERY_PATH` | `/InvoiceAPI/InvoiceWS/searchInvoiceByTransactionUuid` | No | Path for querying invoice status |
| `THIRD_PARTY_REPORT_TO_AUTHORITY_PATH` | `/InvoiceAPI/InvoiceWS/sendInvoiceByTransactionUuid` | No | Path for reporting to tax authority |
| `THIRD_PARTY_GET_FILE_PATH` | `/InvoiceAPI/InvoiceUtilsWS/getInvoiceRepresentationFile` | No | Path for downloading the invoice PDF |
| `THIRD_PARTY_SUPPLIER_CODE` | _(empty)_ | **Yes** | Supplier code issued by Viettel, e.g. `0100109106-507` |
| `THIRD_PARTY_USERNAME` | _(empty)_ | **Yes** | Login username for the Viettel portal (usually same as supplier code) |
| `THIRD_PARTY_PASSWORD` | _(empty)_ | **Yes** | Login password set on the Viettel portal |
| `THIRD_PARTY_API_KEY` | _(empty)_ | No | Reserved for alternative API-key-based providers; leave empty for Viettel |
| `THIRD_PARTY_TIMEOUT` | `95s` | No | HTTP timeout for Viettel API calls — Viettel recommends >= 90s |
| `THIRD_PARTY_TEMPLATE_CODE` | _(empty)_ | **Yes** | Invoice template code registered with Viettel, e.g. `1/770` |
| `THIRD_PARTY_INVOICE_SERIES` | _(empty)_ | **Yes** | Invoice series registered with Viettel, e.g. `K23TXM` |
| `THIRD_PARTY_INVOICE_TYPE` | `1` | No | Invoice type per Circular TT78 (1–6); `1` = standard VAT invoice |

```env
THIRD_PARTY_BASE_URL=https://api-vinvoice.viettel.vn/services/einvoiceapplication/api
THIRD_PARTY_AUTH_URL=https://api-vinvoice.viettel.vn/auth/login
THIRD_PARTY_CREATE_PATH=/InvoiceAPI/InvoiceWS/createInvoice
THIRD_PARTY_QUERY_PATH=/InvoiceAPI/InvoiceWS/searchInvoiceByTransactionUuid
THIRD_PARTY_REPORT_TO_AUTHORITY_PATH=/InvoiceAPI/InvoiceWS/sendInvoiceByTransactionUuid
THIRD_PARTY_GET_FILE_PATH=/InvoiceAPI/InvoiceUtilsWS/getInvoiceRepresentationFile
THIRD_PARTY_SUPPLIER_CODE=0100109106-507
THIRD_PARTY_USERNAME=0100109106-507
THIRD_PARTY_PASSWORD=your-viettel-password
THIRD_PARTY_API_KEY=
THIRD_PARTY_TIMEOUT=95s
THIRD_PARTY_TEMPLATE_CODE=1/770
THIRD_PARTY_INVOICE_SERIES=K23TXM
THIRD_PARTY_INVOICE_TYPE=1
```

---

### Worker

Background pool that asynchronously publishes submitted invoices to Viettel and polls for their status.

| Variable | Default | Required | Description |
|---|---|---|---|
| `WORKER_POOL_SIZE` | `10` | No | Number of concurrent goroutines in the worker pool |
| `WORKER_QUEUE_SIZE` | `100` | No | Max pending jobs in the in-memory queue; returns `QUEUE_FULL` when full |
| `WORKER_POLL_INTERVAL` | `60s` | No | How often the background poller re-checks `submitted`/`processing` invoices |
| `WORKER_MAX_RETRIES` | `5` | No | Maximum attempts before marking an invoice as `failed` permanently |

```env
WORKER_POOL_SIZE=10
WORKER_QUEUE_SIZE=100
WORKER_POLL_INTERVAL=60s
WORKER_MAX_RETRIES=5
```

---

### Logging

| Variable | Default | Description |
|---|---|---|
| `LOG_LEVEL` | `info` | Log verbosity: `trace`, `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `console` | Output format: `console` (human-readable) or `json` (structured) |

```env
LOG_LEVEL=info
LOG_FORMAT=json
```

> Use `LOG_FORMAT=json` in production for log aggregation tools; `console` is easier to read during local development.

---

### Seller Information

Used to populate the seller fields on every invoice sent to Viettel. Must match the information registered on the Viettel portal.

| Variable | Default | Required | Description |
|---|---|---|---|
| `SELLER_LEGAL_NAME` | _(empty)_ | **Yes** | Company legal name as registered |
| `SELLER_TAX_CODE` | _(empty)_ | **Yes** | Company tax code (MST) |
| `SELLER_ADDRESS` | _(empty)_ | **Yes** | Registered business address |
| `SELLER_PHONE` | _(empty)_ | No | Contact phone number |
| `SELLER_EMAIL` | _(empty)_ | No | Contact email address |
| `SELLER_BANK_NAME` | _(empty)_ | No | Bank name for invoice display |
| `SELLER_BANK_ACCOUNT` | _(empty)_ | No | Bank account number |

```env
SELLER_LEGAL_NAME="CÔNG TY TNHH ABC"
SELLER_TAX_CODE="0100109106"
SELLER_ADDRESS="123 Nguyen Hue, Quan 1, TP.HCM"
SELLER_PHONE="0901234567"
SELLER_EMAIL="invoice@company.com"
SELLER_BANK_NAME="Vietcombank"
SELLER_BANK_ACCOUNT="1234567890"
```
