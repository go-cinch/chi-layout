package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"{{ .Computed.module_name_final }}/internal/common/config"
{{- if .Computed.enable_grpc_final }}
"{{ .Computed.module_name_final }}/internal/docs"
"{{ .Computed.module_name_final }}/internal/modules"
"google.golang.org/grpc"
{{- end }}
{{- if .Computed.enable_database_final }}
	"{{ .Computed.module_name_final }}/internal/infra/db"
{{- end }}
)

func TestNewRouter(t *testing.T) {
	application := &Application{
{{- if .Computed.enable_database_final }}
		db: &db.Store{},
{{- end }}
{{- if .Computed.enable_redis_final }}
		rds: nil,
{{- end }}
	}
	var cfg config.Config
	cfg.HTTP.Docs.Enabled = true
	handler, err := application.NewRouter(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	path := "/missing"
	want := http.StatusNotFound
{{- if .Computed.enable_game_example_final }}
	path = "/game/bad"
	want = http.StatusBadRequest
{{- end }}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	if recorder.Code != want {
		t.Fatalf("status = %d, want %d", recorder.Code, want)
	}
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/docs/", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("docs status = %d", recorder.Code)
	}
}

{{ if .Computed.enable_grpc_final -}}
type rpcOnlyModule struct{}
func (rpcOnlyModule) GRPC(grpc.ServiceRegistrar) {}
func TestConfigureOptionalTransports(t *testing.T) {
 for _,test:=range []struct{http,grpc bool}{
  {true,false},{false,true},{true,true},
 } {
  a:=&Application{
{{- if .Computed.enable_database_final }}
   db:&db.Store{},
{{- end }}
  }
  var httpModules []modules.HTTPModule
  var grpcModules []modules.GRPCModule
  if test.http {h,err:=docs.New(nil);if err!=nil {t.Fatal(err)};httpModules=append(httpModules,h)}
  if test.grpc {grpcModules=append(grpcModules,rpcOnlyModule{})}
  if err:=a.configureTransports(&config.Config{},httpModules,grpcModules);err!=nil {t.Fatal(err)}
  if (a.server!=nil)!=test.http || (a.grpcServer!=nil)!=test.grpc {t.Fatalf("servers: http=%v grpc=%v",a.server!=nil,a.grpcServer!=nil)}
  if a.grpcServer!=nil {a.grpcServer.Stop()}
 }
}
{{ end -}}
