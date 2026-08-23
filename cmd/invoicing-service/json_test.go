package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadJsonValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "empty", body: "", want: "must not be empty"},
		{name: "malformed", body: `{"name":`, want: "malformed JSON"},
		{name: "wrong type", body: `{"count":"two"}`, want: `invalid value for field "count"`},
		{name: "unknown field", body: `{"extra":true}`, want: `unknown field "extra"`},
		{name: "multiple values", body: `{"count":1}{"count":2}`, want: "single JSON value"},
	}
	type input struct {
		Count int `json:"count"`
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			err := readJson(httptest.NewRecorder(), req, &input{})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestWriteJsonDoesNotCommitHeadersWhenMarshalFails(t *testing.T) {
	recorder := httptest.NewRecorder()
	err := writeJson(recorder, http.StatusOK, make(chan int))
	if err == nil {
		t.Fatal("expected marshal error")
	}
	if recorder.Code != http.StatusOK || recorder.Header().Get("Content-Type") != "" || recorder.Body.Len() != 0 {
		t.Fatalf("response was committed after marshal failure: code=%d body=%q", recorder.Code, recorder.Body.String())
	}
}
