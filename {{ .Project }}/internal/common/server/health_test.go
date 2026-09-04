package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthz(t *testing.T) {
	checks := 0
	deadlineSeen := false
	health := NewHealth(
		func(ctx context.Context) error {
			checks++
			deadline, ok := ctx.Deadline()
			deadlineSeen = ok && time.Until(deadline) <= healthCheckTimeout && time.Until(deadline) > healthCheckTimeout-time.Second
			return errors.New("unavailable")
		},
		func(context.Context) error { checks++; return nil },
	)
	if health.Name() != "/healthz" {
		t.Fatalf("name = %q", health.Name())
	}
	handler := health.HTTP()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if recorder.Code != http.StatusServiceUnavailable || checks != 2 || !deadlineSeen {
		t.Fatalf("response = %d, checks = %d", recorder.Code, checks)
	}
	recorder = httptest.NewRecorder()
	NewHealth().HTTP().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if recorder.Code != http.StatusOK || recorder.Body.Len() != 0 {
		t.Fatalf("healthy response = %d %q", recorder.Code, recorder.Body.String())
	}
	recorder = httptest.NewRecorder()
	NewHealth().HTTP().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/healthz", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("method response = %d", recorder.Code)
	}
}
