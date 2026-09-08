package db

import (
{{- if not .Computed.enable_game_example_final }}
	"context"
{{- else }}
	"io/fs"
{{- end }}
	"testing"
)

{{ if .Computed.enable_game_example_final -}}
func TestEmbeddedMigrationsIncludeSQL(t *testing.T) {
	files, err := fs.Glob(SQLFiles, SQLRoot+"/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("embedded migrations contain no SQL files")
	}
}
{{ else -}}
func TestMigrateUpSkipsWithoutSQLFiles(t *testing.T) {
	if err := MigrateUp(context.Background(), nil, "postgres"); err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}
}
{{ end }}
