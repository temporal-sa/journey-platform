package middleware

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// ErrorDetail represents detailed information about an error.
type ErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
	Details   any    `json:"details,omitempty"`
}

// ErrorResponse is the standardized JSON error payload returned by APIs.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// WriteError sends a standardized JSON error response.
func WriteError(w http.ResponseWriter, r *http.Request, statusCode int, message string, details ...any) {
	code := http.StatusText(statusCode)
	if code == "" {
		code = "UNKNOWN_ERROR"
	}

	reqID := GetRequestID(r.Context())
	traceID := GetTraceID(r.Context())

	resp := ErrorResponse{
		Error: ErrorDetail{
			Code:      code,
			Message:   message,
			RequestID: reqID,
			TraceID:   traceID,
		},
	}

	if len(details) > 0 && details[0] != nil {
		resp.Error.Details = details[0]
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(resp)
}

// WriteJSON sends a standardized JSON success response.
func WriteJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

// DecodeJSON decodes the JSON request body and handles malformed JSON and body size limit errors.
func DecodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	if r.Body == nil {
		err := errors.New("empty request body")
		WriteError(w, r, http.StatusBadRequest, "malformed JSON payload", err.Error())
		return err
	}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) || err.Error() == "http: request body too large" {
			WriteError(w, r, http.StatusRequestEntityTooLarge, "request body exceeds size limit")
			return err
		}
		if errors.Is(err, io.EOF) {
			WriteError(w, r, http.StatusBadRequest, "malformed JSON payload", "empty payload")
			return err
		}
		WriteError(w, r, http.StatusBadRequest, "malformed JSON payload", err.Error())
		return err
	}
	return nil
}
