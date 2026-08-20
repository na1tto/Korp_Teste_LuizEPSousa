package store

import (
	"context"
	"database/sql"
	"errors"
)

var (
	ErrNotFound          = errors.New("record not found")
	ErrConflict          = errors.New("record already exists")
	ErrInsufficientStock = errors.New("insufficient stock for the product.")
)

type Storage struct {
	Products interface {
		Create(ctx context.Context, p *Product) error
		GetAll(ctx context.Context) ([]Product, error)
		GetByID(ctx context.Context, id int64) (*Product, error)
		DeductStock(ctx context.Context, productID int64, quantity int) error
	}
	Invoices interface {
		Create(ctx context.Context, inv *Invoice, items []InvoiceItem) error
		GetAll(ctx context.Context) ([]Invoice, error)
		GetByID(ctx context.Context, id int64) (*InvoiceWithItems, error)
		UpdateStatus(ctx context.Context, id int64, status string) error
	}
}

func NewStorage(db *sql.DB) Storage {
	return Storage{
		Products: &ProductStore{db: db},
		Invoices: &InvoiceStore{db: db},
	}
}
