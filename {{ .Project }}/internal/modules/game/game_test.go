package game

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"{{ .Computed.module_name_final }}/internal/common/pagination"
	"{{ .Computed.module_name_final }}/internal/infra/db"
)

var testTime = time.Date(2026, 9, 8, 10, 0, 0, 123456000, time.FixedZone("CST", 8*60*60))

func newTestModule(t *testing.T) (*Module, sqlmock.Sqlmock) {
	t.Helper()
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
		_ = database.Close()
	})
	limits, err := pagination.New(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	return New(&db.Store{DB: database}, limits), mock
}
func gameRows(name string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name", "description"}).AddRow(1, testTime, testTime, name, "description")
}
func expectGet(mock sqlmock.Sqlmock, name string) {
	mock.ExpectQuery("SELECT .* FROM t_game").WithArgs(int64(1)).WillReturnRows(gameRows(name))
}
func expectCreate(mock sqlmock.Sqlmock) {
	mock.ExpectBegin()
{{ if eq .Computed.database_driver_final "mysql" }}
	mock.ExpectExec("INSERT INTO t_game").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "Demo Game", "description").WillReturnResult(sqlmock.NewResult(1, 1))
{{ else }}
	mock.ExpectQuery("INSERT INTO t_game").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "Demo Game", "description").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
{{ end }}
	expectGet(mock, "Demo Game")
	mock.ExpectCommit()
}
func expectUpdate(mock sqlmock.Sqlmock) {
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE t_game").WithArgs("Updated Game", "description", sqlmock.AnyArg(), int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
	expectGet(mock, "Updated Game")
	mock.ExpectCommit()
}
func expectDelete(mock sqlmock.Sqlmock) {
	mock.ExpectExec("DELETE FROM t_game").WithArgs(int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
}
func expectMissing(mock sqlmock.Sqlmock) {
	mock.ExpectQuery("SELECT .* FROM t_game").WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name", "description"}))
}

func TestCRUD(t *testing.T) {
	m, mock := newTestModule(t)
	expectCreate(mock)
	value, err := m.Create(t.Context(), Input{Name: " Demo Game ", Description: "description"})
	if err != nil || value.Name != "Demo Game" || value.ID != 1 {
		t.Fatalf("create: %v %v", value, err)
	}
	expectGet(mock, "Demo Game")
	value, err = m.Get(t.Context(), 1)
	if err != nil || value.CreatedAt != testTime.UnixMilli() || value.UpdatedAt != testTime.UnixMilli() {
		t.Fatalf("get: %v %v", value, err)
	}
	expectUpdate(mock)
	value, err = m.Update(t.Context(), 1, UpdateInput{Name: stringPointer("Updated Game"), Description: stringPointer("description")})
	if err != nil || value.Name != "Updated Game" {
		t.Fatalf("update: %v %v", value, err)
	}
	expectDelete(mock)
	if err := m.Delete(t.Context(), 1); err != nil {
		t.Fatal(err)
	}
	expectMissing(mock)
	if _, err := m.Get(t.Context(), 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted game: %v", err)
	}
}
func TestInvalidInput(t *testing.T) {
	m, _ := newTestModule(t)
	for _, name := range []string{"", "  ", strings.Repeat("界", 101)} {
		if _, err := m.Create(t.Context(), Input{Name: name}); !errors.Is(err, ErrInvalid) {
			t.Fatal(err)
		}
		if _, err := m.Update(t.Context(), 1, UpdateInput{Name: stringPointer(name)}); !errors.Is(err, ErrInvalid) {
			t.Fatal(err)
		}
	}
	if _, err := m.Get(t.Context(), 0); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if _, err := m.Update(t.Context(), 0, UpdateInput{Name: stringPointer("valid")}); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if err := m.Delete(t.Context(), 0); !errors.Is(err, ErrIDs) {
		t.Fatal(err)
	}
}
func TestDatabaseFailures(t *testing.T) {
	m, mock := newTestModule(t)
	failure := errors.New("database down")
	mock.ExpectQuery("SELECT").WillReturnError(failure)
	if _, err := m.Get(t.Context(), 1); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	mock.ExpectBegin().WillReturnError(failure)
	if _, err := m.Create(t.Context(), Input{Name: "valid"}); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	mock.ExpectBegin()
{{ if eq .Computed.database_driver_final "mysql" }}
	mock.ExpectExec("INSERT").WillReturnError(failure)
{{ else }}
	mock.ExpectQuery("INSERT").WillReturnError(failure)
{{ end }}
	mock.ExpectRollback()
	if _, err := m.Create(t.Context(), Input{Name: "valid"}); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").WillReturnError(failure)
	mock.ExpectRollback()
	if _, err := m.Update(t.Context(), 1, UpdateInput{Name: stringPointer("valid")}); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 0))
	expectMissing(mock)
	mock.ExpectRollback()
	if _, err := m.Update(t.Context(), 1, UpdateInput{Name: stringPointer("valid")}); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	mock.ExpectExec("DELETE").WillReturnError(failure)
	if err := m.Delete(t.Context(), 1); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	mock.ExpectExec("DELETE").WillReturnResult(sqlmock.NewErrorResult(failure))
	if err := m.Delete(t.Context(), 1); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	mock.ExpectExec("DELETE").WillReturnResult(sqlmock.NewResult(0, 0))
	if err := m.Delete(t.Context(), 1); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := m.Get(ctx, 1); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func stringPointer(value string) *string { return &value }
func pagePointer(value int32) *int32     { return &value }

func TestPartialUpdate(t *testing.T) {
	m, mock := newTestModule(t)
	if _, err := m.Update(t.Context(), 1, UpdateInput{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty patch: %v", err)
	}
	for _, tc := range []struct {
		input                         UpdateInput
		name, description             any
		resultName, resultDescription string
	}{
		{UpdateInput{Name: stringPointer("Renamed")}, "Renamed", nil, "Renamed", "description"},
		{UpdateInput{Description: stringPointer("")}, nil, "", "Original", ""},
	} {
		mock.ExpectBegin()
		mock.ExpectExec("UPDATE t_game SET name = COALESCE").WithArgs(tc.name, tc.description, sqlmock.AnyArg(), int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectQuery("SELECT .* FROM t_game").WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name", "description"}).AddRow(1, testTime, testTime, tc.resultName, tc.resultDescription))
		mock.ExpectCommit()
		value, err := m.Update(t.Context(), 1, tc.input)
		if err != nil || value.Name != tc.resultName || value.Description != tc.resultDescription {
			t.Fatalf("partial update: %v %v", value, err)
		}
	}
}

func expectList(mock sqlmock.Sqlmock, filter string, pageSize, offset int64) {
	mock.ExpectQuery("SELECT COUNT").WithArgs(filter).WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(offset + 1))
	mock.ExpectQuery("SELECT id.*ORDER BY id DESC LIMIT").WithArgs(filter, pageSize, offset).WillReturnRows(gameRows("Demo Game"))
}
func TestList(t *testing.T) {
	m, mock := newTestModule(t)
	expectList(mock, "%Demo%", 1, 0)
	value, err := m.List(t.Context(), ListInput{Name: " Demo "})
	if err != nil || value.Total != 1 || value.Page != 1 || value.PageSize != 1 || len(value.Items) != 1 || value.Items[0].Name != "Demo Game" || value.Items[0].CreatedAt != testTime.UnixMilli() || value.Items[0].UpdatedAt != testTime.UnixMilli() {
		t.Fatalf("list: %v %v", value, err)
	}
	expectList(mock, "%!%!!%", 10, 10)
	if _, err := m.List(t.Context(), ListInput{Name: "%!", Page: pagePointer(2), PageSize: pagePointer(10)}); err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(0))
	value, err = m.List(t.Context(), ListInput{})
	if err != nil || value.Items == nil || len(value.Items) != 0 {
		t.Fatalf("empty list: %v %v", value, err)
	}
	for _, input := range []ListInput{
		{Page: pagePointer(-1)}, {Page: pagePointer(10001)}, {PageSize: pagePointer(-1)}, {PageSize: pagePointer(10001)},
	} {
		if value, err := m.List(t.Context(), input); err != nil || len(value.Items) != 0 || value.Items == nil {
			t.Fatalf("out of range: %v %v", value, err)
		}
	}
	failure := errors.New("database down")
	mock.ExpectQuery("SELECT COUNT").WillReturnError(failure)
	if _, err := m.List(t.Context(), ListInput{}); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(1))
	mock.ExpectQuery("SELECT id").WillReturnError(failure)
	if _, err := m.List(t.Context(), ListInput{}); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(1))
	mock.ExpectQuery("SELECT id").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	if _, err := m.List(t.Context(), ListInput{}); err == nil {
		t.Fatal("scan error ignored")
	}
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(1))
	mock.ExpectQuery("SELECT id").WillReturnRows(gameRows("Demo").RowError(0, failure))
	if _, err := m.List(t.Context(), ListInput{}); !errors.Is(err, failure) {
		t.Fatal(err)
	}
}
func TestDeleteMany(t *testing.T) {
	m, mock := newTestModule(t)
	for _, ids := range [][]int64{nil, {0}, {1, -2}, make([]int64, 101)} {
		if err := m.Delete(t.Context(), ids...); !errors.Is(err, ErrIDs) {
			t.Fatal(err)
		}
	}
	// Validate all IDs before SQL and bind each unique value separately.
	mock.ExpectExec(`DELETE FROM t_game WHERE id IN`).WithArgs(int64(1), int64(2)).WillReturnResult(sqlmock.NewResult(0, 2))
	if err := m.Delete(t.Context(), 1, 2, 1); err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec(`DELETE FROM t_game WHERE id IN`).WithArgs(int64(1), int64(999)).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := m.Delete(t.Context(), 1, 999); err != nil {
		t.Fatal(err)
	}
}

func TestConfiguredPagination(t *testing.T) {
	m, mock := newTestModule(t)
	m.limits = pagination.Limits{MaxP: 2, MaxS: 5}
	expectList(mock, "%%", 1, 0)
	value, err := m.List(t.Context(), ListInput{})
	if err != nil || value.PageSize != 1 {
		t.Fatalf("configured default: %v %v", value, err)
	}
	for _, input := range []ListInput{
		{Page: pagePointer(3), PageSize: pagePointer(1)}, {Page: pagePointer(1), PageSize: pagePointer(6)},
	} {
		if value, err := m.List(t.Context(), input); err != nil || len(value.Items) != 0 || value.Items == nil {
			t.Fatalf("out of range: %v %v", value, err)
		}
	}
}

func TestPaginationEdges(t *testing.T) {
	m, mock := newTestModule(t)
	for _, input := range []ListInput{
		{Page: pagePointer(0)}, {PageSize: pagePointer(0)}, {Page: pagePointer(-1)}, {PageSize: pagePointer(-1)},
	} {
		value, err := m.List(t.Context(), input)
		if err != nil || value.Items == nil || len(value.Items) != 0 {
			t.Fatalf("edge: %v %v", value, err)
		}
		if input.Page != nil && value.Page != *input.Page {
			t.Fatal("page was reset")
		}
		if input.PageSize != nil && value.PageSize != *input.PageSize {
			t.Fatal("size was reset")
		}
	}
	// The data ends on page 2: page 3 returns an empty list without querying rows.
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(2))
	value, err := m.List(t.Context(), ListInput{Page: pagePointer(3), PageSize: pagePointer(1)})
	if err != nil || len(value.Items) != 0 || value.Total != 2 || value.Page != 3 {
		t.Fatalf("past last page: %v %v", value, err)
	}
}

func TestDateMilliseconds(t *testing.T) {
	m, mock := newTestModule(t)
	for _, instant := range []time.Time{
		time.Unix(0, 0), time.Unix(-1, 999999999), testTime,
		time.Date(9999, 12, 31, 23, 59, 59, 999999000, time.UTC),
	} {
		mock.ExpectQuery("SELECT .* FROM t_game").WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name", "description"}).AddRow(1, instant, instant, "Game", ""))
		value, err := m.Get(t.Context(), 1)
		if err != nil || value.CreatedAt != instant.UnixMilli() || value.UpdatedAt != instant.UnixMilli() {
			t.Fatalf("milliseconds: %v %v", value, err)
		}
	}
}
