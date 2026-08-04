package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recoverer recovers from panics, logs stack trace, and returns a 500 internal server error.
func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					stack := debug.Stack()
					errMsg := fmt.Sprintf("%v", rvr)

					if logger != nil {
						logger.ErrorContext(r.Context(), "panic recovered",
							slog.String("error", errMsg),
							slog.String("stack", string(stack)),
							slog.String("request_id", GetRequestID(r.Context())),
						)
					}

					WriteError(w, r, http.StatusInternalServerError, "Internal Server Error", "an unexpected server error occurred")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
