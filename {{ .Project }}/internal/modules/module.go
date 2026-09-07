{{ if eq .Computed.http_router_final "gin" -}}
package modules

import "github.com/gin-gonic/gin"

type Module interface {
	Name() string
	HTTP(*gin.RouterGroup)
}
{{ else -}}
package modules

import "net/http"

type Module interface {
	Name() string
	HTTP() http.Handler
}
{{ end -}}
