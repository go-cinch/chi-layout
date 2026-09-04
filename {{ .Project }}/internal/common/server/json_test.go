package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSONHelpers(t *testing.T) {
	recorder := httptest.NewRecorder()
	WriteOK(recorder)
	if recorder.Code != http.StatusOK || recorder.Body.Len() != 0 {
		t.Fatalf("empty response = %d %q", recorder.Code, recorder.Body.String())
	}
	recorder = httptest.NewRecorder()
	WriteError(recorder, http.StatusBadRequest, "bad request")
	var body ErrorResponse
	if recorder.Code != http.StatusBadRequest || json.Unmarshal(recorder.Body.Bytes(), &body) != nil || body.Msg != "bad request" {
		t.Fatalf("response = %d %s", recorder.Code, recorder.Body.String())
	}
	recorder = httptest.NewRecorder()
	WriteNoContent(recorder)
	if recorder.Code != http.StatusNoContent || recorder.Body.Len() != 0 {
		t.Fatalf("no content response = %d %q", recorder.Code, recorder.Body.String())
	}
	recorder = httptest.NewRecorder()
	WriteJSON(recorder, http.StatusOK, func() {})
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("encoding failure status = %d", recorder.Code)
	}
}
