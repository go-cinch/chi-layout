package user

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"{{ .Computed.module_name_final }}/internal/infra/db"
)

const testDriverName = "chi-layout-user-test"

var registerTestDriver sync.Once

func TestFind(t *testing.T) {
	module := newTestModule(t)
	if module.Name() != "/user" {
		t.Fatalf("Name() = %q", module.Name())
	}
	value, err := module.Find(t.Context(), 1)
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if value.ID != 1 || value.Name != "Alice" || value.Email != "alice@example.com" {
		t.Fatalf("user = %#v", value)
	}
	if _, err := module.Find(t.Context(), 2); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing user error = %v", err)
	}
	if _, err := module.Find(t.Context(), 3); err == nil || !strings.Contains(err.Error(), "query user") {
		t.Fatalf("query error = %v", err)
	}
}

func newTestModule(t *testing.T) *Module {
	t.Helper()
	registerTestDriver.Do(func() {
		sql.Register(testDriverName, userTestDriver{})
	})
	database, err := sql.Open(testDriverName, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return New(&db.Store{DB: database})
}

type userTestDriver struct{}
type userTestConn struct{}
type userTestRows struct {
	values [][]driver.Value
	index  int
}

func (userTestDriver) Open(string) (driver.Conn, error) { return userTestConn{}, nil }
func (userTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not supported")
}
func (userTestConn) Close() error              { return nil }
func (userTestConn) Begin() (driver.Tx, error) { return nil, errors.New("not supported") }
func (userTestConn) QueryContext(_ context.Context, _ string, args []driver.NamedValue) (driver.Rows, error) {
	id, _ := args[0].Value.(int64)
	switch id {
	case 1:
		timestamp := time.Date(2026, time.September, 4, 12, 0, 0, 0, time.UTC)
		values := []driver.Value{id, timestamp, timestamp, "Alice", "alice@example.com"}
		return &userTestRows{values: [][]driver.Value{values}}, nil
	case 2:
		return &userTestRows{}, nil
	default:
		return nil, errors.New("database unavailable")
	}
}

func (*userTestRows) Columns() []string {
	return []string{"id", "created_at", "updated_at", "name", "email"}
}
func (*userTestRows) Close() error { return nil }
func (r *userTestRows) Next(destination []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(destination, r.values[r.index])
	r.index++
	return nil
}
