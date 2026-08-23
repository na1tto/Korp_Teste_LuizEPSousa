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

func expectOpenInvoice(mock sqlmock.Sqlmock, invoiceID int64) {
	mock.ExpectQuery("SELECT status FROM invoices").WithArgs(invoiceID).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("Open"))
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
	expectOpenInvoice(mock, 1)
	mock.ExpectExec("INSERT INTO stock_deductions").WithArgs("invoice:1", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	for _, item := range items {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT balance FROM products WHERE id = $1 FOR UPDATE")).
			WithArgs(item.ProductID).
			WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(10))
		mock.ExpectExec("UPDATE products").WithArgs(item.Quantity, item.ProductID).WillReturnResult(sqlmock.NewResult(0, 1))
	}
	mock.ExpectCommit()

	if err := store.DeductStock(context.Background(), 1, "invoice:1", items); err != nil {
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
	expectOpenInvoice(mock, 1)
	mock.ExpectExec("INSERT INTO stock_deductions").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT balance").WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(10))
	mock.ExpectExec("UPDATE products").WithArgs(2, int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT balance").WithArgs(int64(2)).WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(1))
	mock.ExpectRollback()

	err := store.DeductStock(context.Background(), 1, "invoice:1", items)
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
	expectOpenInvoice(mock, 1)
	mock.ExpectExec("INSERT INTO stock_deductions").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT payload_hash").WithArgs("invoice:1").WillReturnRows(sqlmock.NewRows([]string{"payload_hash"}).AddRow(hash))
	mock.ExpectRollback()

	if err := store.DeductStock(context.Background(), 1, "invoice:1", items); err != nil {
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
	expectOpenInvoice(mock, 1)
	mock.ExpectExec("INSERT INTO stock_deductions").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT payload_hash").WithArgs("invoice:1").WillReturnRows(sqlmock.NewRows([]string{"payload_hash"}).AddRow("different"))
	mock.ExpectRollback()

	err := store.DeductStock(context.Background(), 1, "invoice:1", items)
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
		invoiceID int64
		requestID string
		items     []StockDeduction
	}{
		{name: "missing invoice id", requestID: "invoice:1", items: []StockDeduction{{ProductID: 1, Quantity: 1}}},
		{name: "missing request id", invoiceID: 1, items: []StockDeduction{{ProductID: 1, Quantity: 1}}},
		{name: "mismatched request id", invoiceID: 2, requestID: "invoice:1", items: []StockDeduction{{ProductID: 1, Quantity: 1}}},
		{name: "empty items", invoiceID: 1, requestID: "invoice:1"},
		{name: "invalid product", invoiceID: 1, requestID: "invoice:1", items: []StockDeduction{{ProductID: 0, Quantity: 1}}},
		{name: "negative quantity", invoiceID: 1, requestID: "invoice:1", items: []StockDeduction{{ProductID: 1, Quantity: -1}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := store.DeductStock(context.Background(), tt.invoiceID, tt.requestID, tt.items); !errors.Is(err, ErrInvalidDeduction) {
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
	expectOpenInvoice(mock, 1)
	mock.ExpectExec("INSERT INTO stock_deductions").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT balance").WithArgs(int64(99)).WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	err := store.DeductStock(context.Background(), 1, "invoice:1", []StockDeduction{{ProductID: 99, Quantity: 1}})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeductStockRejectsUnavailableInvoice(t *testing.T) {
	tests := []struct {
		name   string
		row    *sqlmock.Rows
		rowErr error
	}{
		{name: "closed", row: sqlmock.NewRows([]string{"status"}).AddRow("Closed")},
		{name: "deleted", rowErr: sql.ErrNoRows},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, mock := newProductStoreMock(t)
			mock.ExpectBegin()
			query := mock.ExpectQuery("SELECT status FROM invoices").WithArgs(int64(1))
			if tt.rowErr != nil {
				query.WillReturnError(tt.rowErr)
			} else {
				query.WillReturnRows(tt.row)
			}
			mock.ExpectRollback()

			err := store.DeductStock(context.Background(), 1, "invoice:1", []StockDeduction{{ProductID: 1, Quantity: 1}})
			if !errors.Is(err, ErrInvoiceStateConflict) {
				t.Fatalf("expected ErrInvoiceStateConflict, got %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
