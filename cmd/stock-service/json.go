package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

var Validate *validator.Validate

func init() {
	Validate = validator.New(validator.WithRequiredStructEnabled())
}

func writeJson(w http.ResponseWriter, status int, data any) error {
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return nil
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(payload)
	return err
}

func readJson(w http.ResponseWriter, r *http.Request, data any) error {
	maxBytes := 1_048_578 // maximum of ONE megabyte for the request
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(data); err != nil {
		var syntaxError *json.SyntaxError
		var typeError *json.UnmarshalTypeError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.Is(err, io.EOF):
			return errors.New("request body must not be empty")
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("request body contains malformed JSON")
		case errors.As(err, &syntaxError):
			return fmt.Errorf("request body contains malformed JSON at position %d", syntaxError.Offset)
		case errors.As(err, &typeError):
			if typeError.Field != "" {
				return fmt.Errorf("request body contains an invalid value for field %q", typeError.Field)
			}
			return errors.New("request body contains an invalid JSON value")
		case errors.As(err, &maxBytesError):
			return fmt.Errorf("request body must not exceed %d bytes", maxBytesError.Limit)
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			return fmt.Errorf("request body contains unknown field %s", strings.TrimPrefix(err.Error(), "json: unknown field "))
		default:
			return errors.New("request body contains invalid JSON")
		}
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON value")
	}

	return nil
}

// we want to return our errors in the json format as well
func writeJsonError(w http.ResponseWriter, status int, message string) error {
	type envelope struct {
		Error string `json:"error"`
	}

	return writeJson(w, status, &envelope{Error: message})
}

// this is a standarized way of returning json responses across our application
// this abstracts the writeJson method by allowing it to return any type of data
// in this way all of the data in the response will be inside a "data" value
// we did the same thing at the errors.go package for standarazing error responses
func (app *application) jsonResponse(w http.ResponseWriter, status int, data any) error {
	type envelope struct {
		Data any `json:"data"`
	}

	return writeJson(w, status, &envelope{Data: data})
}
