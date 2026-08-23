package main

import (
	"errors"
	"net/http"

	"github.com/naitto/korperp-challenge/internal/store"
)

type CreateProductPayload struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	Balance     int    `json:"balance"`
}

func (app *application) createProductHandler(w http.ResponseWriter, r *http.Request) {
	var payload CreateProductPayload
	if err := readJson(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if payload.Code == "" || payload.Description == "" || payload.Balance < 0 {
		app.badRequestResponse(w, r, errProductFieldsRequired)
		return
	}

	product := &store.Product{
		Code:        payload.Code,
		Description: payload.Description,
		Balance:     payload.Balance,
	}

	ctx := r.Context()
	if err := app.store.Products.Create(ctx, product); err != nil {
		if errors.Is(err, store.ErrConflict) {
			app.conflictResponse(w, r, errProductCodeConflict)
			return
		}
		app.internalServerError(w, r, err)
		return
	}

	if err := app.jsonResponse(w, http.StatusCreated, product); err != nil {
		app.internalServerError(w, r, err)
	}
}

type DeductItemPayload struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

func (app *application) listProductsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	products, err := app.store.Products.GetAll(ctx)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if products == nil {
		products = []store.Product{}
	}

	if err := app.jsonResponse(w, http.StatusOK, products); err != nil {
		app.internalServerError(w, r, err)
		return
	}
}

type DeductStockPayload struct {
	RequestID string              `json:"request_id"`
	Items     []DeductItemPayload `json:"items"`
}

// Endpoint consumed internally by the Invoicing Service during printing.
func (app *application) deductStockHandler(w http.ResponseWriter, r *http.Request) {
	var payload DeductStockPayload
	if err := readJson(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if payload.RequestID == "" || len(payload.Items) == 0 {
		app.badRequestResponse(w, r, errDeductionFieldsMissing)
		return
	}

	items := make([]store.StockDeduction, len(payload.Items))
	for i, item := range payload.Items {
		items[i] = store.StockDeduction{ProductID: item.ProductID, Quantity: item.Quantity}
	}

	ctx := r.Context()
	if err := app.store.Products.DeductStock(ctx, payload.RequestID, items); err != nil {
		switch {
		case errors.Is(err, store.ErrInvalidDeduction):
			app.badRequestResponse(w, r, err)
		case errors.Is(err, store.ErrNotFound):
			app.notFoundResponse(w, r, err)
		case errors.Is(err, store.ErrInsufficientStock):
			app.conflictResponse(w, r, err)
		case errors.Is(err, store.ErrIdempotencyConflict):
			app.unprocessableEntityResponse(w, r, err)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	if err := app.jsonResponse(w, http.StatusOK, map[string]string{"message": "stock successfully deducted"}); err != nil {
		app.internalServerError(w, r, err)
	}
}
