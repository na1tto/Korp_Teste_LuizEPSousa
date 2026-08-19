package store

import (
	"context"
	"database/sql"
	"errors"
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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

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

	return &res, nil
}

func (s *InvoiceStore) UpdateStatus(ctx context.Context, id int64, status string) error {
	query := `UPDATE invoices SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, status, id)
	return err
}
