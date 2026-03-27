package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"tax-module/internal/domain"
)

// InvoiceRepo implements domain.InvoiceRepository using PostgreSQL.
type InvoiceRepo struct {
	pool *pgxpool.Pool
	log  *zerolog.Logger
}

func NewInvoiceRepo(pool *pgxpool.Pool, log *zerolog.Logger) *InvoiceRepo {
	return &InvoiceRepo{pool: pool, log: log}
}

// invoiceColumns is the shared column list for SELECT queries on the invoices table.
const invoiceColumns = `id, external_id, transaction_uuid, status,
	buyer_name, buyer_legal_name, buyer_tax_code, buyer_address,
	buyer_email, buyer_phone, buyer_code,
	currency, total_amount_with_tax, total_tax_amount, total_amount_without_tax,
	token_currency, exchange_rate, exchange_rate_source, hbar_amount,
	token_total_amount, token_tax_amount, token_net_amount,
	payment_method, transaction_hash, erp_order_id,
	notes, issued_at, submitted_at, completed_at,
	retry_count, last_error, metadata, created_at, updated_at`

// scanInvoice scans a row into a domain.Invoice using the standard column order.
func scanInvoice(row interface{ Scan(dest ...any) error }) (*domain.Invoice, error) {
	var inv domain.Invoice
	err := row.Scan(
		&inv.ID, &inv.ExternalID, &inv.TransactionUuid, &inv.Status,
		&inv.BuyerName, &inv.BuyerLegalName, &inv.BuyerTaxCode, &inv.BuyerAddress,
		&inv.BuyerEmail, &inv.BuyerPhone, &inv.BuyerCode,
		&inv.Currency, &inv.TotalAmountWithTax, &inv.TotalTaxAmount, &inv.TotalAmountWithoutTax,
		&inv.TokenCurrency, &inv.ExchangeRate, &inv.ExchangeRateSource, &inv.HbarAmount,
		&inv.TokenTotalAmount, &inv.TokenTaxAmount, &inv.TokenNetAmount,
		&inv.PaymentMethod, &inv.TransactionHash, &inv.ErpOrderID,
		&inv.Notes, &inv.IssuedAt, &inv.SubmittedAt, &inv.CompletedAt,
		&inv.RetryCount, &inv.LastError, &inv.Metadata, &inv.CreatedAt, &inv.UpdatedAt,
	)
	return &inv, err
}

// --- Invoice CRUD ---

func (r *InvoiceRepo) Create(ctx context.Context, invoice *domain.Invoice) error {
	query := `
		INSERT INTO invoices (id, external_id, transaction_uuid, status,
			buyer_name, buyer_legal_name, buyer_tax_code, buyer_address,
			buyer_email, buyer_phone, buyer_code,
			currency, total_amount_with_tax, total_tax_amount, total_amount_without_tax,
			token_currency, exchange_rate, exchange_rate_source, hbar_amount,
			token_total_amount, token_tax_amount, token_net_amount,
			payment_method, transaction_hash, erp_order_id,
			notes, issued_at, metadata, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30)`

	_, err := r.pool.Exec(ctx, query,
		invoice.ID, invoice.ExternalID, invoice.TransactionUuid, invoice.Status,
		invoice.BuyerName, invoice.BuyerLegalName, invoice.BuyerTaxCode, invoice.BuyerAddress,
		invoice.BuyerEmail, invoice.BuyerPhone, invoice.BuyerCode,
		invoice.Currency, invoice.TotalAmountWithTax, invoice.TotalTaxAmount, invoice.TotalAmountWithoutTax,
		invoice.TokenCurrency, invoice.ExchangeRate, invoice.ExchangeRateSource, invoice.HbarAmount,
		invoice.TokenTotalAmount, invoice.TokenTaxAmount, invoice.TokenNetAmount,
		invoice.PaymentMethod, invoice.TransactionHash, invoice.ErpOrderID,
		invoice.Notes, invoice.IssuedAt, invoice.Metadata, invoice.CreatedAt, invoice.UpdatedAt,
	)
	if err != nil {
		return domain.NewInternalError("failed to create invoice", err)
	}
	return nil
}

func (r *InvoiceRepo) CreateWithItems(ctx context.Context, invoice *domain.Invoice) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.NewInternalError("failed to begin transaction", err)
	}
	defer tx.Rollback(ctx)

	invoiceQuery := `
		INSERT INTO invoices (id, external_id, transaction_uuid, status,
			buyer_name, buyer_legal_name, buyer_tax_code, buyer_address,
			buyer_email, buyer_phone, buyer_code,
			currency, total_amount_with_tax, total_tax_amount, total_amount_without_tax,
			token_currency, exchange_rate, exchange_rate_source, hbar_amount,
			token_total_amount, token_tax_amount, token_net_amount,
			payment_method, transaction_hash, erp_order_id,
			notes, issued_at, metadata, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30)`

	_, err = tx.Exec(ctx, invoiceQuery,
		invoice.ID, invoice.ExternalID, invoice.TransactionUuid, invoice.Status,
		invoice.BuyerName, invoice.BuyerLegalName, invoice.BuyerTaxCode, invoice.BuyerAddress,
		invoice.BuyerEmail, invoice.BuyerPhone, invoice.BuyerCode,
		invoice.Currency, invoice.TotalAmountWithTax, invoice.TotalTaxAmount, invoice.TotalAmountWithoutTax,
		invoice.TokenCurrency, invoice.ExchangeRate, invoice.ExchangeRateSource, invoice.HbarAmount,
		invoice.TokenTotalAmount, invoice.TokenTaxAmount, invoice.TokenNetAmount,
		invoice.PaymentMethod, invoice.TransactionHash, invoice.ErpOrderID,
		invoice.Notes, invoice.IssuedAt, invoice.Metadata, invoice.CreatedAt, invoice.UpdatedAt,
	)
	if err != nil {
		return domain.NewInternalError("failed to create invoice", err)
	}

	itemQuery := `
		INSERT INTO invoice_items (id, invoice_id, item_name, quantity, unit_price,
			tax_percentage, tax_amount, item_total_amount_without_tax,
			item_total_amount_with_tax, item_total_amount_after_discount, item_discount,
			token_unit_price, token_tax_amount, token_line_total,
			line_number, created_at,
			selection, item_type, item_code, unit_code, unit_name,
			discount, discount2, item_note, is_increase_item,
			batch_no, exp_date, adjust_ratio, unit_price_with_tax, special_info)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,
			$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30)`

	for _, item := range invoice.Items {
		specialInfoJSON, err := json.Marshal(item.SpecialInfo)
		if err != nil {
			return domain.NewInternalError("failed to marshal special_info", err)
		}

		_, err = tx.Exec(ctx, itemQuery,
			item.ID, item.InvoiceID, item.ItemName, item.Quantity, item.UnitPrice,
			item.TaxPercentage, item.TaxAmount, item.ItemTotalAmountWithoutTax,
			item.ItemTotalAmountWithTax, item.ItemTotalAmountAfterDiscount, item.ItemDiscount,
			item.TokenUnitPrice, item.TokenTaxAmount, item.TokenLineTotal,
			item.LineNumber, item.CreatedAt,
			item.Selection, item.ItemType, item.ItemCode, item.UnitCode, item.UnitName,
			item.Discount, item.Discount2, item.ItemNote, item.IsIncreaseItem,
			item.BatchNo, item.ExpDate, item.AdjustRatio, item.UnitPriceWithTax, specialInfoJSON,
		)
		if err != nil {
			return domain.NewInternalError("failed to add item", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.NewInternalError("failed to commit transaction", err)
	}
	return nil
}

func (r *InvoiceRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	query := `SELECT ` + invoiceColumns + ` FROM invoices WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)
	inv, err := scanInvoice(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewNotFoundError("invoice not found")
		}
		return nil, domain.NewInternalError("failed to get invoice", err)
	}
	return inv, nil
}

func (r *InvoiceRepo) Update(ctx context.Context, invoice *domain.Invoice) error {
	query := `
		UPDATE invoices SET
			external_id=$1, transaction_uuid=$2,
			buyer_name=$3, buyer_legal_name=$4, buyer_tax_code=$5, buyer_address=$6,
			buyer_email=$7, buyer_phone=$8, buyer_code=$9,
			currency=$10, total_amount_with_tax=$11, total_tax_amount=$12, total_amount_without_tax=$13,
			token_currency=$14, exchange_rate=$15, exchange_rate_source=$16, hbar_amount=$17,
			token_total_amount=$18, token_tax_amount=$19, token_net_amount=$20,
			payment_method=$21, transaction_hash=$22, erp_order_id=$23,
			notes=$24, issued_at=$25,
			submitted_at=$26, completed_at=$27, retry_count=$28, last_error=$29,
			metadata=$30, updated_at=$31
		WHERE id = $32`

	tag, err := r.pool.Exec(ctx, query,
		invoice.ExternalID, invoice.TransactionUuid,
		invoice.BuyerName, invoice.BuyerLegalName, invoice.BuyerTaxCode, invoice.BuyerAddress,
		invoice.BuyerEmail, invoice.BuyerPhone, invoice.BuyerCode,
		invoice.Currency, invoice.TotalAmountWithTax, invoice.TotalTaxAmount, invoice.TotalAmountWithoutTax,
		invoice.TokenCurrency, invoice.ExchangeRate, invoice.ExchangeRateSource, invoice.HbarAmount,
		invoice.TokenTotalAmount, invoice.TokenTaxAmount, invoice.TokenNetAmount,
		invoice.PaymentMethod, invoice.TransactionHash, invoice.ErpOrderID,
		invoice.Notes, invoice.IssuedAt,
		invoice.SubmittedAt, invoice.CompletedAt, invoice.RetryCount, invoice.LastError,
		invoice.Metadata, invoice.UpdatedAt, invoice.ID,
	)
	if err != nil {
		return domain.NewInternalError("failed to update invoice", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.NewNotFoundError("invoice not found")
	}
	return nil
}

func (r *InvoiceRepo) UpdateTransactionHash(ctx context.Context, id uuid.UUID, transactionHash string) error {
	query := `UPDATE invoices SET transaction_hash = $1, updated_at = $2 WHERE id = $3`
	tag, err := r.pool.Exec(ctx, query, transactionHash, time.Now(), id)
	if err != nil {
		return domain.NewInternalError("failed to update transaction hash", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.NewNotFoundError("invoice not found")
	}
	return nil
}

func (r *InvoiceRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.InvoiceStatus, reason string) error {
	query := `UPDATE invoices SET status=$1, last_error=$2, updated_at=$3`

	args := []interface{}{status, reason, time.Now()}
	argIdx := 4

	if status == domain.StatusSubmitted {
		query += fmt.Sprintf(", submitted_at=$%d", argIdx)
		now := time.Now()
		args = append(args, now)
		argIdx++
	}
	if status == domain.StatusCompleted {
		query += fmt.Sprintf(", completed_at=$%d", argIdx)
		now := time.Now()
		args = append(args, now)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE id=$%d", argIdx)
	args = append(args, id)

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return domain.NewInternalError("failed to update status", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.NewNotFoundError("invoice not found")
	}
	return nil
}

func (r *InvoiceRepo) List(ctx context.Context, filter domain.InvoiceFilter) ([]*domain.Invoice, int64, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.FromDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filter.FromDate)
		argIdx++
	}
	if filter.ToDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filter.ToDate)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM invoices" + where
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, domain.NewInternalError("failed to count invoices", err)
	}

	// Fetch page
	dataQuery := fmt.Sprintf(
		`SELECT `+invoiceColumns+` FROM invoices%s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, argIdx, argIdx+1,
	)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, domain.NewInternalError("failed to list invoices", err)
	}
	defer rows.Close()

	var invoices []*domain.Invoice
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, 0, domain.NewInternalError("failed to scan invoice", err)
		}
		invoices = append(invoices, inv)
	}

	return invoices, total, nil
}

func (r *InvoiceRepo) GetByExternalID(ctx context.Context, externalID string) (*domain.Invoice, error) {
	query := `SELECT ` + invoiceColumns + ` FROM invoices WHERE external_id = $1`

	row := r.pool.QueryRow(ctx, query, externalID)
	inv, err := scanInvoice(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewNotFoundError("invoice not found for external_id: " + externalID)
		}
		return nil, domain.NewInternalError("failed to get invoice by external_id", err)
	}
	return inv, nil
}

func (r *InvoiceRepo) GetByTransactionUuid(ctx context.Context, transactionUuid string) (*domain.Invoice, error) {
	query := `SELECT ` + invoiceColumns + ` FROM invoices WHERE transaction_uuid = $1`

	row := r.pool.QueryRow(ctx, query, transactionUuid)
	inv, err := scanInvoice(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewNotFoundError("invoice not found for transaction_uuid: " + transactionUuid)
		}
		return nil, domain.NewInternalError("failed to get invoice by transaction_uuid", err)
	}
	return inv, nil
}

func (r *InvoiceRepo) GetPendingPolling(ctx context.Context, limit int) ([]*domain.Invoice, error) {
	query := `SELECT ` + invoiceColumns + `
		FROM invoices
		WHERE status IN ('submitted', 'processing')
		ORDER BY updated_at ASC
		LIMIT $1`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, domain.NewInternalError("failed to get pending invoices", err)
	}
	defer rows.Close()

	var invoices []*domain.Invoice
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, domain.NewInternalError("failed to scan invoice", err)
		}
		invoices = append(invoices, inv)
	}
	return invoices, nil
}

// --- Items ---

// itemColumns is the shared column list for SELECT queries on the invoice_items table.
const itemColumns = `id, invoice_id, item_name, quantity, unit_price,
	tax_percentage, tax_amount, item_total_amount_without_tax,
	item_total_amount_with_tax, item_total_amount_after_discount, item_discount,
	token_unit_price, token_tax_amount, token_line_total,
	line_number, created_at,
	selection, item_type, item_code, unit_code, unit_name,
	discount, discount2, item_note, is_increase_item,
	batch_no, exp_date, adjust_ratio, unit_price_with_tax, special_info`

func (r *InvoiceRepo) AddItem(ctx context.Context, item *domain.InvoiceItem) error {
	query := `
		INSERT INTO invoice_items (id, invoice_id, item_name, quantity, unit_price,
			tax_percentage, tax_amount, item_total_amount_without_tax,
			item_total_amount_with_tax, item_total_amount_after_discount, item_discount,
			token_unit_price, token_tax_amount, token_line_total,
			line_number, created_at,
			selection, item_type, item_code, unit_code, unit_name,
			discount, discount2, item_note, is_increase_item,
			batch_no, exp_date, adjust_ratio, unit_price_with_tax, special_info)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,
			$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30)`

	specialInfoJSON, err := json.Marshal(item.SpecialInfo)
	if err != nil {
		return domain.NewInternalError("failed to marshal special_info", err)
	}

	_, err = r.pool.Exec(ctx, query,
		item.ID, item.InvoiceID, item.ItemName, item.Quantity, item.UnitPrice,
		item.TaxPercentage, item.TaxAmount, item.ItemTotalAmountWithoutTax,
		item.ItemTotalAmountWithTax, item.ItemTotalAmountAfterDiscount, item.ItemDiscount,
		item.TokenUnitPrice, item.TokenTaxAmount, item.TokenLineTotal,
		item.LineNumber, item.CreatedAt,
		item.Selection, item.ItemType, item.ItemCode, item.UnitCode, item.UnitName,
		item.Discount, item.Discount2, item.ItemNote, item.IsIncreaseItem,
		item.BatchNo, item.ExpDate, item.AdjustRatio, item.UnitPriceWithTax, specialInfoJSON,
	)
	if err != nil {
		return domain.NewInternalError("failed to add item", err)
	}
	return nil
}

func (r *InvoiceRepo) UpdateItem(ctx context.Context, item *domain.InvoiceItem) error {
	query := `
		UPDATE invoice_items SET
			item_name=$1, quantity=$2, unit_price=$3,
			tax_percentage=$4, tax_amount=$5,
			item_total_amount_without_tax=$6, item_total_amount_with_tax=$7,
			item_total_amount_after_discount=$8, item_discount=$9,
			token_unit_price=$10, token_tax_amount=$11, token_line_total=$12
		WHERE id = $13`

	tag, err := r.pool.Exec(ctx, query,
		item.ItemName, item.Quantity, item.UnitPrice,
		item.TaxPercentage, item.TaxAmount,
		item.ItemTotalAmountWithoutTax, item.ItemTotalAmountWithTax,
		item.ItemTotalAmountAfterDiscount, item.ItemDiscount,
		item.TokenUnitPrice, item.TokenTaxAmount, item.TokenLineTotal,
		item.ID,
	)
	if err != nil {
		return domain.NewInternalError("failed to update item", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.NewNotFoundError("item not found")
	}
	return nil
}

func (r *InvoiceRepo) GetItemsByInvoiceID(ctx context.Context, invoiceID uuid.UUID) ([]*domain.InvoiceItem, error) {
	query := `SELECT ` + itemColumns + ` FROM invoice_items WHERE invoice_id = $1
		ORDER BY line_number, created_at`

	rows, err := r.pool.Query(ctx, query, invoiceID)
	if err != nil {
		return nil, domain.NewInternalError("failed to get items", err)
	}
	defer rows.Close()

	var items []*domain.InvoiceItem
	for rows.Next() {
		var item domain.InvoiceItem
		var specialInfoJSON []byte
		if err := rows.Scan(
			&item.ID, &item.InvoiceID, &item.ItemName, &item.Quantity, &item.UnitPrice,
			&item.TaxPercentage, &item.TaxAmount, &item.ItemTotalAmountWithoutTax,
			&item.ItemTotalAmountWithTax, &item.ItemTotalAmountAfterDiscount, &item.ItemDiscount,
			&item.TokenUnitPrice, &item.TokenTaxAmount, &item.TokenLineTotal,
			&item.LineNumber, &item.CreatedAt,
			&item.Selection, &item.ItemType, &item.ItemCode, &item.UnitCode, &item.UnitName,
			&item.Discount, &item.Discount2, &item.ItemNote, &item.IsIncreaseItem,
			&item.BatchNo, &item.ExpDate, &item.AdjustRatio, &item.UnitPriceWithTax, &specialInfoJSON,
		); err != nil {
			return nil, domain.NewInternalError("failed to scan item", err)
		}
		if len(specialInfoJSON) > 0 {
			_ = json.Unmarshal(specialInfoJSON, &item.SpecialInfo)
		}
		items = append(items, &item)
	}
	return items, nil
}

func (r *InvoiceRepo) DeleteItem(ctx context.Context, itemID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM invoice_items WHERE id = $1", itemID)
	if err != nil {
		return domain.NewInternalError("failed to delete item", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.NewNotFoundError("item not found")
	}
	return nil
}

// --- Status history ---

func (r *InvoiceRepo) AddStatusHistory(ctx context.Context, h *domain.InvoiceStatusHistory) error {
	query := `
		INSERT INTO invoice_status_history (id, invoice_id, from_status, to_status, reason, changed_by, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`

	_, err := r.pool.Exec(ctx, query,
		h.ID, h.InvoiceID, h.FromStatus, h.ToStatus, h.Reason, h.ChangedBy, h.CreatedAt,
	)
	if err != nil {
		return domain.NewInternalError("failed to add status history", err)
	}
	return nil
}

func (r *InvoiceRepo) GetStatusHistory(ctx context.Context, invoiceID uuid.UUID) ([]*domain.InvoiceStatusHistory, error) {
	query := `
		SELECT id, invoice_id, from_status, to_status, reason, changed_by, created_at
		FROM invoice_status_history WHERE invoice_id = $1
		ORDER BY created_at`

	rows, err := r.pool.Query(ctx, query, invoiceID)
	if err != nil {
		return nil, domain.NewInternalError("failed to get status history", err)
	}
	defer rows.Close()

	var history []*domain.InvoiceStatusHistory
	for rows.Next() {
		var h domain.InvoiceStatusHistory
		if err := rows.Scan(&h.ID, &h.InvoiceID, &h.FromStatus, &h.ToStatus, &h.Reason, &h.ChangedBy, &h.CreatedAt); err != nil {
			return nil, domain.NewInternalError("failed to scan status history", err)
		}
		history = append(history, &h)
	}
	return history, nil
}

// --- Audit ---

func (r *InvoiceRepo) AddAuditLog(ctx context.Context, a *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (id, entity_type, entity_id, action, actor, old_data, new_data, request_id, ip_address, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`

	_, err := r.pool.Exec(ctx, query,
		a.ID, a.EntityType, a.EntityID, a.Action, a.Actor,
		a.OldData, a.NewData, a.RequestID, a.IPAddress, a.CreatedAt,
	)
	if err != nil {
		return domain.NewInternalError("failed to add audit log", err)
	}
	return nil
}
