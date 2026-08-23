package store

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func newInvoiceStoreMock(t *testing.T) (*InvoiceStore, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return &InvoiceStore{db: db}, mock
}

func expectProductReservation(mock sqlmock.Sqlmock, productID, balance, reserved int64) {
	mock.ExpectQuery("SELECT balance FROM products").WithArgs(productID).
		WillReturnRows(sqlmock.NewRows([]string{"balance"}).AddRow(balance))
	mock.ExpectQuery("SELECT COALESCE").WithArgs(productID).
		WillReturnRows(sqlmock.NewRows([]string{"reserved"}).AddRow(reserved))
}

func TestCreateInvoiceLocksProductsInOrderBeforeInsert(t *testing.T) {
	store, mock := newInvoiceStoreMock(t)
	items := []InvoiceItem{{ProductID: 2, Quantity: 1}, {ProductID: 1, Quantity: 2}}
	now := time.Now()

	mock.ExpectBegin()
	expectProductReservation(mock, 1, 10, 3)
	expectProductReservation(mock, 2, 10, 4)
	mock.ExpectQuery("INSERT INTO invoices").WillReturnRows(
		sqlmock.NewRows([]string{"id", "sequential_number", "status", "created_at", "updated_at"}).
			AddRow(7, 100, "Open", now, now),
	)
	mock.ExpectQuery("INSERT INTO invoice_items").WithArgs(int64(7), int64(2), 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(11, now))
	mock.ExpectQuery("INSERT INTO invoice_items").WithArgs(int64(7), int64(1), 2).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(12, now))
	mock.ExpectCommit()

	invoice := &Invoice{}
	if err := store.Create(context.Background(), invoice, items); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if invoice.ID != 7 || items[0].InvoiceID != 7 || items[1].InvoiceID != 7 {
		t.Fatalf("created values were not populated: invoice=%+v items=%+v", invoice, items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateInvoiceAggregatesReservationsForDuplicateProducts(t *testing.T) {
	store, mock := newInvoiceStoreMock(t)
	items := []InvoiceItem{{ProductID: 1, Quantity: 1}, {ProductID: 1, Quantity: 1}}

	mock.ExpectBegin()
	expectProductReservation(mock, 1, 2, 1)
	mock.ExpectRollback()

	err := store.Create(context.Background(), &Invoice{}, items)
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateInvoiceReturnsNotFoundBeforeInsert(t *testing.T) {
	store, mock := newInvoiceStoreMock(t)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT balance FROM products").WithArgs(int64(99)).WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	err := store.Create(context.Background(), &Invoice{}, []InvoiceItem{{ProductID: 99, Quantity: 1}})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteInvoice(t *testing.T) {
	tests := []struct {
		name         string
		status       string
		deducted     bool
		missing      bool
		expected     error
		shouldDelete bool
	}{
		{name: "open invoice", status: "Open", shouldDelete: true},
		{name: "closed invoice", status: "Closed", expected: ErrConflict},
		{name: "deducted invoice", status: "Open", deducted: true, expected: ErrConflict},
		{name: "missing invoice", missing: true, expected: ErrNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, mock := newInvoiceStoreMock(t)
			mock.ExpectBegin()
			statusQuery := mock.ExpectQuery("SELECT status FROM invoices").WithArgs(int64(7))
			if tt.missing {
				statusQuery.WillReturnError(sql.ErrNoRows)
			} else {
				statusQuery.WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow(tt.status))
			}
			if tt.status == "Open" {
				mock.ExpectQuery("SELECT EXISTS").WithArgs("invoice:7").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(tt.deducted))
			}
			if tt.shouldDelete {
				mock.ExpectExec("DELETE FROM invoices").WithArgs(int64(7)).WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}

			err := store.Delete(context.Background(), 7)
			if !errors.Is(err, tt.expected) {
				t.Fatalf("expected %v, got %v", tt.expected, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
