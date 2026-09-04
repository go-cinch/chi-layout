package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequestTimeout(t *testing.T) {
	handler := RequestTimeout(time.Millisecond)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d", recorder.Code)
	}
}
