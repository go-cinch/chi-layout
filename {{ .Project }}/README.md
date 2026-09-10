# {{ .Computed.service_name_final }}

A Go service using **{{ .Computed.http_router_final }}** {{ if .Computed.enable_grpc_final }}with optional gRPC{{ else }}for HTTP{{ end }}.

## Run

```bash
make gen
make tidy
make test
make run
```
