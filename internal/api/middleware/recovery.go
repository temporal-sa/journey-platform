package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/validated-pattern/journey-platform/internal/telemetry/logging"
)

// Recoverer recovers from panics, logs stack trace, and returns a 500 internal server error.
func Recoverer(logger any) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					stack := debug.Stack()
					errMsg := fmt.Sprintf("%v", rvr)

					logging.Debug().
						Str("error", errMsg).
						Str("stack", string(stack)).
						Str("request_id", GetRequestID(r.Context())).
						Msg("panic recovered")
					WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "an unexpected server error occurred")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
