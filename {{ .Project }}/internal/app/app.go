package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"{{ .Computed.module_name_final }}/internal/common/config"
	"{{ .Computed.module_name_final }}/internal/common/logging"
	"{{ .Computed.module_name_final }}/internal/common/server"
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
	server         *http.Server
	profilerServer *http.Server
{{- if .Computed.enable_database_final }}
	db *db.Store
{{- end }}
{{- if .Computed.enable_redis_final }}
	rds rds.Client
{{- end }}
	cleanups []func()
}

type serverResult struct {
	name string
	err  error
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
{{- if .Computed.enable_database_final }}
		db: dbStore,
{{- end }}
{{- if .Computed.enable_redis_final }}
		rds: rdsClient,
{{- end }}
		cleanups: cleanups,
	}
	handler, err := application.NewRouter(cfg)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("initialize http router: %w", err)
	}

	application.server = &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           handler,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}
	if cfg.HTTP.Profiler.Enabled {
		application.profilerServer = &http.Server{
			Addr:              cfg.HTTP.Profiler.Addr,
			Handler:           server.NewProfilerHandler(),
			ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
			IdleTimeout:       cfg.HTTP.IdleTimeout,
		}
	}
	return application, nil
}

func SignalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

func (a *Application) Run(ctx context.Context) error {
	if ctx.Err() != nil {
		return nil
	}
	httpAddress := a.server.Addr
	if httpAddress == "" {
		httpAddress = ":http"
	}
	httpListener, err := net.Listen("tcp", httpAddress)
	if err != nil {
		return fmt.Errorf("listen http server: %w", err)
	}
	var profilerListener net.Listener
	if a.profilerServer != nil {
		profilerAddress := a.profilerServer.Addr
		if profilerAddress == "" {
			profilerAddress = ":http"
		}
		profilerListener, err = net.Listen("tcp", profilerAddress)
		if err != nil {
			_ = httpListener.Close()
			return fmt.Errorf("listen profiler server: %w", err)
		}
	}

	errCh := make(chan serverResult, 2)
	slog.Info("http server running at " + a.server.Addr)
	go func() {
		errCh <- serverResult{name: "http", err: a.server.Serve(httpListener)}
	}()
	if a.profilerServer != nil {
		slog.Info("profiler server running at " + a.profilerServer.Addr)
		go func() {
			errCh <- serverResult{name: "profiler", err: a.profilerServer.Serve(profilerListener)}
		}()
	}

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var shutdownErr error
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			shutdownErr = errors.Join(shutdownErr, err, a.server.Close())
		}
		if a.profilerServer != nil {
			if err := a.profilerServer.Shutdown(shutdownCtx); err != nil {
				shutdownErr = errors.Join(shutdownErr, err, a.profilerServer.Close())
			}
		}
		if shutdownErr != nil {
			return shutdownErr
		}
		slog.Info("http server stopped")
		return nil
	case result := <-errCh:
		if result.err != nil && !errors.Is(result.err, http.ErrServerClosed) {
			_ = a.server.Close()
			if a.profilerServer != nil {
				_ = a.profilerServer.Close()
			}
			return fmt.Errorf("%s server: %w", result.name, result.err)
		}
		return nil
	}
}

func (a *Application) Close() {
	for index := len(a.cleanups) - 1; index >= 0; index-- {
		a.cleanups[index]()
	}
}
