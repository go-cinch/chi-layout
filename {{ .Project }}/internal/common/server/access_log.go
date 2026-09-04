package server

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"{{ .Computed.module_name_final }}/internal/common/redact"

	"github.com/go-chi/chi/v5"
)

func AccessLog(policy redact.Policy) func(http.Handler) http.Handler {
	logger := slog.Default()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			writer := wrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(writer, r)

			status := writer.Status()
			if status == 0 {
				status = http.StatusOK
			}
			route := chi.RouteContext(r.Context()).RoutePattern()
			logger.InfoContext(r.Context(), "http request",
				"method", r.Method,
				"path", redactedPath(r, policy),
				"route", route,
				"status", status,
				"bytes", writer.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			)
		})
	}
}

func redactedPath(r *http.Request, policy redact.Policy) string {
	routeContext := chi.RouteContext(r.Context())
	if routeContext == nil {
		return r.URL.Path
	}
	replacements := make([]string, 0, len(routeContext.URLParams.Keys)*2)
	for index, name := range routeContext.URLParams.Keys {
		if name == "" || index >= len(routeContext.URLParams.Values) || !policy.IsSensitive(name) {
			continue
		}
		value := routeContext.URLParams.Values[index]
		if value != "" {
			replacements = append(replacements, value, policy.Value(name, value))
		}
	}
	if len(replacements) == 0 {
		return r.URL.Path
	}
	return strings.NewReplacer(replacements...).Replace(r.URL.Path)
}
