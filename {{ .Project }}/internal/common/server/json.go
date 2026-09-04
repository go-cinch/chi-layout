package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type ErrorResponse struct {
	Msg string `json:"msg"`
}

func WriteJSON(w http.ResponseWriter, status int, body any) {
	data, err := json.Marshal(body)
	if err != nil {
		slog.Error("encode json response failed: " + err.Error())
		status = http.StatusInternalServerError
		data = []byte(`{"msg":"Internal Server Error"}`)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(data); err != nil {
		slog.Debug("write json response failed: " + err.Error())
	}
}

func WriteOK(w http.ResponseWriter, body ...any) {
	if len(body) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}
	WriteJSON(w, http.StatusOK, body[0])
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, ErrorResponse{Msg: message})
}

func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
