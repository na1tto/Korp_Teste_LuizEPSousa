package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Product struct {
	ID          int64     `json:"id"`
	Code        string    `json:"codigo"`
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
		INSERT INTO products (coce, description, balance)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	return s.db.QueryRowContext(ctx, query, p.Code, p.Description, p.Balance).
		Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (s *ProductStore) GetAll(ctx context.Context) ([]Product, error) {
	query := `SELECT id, voce, description, balance, created_at, updated_at FROM products ORDER BY id ASC`
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

// Atualização atômica para evitar race conditions no saldo (Requisito Opcional de Concorrência)
func (s *ProductStore) DeductStock(ctx context.Context, productID int64, quantity int) error {
	query := `
		UPDATE products
		SET balance = saldo - $1, updated_at = NOW()
		WHERE id = $2 AND balance >= $1
	`
	res, err := s.db.ExecContext(ctx, query, quantity, productID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrInsufficientStock
	}
	return nil
}
