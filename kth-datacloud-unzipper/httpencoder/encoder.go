package httpencoder

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/amus-sal/kth-datacloud-unzipper/domain"
)

// NewEncoder creates an instance of ResponseEncoder.
func NewEncoder() ResponseEncoder {
	return ResponseEncoder{}
}

// ResponseEncoder handles serialization over HTTP responses.
type ResponseEncoder struct{}

// ErrorResponse represents an error response sent over HTTP.
type errorResponse struct {
	Message string `json:"message"`
}

// Error utilizes the error parameter to write it over HTTP and set the corresponding status code.
func (e ResponseEncoder) Error(ctx context.Context, w http.ResponseWriter, err error) {
	resp := errorResponse{
		Message: err.Error(),
	}

	statusCode := http.StatusInternalServerError
	switch errors.Unwrap(err) {
	case domain.ErrBadRequest:
		statusCode = http.StatusBadRequest
	}

	e.Respond(ctx, w, resp, statusCode)
}

// Respond will respond with an encoded payload if it exists otherwise just with the given status code.
func (e ResponseEncoder) Respond(ctx context.Context, w http.ResponseWriter, payload interface{}, statusCode int) {
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-type", "application/json; charset=utf-8")

		// Write the header before we write the content, otherwise it won't work.
		w.WriteHeader(statusCode)

		// Write the json encoded payload.
		if _, err := w.Write(encoded); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		// If there's no payload just write the header.
		w.WriteHeader(statusCode)
	}
}
