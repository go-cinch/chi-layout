# {{ .Computed.service_name_final }}

A minimal HTTP service using **{{ .Computed.http_router_final }}**.

## Run

```bash
make gen
make tidy
make test
make run
```
