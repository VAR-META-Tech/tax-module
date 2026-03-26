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

// --- Invoice CRUD ---

func (r *InvoiceRepo) Create(ctx context.Context, invoice *domain.Invoice) error {
	query := `
		INSERT INTO invoices (id, external_id, transaction_uuid, status, customer_name, customer_tax_id,
			customer_address, currency, original_currency, exchange_rate,
			total_amount, tax_amount, net_amount,
			original_total_amount, original_tax_amount, original_net_amount,
			transaction_hash, notes, issued_at, due_at, metadata, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)`

	_, err := r.pool.Exec(ctx, query,
		invoice.ID, invoice.ExternalID, invoice.TransactionUuid, invoice.Status,
		invoice.CustomerName, invoice.CustomerTaxID, invoice.CustomerAddress,
		invoice.Currency, invoice.OriginalCurrency, invoice.ExchangeRate,
		invoice.TotalAmount, invoice.TaxAmount, invoice.NetAmount,
		invoice.OriginalTotalAmount, invoice.OriginalTaxAmount, invoice.OriginalNetAmount,
		invoice.TransactionHash, invoice.Notes, invoice.IssuedAt, invoice.DueAt,
		invoice.Metadata, invoice.CreatedAt, invoice.UpdatedAt,
	)
	if err != nil {
		return domain.NewInternalError("failed to create invoice", err)
	}
	return nil
}

func (r *InvoiceRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Invoice, error) {
	query := `
		SELECT id, external_id, transaction_uuid, status, customer_name, customer_tax_id,
			customer_address, currency, original_currency, exchange_rate,
			total_amount, tax_amount, net_amount,
			original_total_amount, original_tax_amount, original_net_amount,
			transaction_hash, notes, issued_at, due_at, submitted_at, completed_at,
			retry_count, last_error, metadata, created_at, updated_at
		FROM invoices WHERE id = $1`

	var inv domain.Invoice
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&inv.ID, &inv.ExternalID, &inv.TransactionUuid, &inv.Status,
		&inv.CustomerName, &inv.CustomerTaxID, &inv.CustomerAddress,
		&inv.Currency, &inv.OriginalCurrency, &inv.ExchangeRate,
		&inv.TotalAmount, &inv.TaxAmount, &inv.NetAmount,
		&inv.OriginalTotalAmount, &inv.OriginalTaxAmount, &inv.OriginalNetAmount,
		&inv.TransactionHash, &inv.Notes, &inv.IssuedAt, &inv.DueAt, &inv.SubmittedAt, &inv.CompletedAt,
		&inv.RetryCount, &inv.LastError, &inv.Metadata, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewNotFoundError("invoice not found")
		}
		return nil, domain.NewInternalError("failed to get invoice", err)
	}
	return &inv, nil
}

func (r *InvoiceRepo) Update(ctx context.Context, invoice *domain.Invoice) error {
	query := `
		UPDATE invoices SET
			external_id=$1, transaction_uuid=$2,
			customer_name=$3, customer_tax_id=$4, customer_address=$5,
			currency=$6, original_currency=$7, exchange_rate=$8,
			total_amount=$9, tax_amount=$10, net_amount=$11,
			original_total_amount=$12, original_tax_amount=$13, original_net_amount=$14,
			transaction_hash=$15, notes=$16, issued_at=$17, due_at=$18,
			submitted_at=$19, completed_at=$20, retry_count=$21, last_error=$22,
			metadata=$23, updated_at=$24
		WHERE id = $25`

	tag, err := r.pool.Exec(ctx, query,
		invoice.ExternalID, invoice.TransactionUuid,
		invoice.CustomerName, invoice.CustomerTaxID, invoice.CustomerAddress,
		invoice.Currency, invoice.OriginalCurrency, invoice.ExchangeRate,
		invoice.TotalAmount, invoice.TaxAmount, invoice.NetAmount,
		invoice.OriginalTotalAmount, invoice.OriginalTaxAmount, invoice.OriginalNetAmount,
		invoice.TransactionHash, invoice.Notes, invoice.IssuedAt, invoice.DueAt,
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
		`SELECT id, external_id, transaction_uuid, status, customer_name, customer_tax_id,
			customer_address, currency, original_currency, exchange_rate,
			total_amount, tax_amount, net_amount,
			original_total_amount, original_tax_amount, original_net_amount,
			transaction_hash, notes, issued_at, due_at, submitted_at, completed_at,
			retry_count, last_error, metadata, created_at, updated_at
		FROM invoices%s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
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
		var inv domain.Invoice
		if err := rows.Scan(
			&inv.ID, &inv.ExternalID, &inv.TransactionUuid, &inv.Status,
			&inv.CustomerName, &inv.CustomerTaxID, &inv.CustomerAddress,
			&inv.Currency, &inv.OriginalCurrency, &inv.ExchangeRate,
			&inv.TotalAmount, &inv.TaxAmount, &inv.NetAmount,
			&inv.OriginalTotalAmount, &inv.OriginalTaxAmount, &inv.OriginalNetAmount,
			&inv.TransactionHash, &inv.Notes, &inv.IssuedAt, &inv.DueAt, &inv.SubmittedAt, &inv.CompletedAt,
			&inv.RetryCount, &inv.LastError, &inv.Metadata, &inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, 0, domain.NewInternalError("failed to scan invoice", err)
		}
		invoices = append(invoices, &inv)
	}

	return invoices, total, nil
}

func (r *InvoiceRepo) GetByExternalID(ctx context.Context, externalID string) (*domain.Invoice, error) {
	query := `
		SELECT id, external_id, transaction_uuid, status, customer_name, customer_tax_id,
			customer_address, currency, original_currency, exchange_rate,
			total_amount, tax_amount, net_amount,
			original_total_amount, original_tax_amount, original_net_amount,
			transaction_hash, notes, issued_at, due_at, submitted_at, completed_at,
			retry_count, last_error, metadata, created_at, updated_at
		FROM invoices WHERE external_id = $1`

	var inv domain.Invoice
	err := r.pool.QueryRow(ctx, query, externalID).Scan(
		&inv.ID, &inv.ExternalID, &inv.TransactionUuid, &inv.Status,
		&inv.CustomerName, &inv.CustomerTaxID, &inv.CustomerAddress,
		&inv.Currency, &inv.OriginalCurrency, &inv.ExchangeRate,
		&inv.TotalAmount, &inv.TaxAmount, &inv.NetAmount,
		&inv.OriginalTotalAmount, &inv.OriginalTaxAmount, &inv.OriginalNetAmount,
		&inv.TransactionHash, &inv.Notes, &inv.IssuedAt, &inv.DueAt, &inv.SubmittedAt, &inv.CompletedAt,
		&inv.RetryCount, &inv.LastError, &inv.Metadata, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewNotFoundError("invoice not found for external_id: " + externalID)
		}
		return nil, domain.NewInternalError("failed to get invoice by external_id", err)
	}
	return &inv, nil
}

func (r *InvoiceRepo) GetByTransactionUuid(ctx context.Context, transactionUuid string) (*domain.Invoice, error) {
	query := `
		SELECT id, external_id, transaction_uuid, status, customer_name, customer_tax_id,
			customer_address, currency, original_currency, exchange_rate,
			total_amount, tax_amount, net_amount,
			original_total_amount, original_tax_amount, original_net_amount,
			transaction_hash, notes, issued_at, due_at, submitted_at, completed_at,
			retry_count, last_error, metadata, created_at, updated_at
		FROM invoices WHERE transaction_uuid = $1`

	var inv domain.Invoice
	err := r.pool.QueryRow(ctx, query, transactionUuid).Scan(
		&inv.ID, &inv.ExternalID, &inv.TransactionUuid, &inv.Status,
		&inv.CustomerName, &inv.CustomerTaxID, &inv.CustomerAddress,
		&inv.Currency, &inv.OriginalCurrency, &inv.ExchangeRate,
		&inv.TotalAmount, &inv.TaxAmount, &inv.NetAmount,
		&inv.OriginalTotalAmount, &inv.OriginalTaxAmount, &inv.OriginalNetAmount,
		&inv.TransactionHash, &inv.Notes, &inv.IssuedAt, &inv.DueAt, &inv.SubmittedAt, &inv.CompletedAt,
		&inv.RetryCount, &inv.LastError, &inv.Metadata, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewNotFoundError("invoice not found for transaction_uuid: " + transactionUuid)
		}
		return nil, domain.NewInternalError("failed to get invoice by transaction_uuid", err)
	}
	return &inv, nil
}

func (r *InvoiceRepo) GetPendingPolling(ctx context.Context, limit int) ([]*domain.Invoice, error) {
	query := `
		SELECT id, external_id, transaction_uuid, status, customer_name, customer_tax_id,
			customer_address, currency, original_currency, exchange_rate,
			total_amount, tax_amount, net_amount,
			original_total_amount, original_tax_amount, original_net_amount,
			transaction_hash, notes, issued_at, due_at, submitted_at, completed_at,
			retry_count, last_error, metadata, created_at, updated_at
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
		var inv domain.Invoice
		if err := rows.Scan(
			&inv.ID, &inv.ExternalID, &inv.TransactionUuid, &inv.Status,
			&inv.CustomerName, &inv.CustomerTaxID, &inv.CustomerAddress,
			&inv.Currency, &inv.OriginalCurrency, &inv.ExchangeRate,
			&inv.TotalAmount, &inv.TaxAmount, &inv.NetAmount,
			&inv.OriginalTotalAmount, &inv.OriginalTaxAmount, &inv.OriginalNetAmount,
			&inv.TransactionHash, &inv.Notes, &inv.IssuedAt, &inv.DueAt, &inv.SubmittedAt, &inv.CompletedAt,
			&inv.RetryCount, &inv.LastError, &inv.Metadata, &inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, domain.NewInternalError("failed to scan invoice", err)
		}
		invoices = append(invoices, &inv)
	}
	return invoices, nil
}

// --- Items ---

func (r *InvoiceRepo) AddItem(ctx context.Context, item *domain.InvoiceItem) error {
	query := `
		INSERT INTO invoice_items (id, invoice_id, description, quantity, unit_price,
			tax_rate, tax_amount, line_total,
			original_unit_price, original_tax_amount, original_line_total,
			sort_order, created_at,
			selection, item_type, item_code, unit_code, unit_name,
			discount, discount2, item_note, is_increase_item,
			batch_no, exp_date, adjust_ratio, unit_price_with_tax, special_info)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,
			$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27)`

	specialInfoJSON, err := json.Marshal(item.SpecialInfo)
	if err != nil {
		return domain.NewInternalError("failed to marshal special_info", err)
	}

	_, err = r.pool.Exec(ctx, query,
		item.ID, item.InvoiceID, item.Description, item.Quantity, item.UnitPrice,
		item.TaxRate, item.TaxAmount, item.LineTotal,
		item.OriginalUnitPrice, item.OriginalTaxAmount, item.OriginalLineTotal,
		item.SortOrder, item.CreatedAt,
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
			unit_price=$1, tax_amount=$2, line_total=$3,
			original_unit_price=$4, original_tax_amount=$5, original_line_total=$6
		WHERE id = $7`

	tag, err := r.pool.Exec(ctx, query,
		item.UnitPrice, item.TaxAmount, item.LineTotal,
		item.OriginalUnitPrice, item.OriginalTaxAmount, item.OriginalLineTotal,
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
	query := `
		SELECT id, invoice_id, description, quantity, unit_price,
			tax_rate, tax_amount, line_total,
			original_unit_price, original_tax_amount, original_line_total,
			sort_order, created_at,
			selection, item_type, item_code, unit_code, unit_name,
			discount, discount2, item_note, is_increase_item,
			batch_no, exp_date, adjust_ratio, unit_price_with_tax, special_info
		FROM invoice_items WHERE invoice_id = $1
		ORDER BY sort_order, created_at`

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
			&item.ID, &item.InvoiceID, &item.Description, &item.Quantity, &item.UnitPrice,
			&item.TaxRate, &item.TaxAmount, &item.LineTotal,
			&item.OriginalUnitPrice, &item.OriginalTaxAmount, &item.OriginalLineTotal,
			&item.SortOrder, &item.CreatedAt,
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
