package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func NewProfilerHandler() http.Handler {
	r := chi.NewRouter()
	r.Mount("/debug", chimiddleware.Profiler())
	return r
}
