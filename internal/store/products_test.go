package store

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
)

func newProductStoreMock(t *testing.T) (*ProductStore, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return &ProductStore{db: db}, mock
}

func TestCreateProductMapsUniqueViolationToConflict(t *testing.T) {
	store, mock := newProductStoreMock(t)
	mock.ExpectQuery("INSERT INTO products").
		WithArgs("SKU-1", "Product", 10).
		WillReturnError(&pq.Error{Code: "23505"})

	err := store.Create(context.Background(), &Product{Code: "SKU-1", Description: "Product", Balance: 10})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeductStockCommitsAllItemsAtomically(t *testing.T) {
	store, mock := newProductStoreMock(t)
	items := []StockDeduction{{ProductID: 1, Quantity: 2}, {ProductID: 2, Quantity: 3}}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO stock_deductions").WithArgs("invoice:1", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	for _, item := range items {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT balance FROM products WHERE id = $1 FOR UPDATE")).
			WithArgs(item.ProductID).
			WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(10))
		mock.ExpectExec("UPDATE products").WithArgs(item.Quantity, item.ProductID).WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()

	if err := store.DeductStock(context.Background(), "invoice:1", items); err != nil {
		t.Fatalf("DeductStock returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeductStockRollsBackPartialBatch(t *testing.T) {
	store, mock := newProductStoreMock(t)
	items := []StockDeduction{{ProductID: 1, Quantity: 2}, {ProductID: 2, Quantity: 3}}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO stock_deductions").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT balance").WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(10))
	mock.ExpectExec("UPDATE products").WithArgs(2, int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT balance").WithArgs(int64(2)).WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(1))
	mock.ExpectRollback()

	err := store.DeductStock(context.Background(), "invoice:1", items)
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeductStockReturnsSuccessForSameIdempotencyRequest(t *testing.T) {
	store, mock := newProductStoreMock(t)
	items := []StockDeduction{{ProductID: 1, Quantity: 2}}
	hash, err := deductionPayloadHash(items)
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO stock_deductions").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT payload_hash").WithArgs("invoice:1").WillReturnRows(sqlmock.NewRows([]string{"payload_hash"}).AddRow(hash))
	mock.ExpectRollback()

	if err := store.DeductStock(context.Background(), "invoice:1", items); err != nil {
		t.Fatalf("expected idempotent success, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeductStockRejectsReusedRequestWithDifferentItems(t *testing.T) {
	store, mock := newProductStoreMock(t)
	items := []StockDeduction{{ProductID: 1, Quantity: 2}}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO stock_deductions").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT payload_hash").WithArgs("invoice:1").WillReturnRows(sqlmock.NewRows([]string{"payload_hash"}).AddRow("different"))
	mock.ExpectRollback()

	err := store.DeductStock(context.Background(), "invoice:1", items)
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("expected ErrIdempotencyConflict, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeductStockValidatesBeforeStartingTransaction(t *testing.T) {
	store, mock := newProductStoreMock(t)
	tests := []struct {
		name      string
		requestID string
		items     []StockDeduction
	}{
		{name: "missing request id", items: []StockDeduction{{ProductID: 1, Quantity: 1}}},
		{name: "empty items", requestID: "invoice:1"},
		{name: "invalid product", requestID: "invoice:1", items: []StockDeduction{{ProductID: 0, Quantity: 1}}},
		{name: "negative quantity", requestID: "invoice:1", items: []StockDeduction{{ProductID: 1, Quantity: -1}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := store.DeductStock(context.Background(), tt.requestID, tt.items); !errors.Is(err, ErrInvalidDeduction) {
				t.Fatalf("expected ErrInvalidDeduction, got %v", err)
			}
		})
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeductStockDistinguishesMissingProduct(t *testing.T) {
	store, mock := newProductStoreMock(t)
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO stock_deductions").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT balance").WithArgs(int64(99)).WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	err := store.DeductStock(context.Background(), "invoice:1", []StockDeduction{{ProductID: 99, Quantity: 1}})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
