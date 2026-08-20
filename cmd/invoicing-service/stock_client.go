package main

import (
	"errors"
	"net/http"
)

var (
	ErrStockServiceUnavailable = errors.New("stock service is temporarily unavailable")
	ErrInsufficientStock       = errors.New("insufficient stock balance for one or more items")
)

type StockClient struct {
	baseURL    string
	httpClient *http.Client
}
