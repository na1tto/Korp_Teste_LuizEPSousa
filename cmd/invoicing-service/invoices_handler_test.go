package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/naitto/korperp-challenge/internal/store"
	"go.uber.org/zap"
)

type invoiceStoreStub struct {
	createErr error
	deleteErr error
}

func (s *invoiceStoreStub) Create(context.Context, *store.Invoice, []store.InvoiceItem) error {
	return s.createErr
}

func (*invoiceStoreStub) GetAll(context.Context) ([]store.Invoice, error) {
	return nil, nil
}

func (*invoiceStoreStub) GetByID(context.Context, int64) (*store.InvoiceWithItems, error) {
	return nil, store.ErrNotFound
}

func (*invoiceStoreStub) UpdateStatus(context.Context, int64, string) error {
	return nil
}

func (s *invoiceStoreStub) Delete(context.Context, int64) error {
	return s.deleteErr
}

func newInvoiceHandlerApp(invoiceStore *invoiceStoreStub) *application {
	return &application{
		store:  store.Storage{Invoices: invoiceStore},
		logger: zap.NewNop().Sugar(),
	}
}

func TestCreateInvoiceMapsReservationErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
	}{
		{name: "insufficient reservation", err: store.ErrInsufficientStock, status: http.StatusConflict},
		{name: "missing product", err: store.ErrNotFound, status: http.StatusUnprocessableEntity},
		{name: "invalid invoice", err: store.ErrInvalidInvoice, status: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newInvoiceHandlerApp(&invoiceStoreStub{createErr: tt.err})
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/v1/invoices/", strings.NewReader(`{"items":[{"product_id":1,"quantity":1}]}`))
			req.Header.Set("Content-Type", "application/json")
			app.mount().ServeHTTP(recorder, req)
			if recorder.Code != tt.status {
				t.Fatalf("expected %d, got %d: %s", tt.status, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestDeleteInvoiceHandler(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		err    error
		status int
	}{
		{name: "deleted", path: "/v1/invoices/7", status: http.StatusNoContent},
		{name: "not found", path: "/v1/invoices/7", err: store.ErrNotFound, status: http.StatusNotFound},
		{name: "closed or deducted", path: "/v1/invoices/7", err: store.ErrConflict, status: http.StatusConflict},
		{name: "invalid id", path: "/v1/invoices/invalid", err: errors.New("unused"), status: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newInvoiceHandlerApp(&invoiceStoreStub{deleteErr: tt.err})
			recorder := httptest.NewRecorder()
			app.mount().ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, tt.path, nil))
			if recorder.Code != tt.status {
				t.Fatalf("expected %d, got %d: %s", tt.status, recorder.Code, recorder.Body.String())
			}
			if tt.status == http.StatusNoContent && recorder.Body.Len() != 0 {
				t.Fatalf("204 response contains body %q", recorder.Body.String())
			}
		})
	}
}
