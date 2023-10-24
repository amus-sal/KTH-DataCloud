package httpencoder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amus-sal/kth-datacloud-unzipper/domain"
)

func TestRespond(t *testing.T) {
	tests := []struct {
		name                string
		code                int
		payload             interface{}
		expectedBody        string
		expectedContentType string
		expectedCode        int
	}{
		{
			name: "successful encoding",
			code: http.StatusOK,
			payload: struct {
				SomeField string `json:"some_field"`
			}{"hello"},
			expectedBody:        `{"some_field":"hello"}`,
			expectedContentType: "application/json; charset=utf-8",
			expectedCode:        http.StatusOK,
		},
		{
			name:                "successful encoding empty payload",
			code:                http.StatusOK,
			payload:             nil,
			expectedBody:        "",
			expectedContentType: "",
			expectedCode:        http.StatusOK,
		},
		{
			name:                "unsuccessful encoding invalid payload",
			code:                http.StatusOK,
			payload:             func() {},
			expectedBody:        "",
			expectedContentType: "",
			expectedCode:        http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder()
			w := httptest.NewRecorder()

			encoder.Respond(context.TODO(), w, tt.payload, tt.code)

			if got, want := w.Header().Get("Content-Type"), tt.expectedContentType; got != want {
				t.Errorf(`expected content type to be = %s, got = %s`, want, got)
			}

			b, err := io.ReadAll(w.Body)
			if err != nil {
				t.Errorf("reading response body error = %q, want = %q", err, tt.expectedBody)
			}

			// Trim the newline added by the json encoder.
			decoded := strings.TrimSuffix(string(b), "\n")
			if decoded != tt.expectedBody {
				t.Errorf("expected body to be = %q, got = %q", tt.expectedBody, decoded)
			}
		})
	}
}

func TestError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode int
	}{
		{
			name:         "bad request",
			err:          fmt.Errorf("something went wrong %w", domain.ErrBadRequest),
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "internal server error",
			err:          fmt.Errorf("something went wrong %w", domain.ErrInternal),
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:         "default to internal server error",
			err:          errors.New("no domain error wrapped"),
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoder := NewEncoder()
			w := httptest.NewRecorder()

			encoder.Error(context.TODO(), w, tt.err)

			if w.Code != tt.expectedCode {
				t.Errorf("expected status code to be = %d, got = %d", tt.expectedCode, w.Code)
			}
		})
	}
}
