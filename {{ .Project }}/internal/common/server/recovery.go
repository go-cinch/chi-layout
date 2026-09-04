package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

func Recoverer() func(http.Handler) http.Handler {
	logger := slog.Default()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writer := wrapResponseWriter(w, r.ProtoMajor)
			defer func() {
				if recovered := recover(); recovered != nil {
					if recovered == http.ErrAbortHandler {
						logger.WarnContext(r.Context(), "http request aborted")
						return
					}
					logger.ErrorContext(r.Context(), "panic recovered: "+fmt.Sprint(recovered), "stack", string(debug.Stack()))
					if !responseWritten(writer) {
						WriteError(writer, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
					}
				}
			}()
			next.ServeHTTP(writer, r)
		})
	}
}
