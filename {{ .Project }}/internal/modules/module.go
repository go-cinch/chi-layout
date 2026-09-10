package modules

import (
{{- if eq .Computed.http_router_final "gin" }}
 "github.com/gin-gonic/gin"
{{- else }}
 "net/http"
{{- end }}
{{- if .Computed.enable_grpc_final }}
 "google.golang.org/grpc"
{{- end }}
)

type HTTPModule interface {
 Name() string
{{- if eq .Computed.http_router_final "gin" }}
 HTTP(*gin.RouterGroup)
{{- else }}
 HTTP() http.Handler
{{- end }}
}

// Module aliases HTTPModule.
type Module = HTTPModule

{{ if .Computed.enable_grpc_final -}}
// GRPCModule exposes RPC services without requiring an HTTP route or Name.
type GRPCModule interface {
 GRPC(grpc.ServiceRegistrar)
}
{{ end -}}
