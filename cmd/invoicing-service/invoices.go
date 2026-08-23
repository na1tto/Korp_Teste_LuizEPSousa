package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/naitto/korperp-challenge/internal/store"
)

type CreateInvoiceItemPayload struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

type CreateInvoicePayload struct {
	Items []CreateInvoiceItemPayload `json:"items"`
}

func (app *application) createInvoiceHandler(w http.ResponseWriter, r *http.Request) {
	var payload CreateInvoicePayload
	if err := readJson(w, r, &payload); err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if len(payload.Items) == 0 {
		app.badRequestResponse(w, r, errInvoiceItemsEmpty)
		return
	}

	items := make([]store.InvoiceItem, len(payload.Items))
	for i, item := range payload.Items {
		if item.ProductID <= 0 || item.Quantity <= 0 {
			app.badRequestResponse(w, r, errInvoiceItemInvalid)
			return
		}
		items[i] = store.InvoiceItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}

	invoice := &store.Invoice{}
	ctx := r.Context()
	if err := app.store.Invoices.Create(ctx, invoice, items); err != nil {
		switch {
		case errors.Is(err, store.ErrInvalidInvoice):
			app.badRequestResponse(w, r, err)
		case errors.Is(err, store.ErrNotFound):
			app.unprocessableEntityResponse(w, r, errInvoiceProductGone)
		case errors.Is(err, store.ErrInsufficientStock):
			app.conflictResponse(w, r, errInvoiceCreationStock)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}

	res := store.InvoiceWithItems{
		Invoice: *invoice,
		Items:   items,
	}

	if err := app.jsonResponse(w, http.StatusCreated, res); err != nil {
		app.internalServerError(w, r, err)
	}
}

func (app *application) deleteInvoiceHandler(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		app.badRequestResponse(w, r, errInvalidInvoiceID)
		return
	}

	err = app.store.Invoices.Delete(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			app.notFoundResponse(w, r, err)
		case errors.Is(err, store.ErrConflict):
			app.conflictResponse(w, r, errInvoiceDeleteConflict)
		default:
			app.internalServerError(w, r, err)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (app *application) listInvoicesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	invoices, err := app.store.Invoices.GetAll(ctx)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	if invoices == nil {
		invoices = []store.Invoice{}
	}

	if err := app.jsonResponse(w, http.StatusOK, invoices); err != nil {
		app.internalServerError(w, r, err)
	}
}

func (app *application) getInvoiceHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		app.badRequestResponse(w, r, errInvalidInvoiceID)
		return
	}

	ctx := r.Context()
	inv, err := app.store.Invoices.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			app.notFoundResponse(w, r, err)
			return
		}
		app.internalServerError(w, r, err)
		return
	}

	if err := app.jsonResponse(w, http.StatusOK, inv); err != nil {
		app.internalServerError(w, r, err)
	}
}

// Print Handler: validates status, debits stock, and closes the invoice.
func (app *application) printInvoiceHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		app.badRequestResponse(w, r, errInvalidInvoiceID)
		return
	}

	ctx := r.Context()
	inv, err := app.store.Invoices.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			app.notFoundResponse(w, r, err)
			return
		}
		app.internalServerError(w, r, err)
		return
	}

	// 1: Do not allow printing of notes that are not open.
	if inv.Status != "Open" {
		app.badRequestResponse(w, r, errInvoiceNotOpen)
		return
	}

	// Create a stock-service withdrawal request.
	var deductItems []DeductItemRequest
	for _, item := range inv.Items {
		deductItems = append(deductItems, DeductItemRequest{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	// 2: Performs inventory write-offs with error handling.
	requestID := fmt.Sprintf("invoice:%d", inv.ID)
	if err := app.stockClient.DeductStock(ctx, inv.ID, requestID, deductItems); err != nil {
		if errors.Is(err, ErrInsufficientStock) {
			app.conflictResponse(w, r, errInvoiceStock)
			return
		}
		if errors.Is(err, ErrStockServiceUnavailable) {
			app.serviceUnavailableResponse(w, r, err, messageStockUnavailable)
			return
		}
		if errors.Is(err, ErrProductNotFound) {
			app.unprocessableEntityResponse(w, r, errInvoiceProductGone)
			return
		}
		if errors.Is(err, ErrInvoiceStateConflict) {
			app.conflictResponse(w, r, errInvoiceStateChanged)
			return
		}
		app.internalServerError(w, r, err)
		return
	}

	// 3: Update invoice status to 'Closed'
	if err := app.store.Invoices.UpdateStatus(ctx, id, "Closed"); err != nil {
		if errors.Is(err, store.ErrConflict) {
			app.conflictResponse(w, r, errInvoiceStateChanged)
			return
		}
		app.internalServerError(w, r, err)
		return
	}

	inv.Status = "Closed"
	if err := app.jsonResponse(w, http.StatusOK, inv); err != nil {
		app.internalServerError(w, r, err)
	}
}
