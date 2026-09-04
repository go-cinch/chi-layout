package app

import (
{{- if and .Computed.enable_health_check_final .Computed.enable_redis_final }}
	"context"
{{- end }}
	"net/http"

	"{{ .Computed.module_name_final }}/internal/common/config"
	"{{ .Computed.module_name_final }}/internal/common/server"
	"{{ .Computed.module_name_final }}/internal/docs"
{{- if .Computed.enable_health_check_final }}
	"{{ .Computed.module_name_final }}/internal/modules"
{{- end }}
)

func (a *Application) NewRouter(cfg *config.Config) (http.Handler, error) {
{{- if .Computed.enable_health_check_final }}
	healthChecks := make([]server.HealthCheck, 0, 2)
{{- if .Computed.enable_database_final }}
	healthChecks = append(healthChecks, a.db.DB.PingContext)
{{- end }}
{{- if .Computed.enable_redis_final }}
	healthChecks = append(healthChecks, func(ctx context.Context) error {
		return a.rds.Ping(ctx).Err()
	})
{{- end }}
{{- end }}

	mounted := a.generatedModules()
{{- if .Computed.enable_health_check_final }}
	mounted = append([]modules.Module{server.NewHealth(healthChecks...)}, mounted...)
{{- end }}
	if cfg.HTTP.Docs.Enabled {
		mounted = append(mounted, docs.New())
	}
	return server.NewRouter(cfg, mounted...)
}
