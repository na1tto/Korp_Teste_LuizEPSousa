package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStockClientDeductStockMapsResponses(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		expected error
	}{
		{name: "success", status: http.StatusOK},
		{name: "insufficient stock", status: http.StatusConflict, expected: ErrInsufficientStock},
		{name: "missing product", status: http.StatusNotFound, expected: ErrProductNotFound},
		{name: "invoice state conflict", status: http.StatusUnprocessableEntity, expected: ErrInvoiceStateConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var payload DeductRequest
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Errorf("decode request: %v", err)
				}
				if payload.RequestID != "invoice:42" {
					t.Errorf("request_id = %q", payload.RequestID)
				}
				if payload.InvoiceID != 42 {
					t.Errorf("invoice_id = %d", payload.InvoiceID)
				}
				w.WriteHeader(tt.status)
			}))
			defer server.Close()

			client := NewStockclient(server.URL)
			err := client.DeductStock(context.Background(), 42, "invoice:42", []DeductItemRequest{{ProductID: 1, Quantity: 2}})
			if !errors.Is(err, tt.expected) {
				t.Fatalf("expected %v, got %v", tt.expected, err)
			}
		})
	}
}

type failingRoundTripper struct{}

func (failingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, io.ErrUnexpectedEOF
}

func TestStockClientDeductStockMapsCommunicationFailure(t *testing.T) {
	client := NewStockclient("http://stock.invalid")
	client.httpClient.Transport = failingRoundTripper{}

	err := client.DeductStock(context.Background(), 42, "invoice:42", []DeductItemRequest{{ProductID: 1, Quantity: 2}})
	if !errors.Is(err, ErrStockServiceUnavailable) {
		t.Fatalf("expected ErrStockServiceUnavailable, got %v", err)
	}
}
