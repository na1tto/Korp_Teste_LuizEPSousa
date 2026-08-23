package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"
)

type Invoice struct {
	ID               int64     `json:"id"`
	SequentialNumber int64     `json:"sequential_number"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type InvoiceItem struct {
	ID        int64     `json:"id"`
	InvoiceID int64     `json:"invoice_id"`
	ProductID int64     `json:"product_id"`
	Quantity  int       `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
}

type InvoiceWithItems struct {
	Invoice
	Items []InvoiceItem `json:"items"`
}

type InvoiceStore struct {
	db *sql.DB
}

func (s *InvoiceStore) Create(ctx context.Context, inv *Invoice, items []InvoiceItem) error {
	productIDs, required, err := invoiceRequirements(items)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, productID := range productIDs {
		var balance int64
		err := tx.QueryRowContext(ctx, `SELECT balance FROM products WHERE id = $1 FOR UPDATE`, productID).Scan(&balance)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		var reserved int64
		err = tx.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(ii.quantity), 0)
			FROM invoice_items ii
			JOIN invoices i ON i.id = ii.invoice_id
			WHERE ii.product_id = $1
			  AND i.status = 'Open'
			  AND NOT EXISTS (
				SELECT 1
				FROM stock_deductions sd
				WHERE sd.request_id = 'invoice:' || i.id::text
			  )
		`, productID).Scan(&reserved)
		if err != nil {
			return err
		}
		if required[productID] > balance-reserved {
			return ErrInsufficientStock
		}
	}

	queryInv := `
		INSERT INTO invoices (status)
		VALUES ('Open')
		RETURNING id, sequential_number, status, created_at, updated_at
	`
	if err := tx.QueryRowContext(ctx, queryInv).
		Scan(&inv.ID, &inv.SequentialNumber, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
		return err
	}

	queryItem := `
		INSERT INTO invoice_items (invoice_id, product_id, Quantity)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	for i := range items {
		items[i].InvoiceID = inv.ID
		if err := tx.QueryRowContext(ctx, queryItem, inv.ID, items[i].ProductID, items[i].Quantity).
			Scan(&items[i].ID, &items[i].CreatedAt); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func invoiceRequirements(items []InvoiceItem) ([]int64, map[int64]int64, error) {
	if len(items) == 0 {
		return nil, nil, ErrInvalidInvoice
	}

	required := make(map[int64]int64, len(items))
	const maxInt64 = int64(^uint64(0) >> 1)
	for _, item := range items {
		if item.ProductID <= 0 || item.Quantity <= 0 {
			return nil, nil, ErrInvalidInvoice
		}
		quantity := int64(item.Quantity)
		if quantity > maxInt64-required[item.ProductID] {
			return nil, nil, ErrInvalidInvoice
		}
		required[item.ProductID] += quantity
	}

	productIDs := make([]int64, 0, len(required))
	for productID := range required {
		productIDs = append(productIDs, productID)
	}
	sort.Slice(productIDs, func(i, j int) bool { return productIDs[i] < productIDs[j] })
	return productIDs, required, nil
}

func (s *InvoiceStore) GetAll(ctx context.Context) ([]Invoice, error) {
	query := `SELECT id, sequential_number, status, created_at, updated_at FROM invoices ORDER BY id DESC`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invoices []Invoice
	for rows.Next() {
		var inv Invoice
		if err := rows.Scan(&inv.ID, &inv.SequentialNumber, &inv.Status, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			return nil, err
		}
		invoices = append(invoices, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return invoices, nil
}

func (s *InvoiceStore) GetByID(ctx context.Context, id int64) (*InvoiceWithItems, error) {
	queryInv := `SELECT id, sequential_number, status, created_at, updated_at FROM invoices WHERE id = $1`
	var res InvoiceWithItems
	err := s.db.QueryRowContext(ctx, queryInv, id).
		Scan(&res.ID, &res.SequentialNumber, &res.Status, &res.CreatedAt, &res.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	queryItems := `SELECT id, invoice_id, product_id, quantity, created_at FROM invoice_items WHERE invoice_id = $1`
	rows, err := s.db.QueryContext(ctx, queryItems, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item InvoiceItem
		if err := rows.Scan(&item.ID, &item.InvoiceID, &item.ProductID, &item.Quantity, &item.CreatedAt); err != nil {
			return nil, err
		}
		res.Items = append(res.Items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &res, nil
}

func (s *InvoiceStore) UpdateStatus(ctx context.Context, id int64, status string) error {
	query := `UPDATE invoices SET status = $1, updated_at = NOW() WHERE id = $2 AND status = 'Open'`
	res, err := s.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrConflict
	}
	return nil
}

func (s *InvoiceStore) Delete(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var status string
	err = tx.QueryRowContext(ctx, `SELECT status FROM invoices WHERE id = $1 FOR UPDATE`, id).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if status != "Open" {
		return ErrConflict
	}

	var deducted bool
	requestID := fmt.Sprintf("invoice:%d", id)
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM stock_deductions WHERE request_id = $1)`, requestID).Scan(&deducted); err != nil {
		return err
	}
	if deducted {
		return ErrConflict
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM invoices WHERE id = $1 AND status = 'Open'`, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrConflict
	}
	return tx.Commit()
}
