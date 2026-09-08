# {{ .Computed.service_name_final }}

A Go service using **{{ .Computed.http_router_final }}** with optional gRPC.

## Run

```bash
make gen
make tidy
make test
make run
```
