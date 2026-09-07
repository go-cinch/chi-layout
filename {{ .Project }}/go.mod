module {{ .Computed.module_name_final }}

go 1.27.1

require (
{{- if eq .Computed.http_router_final "gin" }}
	github.com/gin-gonic/gin v1.12.0
{{- else }}
	github.com/go-chi/chi/v5 v5.3.2
{{- end }}
	github.com/knadh/koanf/parsers/yaml v1.1.0
	github.com/knadh/koanf/providers/file v1.2.1
	github.com/knadh/koanf/v2 v2.3.5
	github.com/pmezard/go-difflib v1.0.0
	github.com/twpayne/go-jsonstruct/v3 v3.3.0
{{- if .Computed.enable_database_final }}
	github.com/go-sql-driver/mysql v1.9.3
	github.com/lib/pq v1.12.3
{{- end }}
{{- if and .Computed.enable_database_final .Computed.enable_trace_final }}
	github.com/XSAM/otelsql v0.42.0
{{- end }}
{{- if .Computed.enable_migrations_final }}
	github.com/rubenv/sql-migrate v1.8.1
{{- end }}
{{- if .Computed.enable_redis_final }}
	github.com/redis/go-redis/v9 v9.21.0
{{- end }}
{{- if .Computed.enable_trace_final }}
	go.opentelemetry.io/otel v1.44.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.44.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.44.0
	go.opentelemetry.io/otel/sdk v1.44.0
	go.opentelemetry.io/otel/trace v1.44.0
{{- end }}
)
