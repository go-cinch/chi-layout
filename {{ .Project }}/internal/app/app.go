package app

import (
	"context"
	"fmt"
{{- if .Computed.enable_trace_final }}
	"log/slog"
{{- end }}
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

{{- if .Computed.enable_grpc_final }}
	"google.golang.org/grpc"
{{- end }}
	"{{ .Computed.module_name_final }}/internal/common/config"
	"{{ .Computed.module_name_final }}/internal/common/logging"
	"{{ .Computed.module_name_final }}/internal/common/pagination"
{{- if .Computed.enable_grpc_final }}
	"{{ .Computed.module_name_final }}/internal/common/rpc"
{{- end }}
	"{{ .Computed.module_name_final }}/internal/common/server"
	"{{ .Computed.module_name_final }}/internal/modules"
{{- if .Computed.enable_trace_final }}
	"{{ .Computed.module_name_final }}/internal/common/tracing"
{{- end }}
{{- if .Computed.enable_database_final }}
	"{{ .Computed.module_name_final }}/internal/infra/db"
{{- end }}
{{- if .Computed.enable_redis_final }}
	"{{ .Computed.module_name_final }}/internal/infra/rds"
{{- end }}
)

type Application struct {
	pagination      pagination.Limits
	server          *http.Server
	profilerServer  *http.Server
{{- if .Computed.enable_grpc_final }}
	grpcServer      *grpc.Server
	grpcAddr        string
{{- end }}
	shutdownTimeout time.Duration
{{- if .Computed.enable_database_final }}
	db *db.Store
{{- end }}
{{- if .Computed.enable_redis_final }}
	rds rds.Client
{{- end }}
	cleanups []func()
}

func New(ctx context.Context, confPath string) (*Application, error) {
	cfg, overrides, err := config.LoadDir(confPath)
	if err != nil {
		return nil, err
	}

	if err := logging.Init(os.Stdout, cfg.Log.Level); err != nil {
		return nil, err
	}
	config.LogOverrides(overrides)
	limits, err := pagination.New(cfg.Pagination.MaxP, cfg.Pagination.MaxS)
	if err != nil {
		return nil, err
	}
	cleanups := make([]func(), 0, 3)
	cleanup := func() {
		for index := len(cleanups) - 1; index >= 0; index-- {
			cleanups[index]()
		}
	}
{{- if .Computed.enable_trace_final }}
	if cfg.Tracer.Enabled {
		provider, err := tracing.NewProvider(ctx, cfg)
		if err != nil {
			return nil, err
		}
		cleanups = append(cleanups, func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := provider.Shutdown(shutdownCtx); err != nil {
				slog.Error("shutdown tracer failed: " + err.Error())
			}
		})
	}
{{- end }}
{{- if .Computed.enable_database_final }}
	dbStore, cleanupDB, err := db.New(ctx, cfg)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("initialize database: %w", err)
	}
	cleanups = append(cleanups, cleanupDB)
{{- end }}
{{- if .Computed.enable_redis_final }}
	rdsClient, cleanupRDS, err := rds.New(ctx, cfg)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("initialize redis: %w", err)
	}
	cleanups = append(cleanups, cleanupRDS)
{{- end }}

	application := &Application{
		pagination: limits,
{{- if .Computed.enable_database_final }}
		db: dbStore,
{{- end }}
{{- if .Computed.enable_redis_final }}
		rds: rdsClient,
{{- end }}
		cleanups: cleanups,
	}
	httpModules{{ if .Computed.enable_grpc_final }}, grpcModules{{ end }} := application.generatedModules()
	if err := application.configureTransports(cfg, httpModules{{ if .Computed.enable_grpc_final }}, grpcModules{{ end }}); err != nil {
		cleanup()
		return nil, err
	}
	return application, nil
}

func SignalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

func (a *Application) Close() {
	for index := len(a.cleanups) - 1; index >= 0; index-- {
		a.cleanups[index]()
	}
}

func (a *Application) configureTransports(cfg *config.Config, httpModules []modules.HTTPModule{{ if .Computed.enable_grpc_final }}, grpcModules []modules.GRPCModule{{ end }}) error {
{{- if .Computed.enable_grpc_final }}
	// Start HTTP for HTTP modules or an empty scaffold; helpers do not enable it.
	if len(httpModules) > 0 || len(grpcModules) == 0 {
{{- end }}
		handler, err := a.newRouter(cfg, httpModules)
		if err != nil {
			return fmt.Errorf("initialize http router: %w", err)
		}
		a.server = &http.Server{Addr: cfg.HTTP.Addr, Handler: handler, ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout, IdleTimeout: cfg.HTTP.IdleTimeout}
{{- if .Computed.enable_grpc_final }}
	}
	if len(grpcModules) > 0 {
		var err error
		a.grpcServer, err = rpc.NewServer(cfg, grpcModules...)
		if err != nil {
			return fmt.Errorf("initialize grpc server: %w", err)
		}
		a.grpcAddr = cfg.GRPC.Addr
	}
	a.shutdownTimeout = cfg.GRPC.ShutdownTimeout
{{- end }}
	if cfg.HTTP.Profiler.Enabled {
		a.profilerServer = &http.Server{
			Addr:              cfg.HTTP.Profiler.Addr,
			Handler:           server.NewProfilerHandler(),
			ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
			IdleTimeout:       cfg.HTTP.IdleTimeout,
		}
	}

	return nil
}
