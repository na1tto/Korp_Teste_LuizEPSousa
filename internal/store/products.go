package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/lib/pq"
)

type Product struct {
	ID          int64     `json:"id"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	Balance     int       `json:"balance"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ProductStore struct {
	db *sql.DB
}

func (s *ProductStore) Create(ctx context.Context, p *Product) error {
	query := `
		INSERT INTO products (code, description, balance)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	err := s.db.QueryRowContext(ctx, query, p.Code, p.Description, p.Balance).
		Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return ErrConflict
		}
	}
	return err
}

func (s *ProductStore) GetAll(ctx context.Context) ([]Product, error) {
	query := `SELECT id, code, description, balance, created_at, updated_at FROM products ORDER BY id ASC`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Code, &p.Description, &p.Balance, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return products, nil
}

func (s *ProductStore) GetByID(ctx context.Context, id int64) (*Product, error) {
	query := `SELECT id, code, description, balance, created_at, updated_at FROM products WHERE id = $1`
	var p Product
	err := s.db.QueryRowContext(ctx, query, id).Scan(&p.ID, &p.Code, &p.Description, &p.Balance, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (s *ProductStore) DeductStock(ctx context.Context, invoiceID int64, requestID string, items []StockDeduction) error {
	if invoiceID <= 0 || requestID != fmt.Sprintf("invoice:%d", invoiceID) || len(items) == 0 {
		return ErrInvalidDeduction
	}
	for _, item := range items {
		if item.ProductID <= 0 || item.Quantity <= 0 {
			return ErrInvalidDeduction
		}
	}

	payloadHash, err := deductionPayloadHash(items)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var invoiceStatus string
	err = tx.QueryRowContext(ctx, `SELECT status FROM invoices WHERE id = $1 FOR UPDATE`, invoiceID).Scan(&invoiceStatus)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && invoiceStatus != "Open") {
		return ErrInvoiceStateConflict
	}
	if err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, `
		INSERT INTO stock_deductions (request_id, payload_hash)
		VALUES ($1, $2)
		ON CONFLICT (request_id) DO NOTHING
	`, requestID, payloadHash)
	if err != nil {
		return err
	}
	inserted, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if inserted == 0 {
		var storedHash string
		if err := tx.QueryRowContext(ctx, `SELECT payload_hash FROM stock_deductions WHERE request_id = $1`, requestID).Scan(&storedHash); err != nil {
			return err
		}
		if storedHash != payloadHash {
			return ErrIdempotencyConflict
		}
		return nil
	}

	for _, item := range orderedDeductions(items) {
		var balance int
		err := tx.QueryRowContext(ctx, `SELECT balance FROM products WHERE id = $1 FOR UPDATE`, item.ProductID).Scan(&balance)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if balance < item.Quantity {
			return ErrInsufficientStock
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE products
			SET balance = balance - $1, updated_at = NOW()
			WHERE id = $2
		`, item.Quantity, item.ProductID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func orderedDeductions(items []StockDeduction) []StockDeduction {
	ordered := append([]StockDeduction(nil), items...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].ProductID < ordered[j].ProductID
	})
	return ordered
}

func deductionPayloadHash(items []StockDeduction) (string, error) {
	payload, err := json.Marshal(items)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(payload)), nil
}
