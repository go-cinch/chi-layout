package modules

import "net/http"

type Module interface {
	Name() string
	HTTP() http.Handler
}
