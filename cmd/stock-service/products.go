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
		app.badRequestResponse(w, r, errors.New("code, description and balance (>=0) fields are required"))
		return
	}

	product := &store.Product{
		Code:        payload.Code,
		Description: payload.Description,
		Balance:     payload.Balance,
	}

	ctx := r.Context()
	if err := app.store.Products.Create(ctx, product); err != nil {
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
	Items []DeductItemPayload `json:"items"`
}

// Endpoint consumido internamente pelo Invoicing Service durante a impressão
func (app *application) deductStockHandler(w http.ResponseWriter, r *http.Request) {
	var payload DeductStockPayload
	if err := readJson(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if len(payload.Items) == 0 {
		app.badRequestResponse(w, r, errors.New("the list cannot be empty"))
		return
	}

	ctx := r.Context()
	for _, item := range payload.Items {
		if err := app.store.Products.DeductStock(ctx, item.ProductID, item.Quantity); err != nil {
			if errors.Is(err, store.ErrInsufficientStock) {
				app.conflictResponse(w, r, err)
				return
			}
			app.internalServerError(w, r, err)
			return
		}
	}

	if err := app.jsonResponse(w, http.StatusOK, map[string]string{"message": "stock successfully deducted"}); err != nil {
		app.internalServerError(w, r, err)
	}
}
