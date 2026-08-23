package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

const (
	messageInternalServer   = "the server encountered a problem"
	messageNotFound         = "not found"
	messageMethodNotAllowed = "method not allowed"
	messageRequestTimeout   = "request timed out"
)

var (
	errRouteNotFound          = errors.New("route not found")
	errProductFieldsRequired  = errors.New("code, description and balance (>=0) fields are required")
	errProductCodeConflict    = errors.New("a product with this code already exists")
	errDeductionFieldsMissing = errors.New("request_id and at least one item are required")
)

// handling errors internaly through application functions
// doing so prevents the user to have access to internal application
// informations, like the stacktree from the error

func (app *application) internalServerError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(r.Context().Err(), context.DeadlineExceeded) {
		app.requestTimeoutResponse(w, r, err)
		return
	}

	app.logger.Errorw("internal error", "method", r.Method, "path", r.URL.Path, "error", err.Error())
	app.writeErrorResponse(w, r, http.StatusInternalServerError, messageInternalServer)
}

func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Warnw("bad request", "method", r.Method, "path", r.URL.Path, "error", err.Error())

	app.writeErrorResponse(w, r, http.StatusBadRequest, err.Error())
}

func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Warnw("not found", "method", r.Method, "path", r.URL.Path, "error", err.Error())
	app.writeErrorResponse(w, r, http.StatusNotFound, messageNotFound)
}

func (app *application) conflictResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Warnw("conflict", "method", r.Method, "path", r.URL.Path, "error", err.Error())
	app.writeErrorResponse(w, r, http.StatusConflict, err.Error())
}

func (app *application) unprocessableEntityResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Warnw("unprocessable entity", "method", r.Method, "path", r.URL.Path, "error", err.Error())
	app.writeErrorResponse(w, r, http.StatusUnprocessableEntity, err.Error())
}

func (app *application) serviceUnavailableResponse(w http.ResponseWriter, r *http.Request, err error, message string) {
	app.logger.Errorw("service unavailable", "method", r.Method, "path", r.URL.Path, "error", err.Error())
	app.writeErrorResponse(w, r, http.StatusServiceUnavailable, message)
}

func (app *application) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	app.logger.Warnw("method not allowed", "method", r.Method, "path", r.URL.Path)
	app.writeErrorResponse(w, r, http.StatusMethodNotAllowed, messageMethodNotAllowed)
}

func (app *application) requestTimeoutResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Warnw("request timeout", "method", r.Method, "path", r.URL.Path, "error", err.Error())
	app.writeErrorResponse(w, r, http.StatusGatewayTimeout, messageRequestTimeout)
}

func (app *application) writeErrorResponse(w http.ResponseWriter, r *http.Request, status int, message string) {
	if err := writeJsonError(w, status, message); err != nil {
		app.logger.Errorw("failed to write error response", "method", r.Method, "path", r.URL.Path, "error", err.Error())
	}
}

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				app.internalServerError(w, r, fmt.Errorf("panic: %v", recovered))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
