package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestMountedRouterUsesJsonErrors(t *testing.T) {
	app := &application{logger: zap.NewNop().Sugar()}
	tests := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{name: "not found", method: http.MethodGet, path: "/missing", status: http.StatusNotFound},
		{name: "method not allowed", method: http.MethodPost, path: "/v1/health", status: http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			app.mount().ServeHTTP(recorder, httptest.NewRequest(tt.method, tt.path, nil))
			if recorder.Code != tt.status || !strings.Contains(recorder.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("status=%d content-type=%q body=%q", recorder.Code, recorder.Header().Get("Content-Type"), recorder.Body.String())
			}
		})
	}
}

func TestRecoverPanicUsesInternalErrorEnvelope(t *testing.T) {
	app := &application{logger: zap.NewNop().Sugar()}
	handler := app.recoverPanic(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/panic", nil))

	if recorder.Code != http.StatusInternalServerError || recorder.Body.String() != "{\"error\":\"the server encountered a problem\"}\n" {
		t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
	}
}

func TestInternalServerErrorMapsDeadlineToGatewayTimeout(t *testing.T) {
	app := &application{logger: zap.NewNop().Sugar()}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	recorder := httptest.NewRecorder()

	app.internalServerError(recorder, req, context.DeadlineExceeded)
	if recorder.Code != http.StatusGatewayTimeout {
		t.Fatalf("expected 504, got %d", recorder.Code)
	}
}

func TestCataloguedErrorResponses(t *testing.T) {
	app := &application{logger: zap.NewNop().Sugar()}
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	tests := []struct {
		name   string
		status int
		call   func(http.ResponseWriter)
	}{
		{name: "bad request", status: http.StatusBadRequest, call: func(w http.ResponseWriter) { app.badRequestResponse(w, req, errors.New("bad input")) }},
		{name: "conflict", status: http.StatusConflict, call: func(w http.ResponseWriter) { app.conflictResponse(w, req, errors.New("conflict")) }},
		{name: "unprocessable", status: http.StatusUnprocessableEntity, call: func(w http.ResponseWriter) { app.unprocessableEntityResponse(w, req, errors.New("invalid entity")) }},
		{name: "unavailable", status: http.StatusServiceUnavailable, call: func(w http.ResponseWriter) {
			app.serviceUnavailableResponse(w, req, errors.New("offline"), "dependency unavailable")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			tt.call(recorder)
			if recorder.Code != tt.status || !strings.Contains(recorder.Body.String(), `"error"`) {
				t.Fatalf("status=%d body=%q", recorder.Code, recorder.Body.String())
			}
		})
	}
}
