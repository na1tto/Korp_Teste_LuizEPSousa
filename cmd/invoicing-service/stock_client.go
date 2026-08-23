package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var (
	ErrStockServiceUnavailable = errors.New("stock service is temporarily unavailable")
	ErrInsufficientStock       = errors.New("insufficient stock balance for one or more items")
	ErrProductNotFound         = errors.New("one or more products no longer exist")
	ErrInvoiceStateConflict    = errors.New("invoice is not available for stock deduction")
)

type StockClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewStockclient(baseURL string) *StockClient {
	return &StockClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second, // resilience timeout
		},
	}
}

type DeductItemRequest struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

type DeductRequest struct {
	InvoiceID int64               `json:"invoice_id"`
	RequestID string              `json:"request_id"`
	Items     []DeductItemRequest `json:"items"`
}

func (c *StockClient) DeductStock(ctx context.Context, invoiceID int64, requestID string, items []DeductItemRequest) error {
	payload, err := json.Marshal(DeductRequest{InvoiceID: invoiceID, RequestID: requestID, Items: items})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/v1/products/deduct", c.baseURL), bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ErrStockServiceUnavailable
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return ErrInsufficientStock
	}
	if resp.StatusCode == http.StatusNotFound {
		return ErrProductNotFound
	}
	if resp.StatusCode == http.StatusUnprocessableEntity {
		return ErrInvoiceStateConflict
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected stock service error: status %d", resp.StatusCode)
	}

	return nil
}
