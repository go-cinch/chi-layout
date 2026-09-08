package game

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"{{ .Computed.module_name_final }}/internal/common/config"
	"{{ .Computed.module_name_final }}/internal/common/server"
)

func TestHTTP(t *testing.T) {
	m, mock := newTestModule(t)
	router, err := server.NewRouter(&config.Config{}, m)
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, path, body string, code int) *httptest.ResponseRecorder {
		t.Helper()
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(method, path, strings.NewReader(body)))
		if response.Code != code {
			t.Fatalf("%s %s: %d %s", method, path, response.Code, response.Body.String())
		}
		return response
	}
	expectList(mock, "%%", 1, 0)
	defaults := request("GET", "/game", "", 200).Body.String()
	if !strings.Contains(defaults, `"p":1`) || !strings.Contains(defaults, `"s":1`) {
		t.Fatal(defaults)
	}
	expectList(mock, "%Demo%", 10, 10)
	listed := request("GET", "/game?name=Demo&p=2&s=10", "", 200)
	var page map[string]any
	if err := json.Unmarshal(listed.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page["p"] != float64(2) || page["s"] != float64(10) || page["page"] != nil || page["pageSize"] != nil {
		t.Fatalf("pagination fields: %v", page)
	}
	if date := page["items"].([]any)[0].(map[string]any)["createdAt"]; date != float64(testTime.UnixMilli()) {
		t.Fatalf("list date must be numeric milliseconds: %v", date)
	}
	if !strings.Contains(listed.Body.String(), `"name":"Demo Game"`) {
		t.Fatal(listed.Body.String())
	}
	for _, query := range []string{"p=bad", "p=2147483648", "p=", "p=1.5", "p=1&p=2", "s=", "s=1.5", "s=20&s=30", "s=%zz", "s=1;2", "s=", "p"} {
		request("GET", "/game?"+query, "", 400)
	}
	for _, query := range []string{"p=0", "p=-1", "s=0", "s=-1", "p=10001", "s=10001"} {
		body := request("GET", "/game?"+query, "", 200).Body.String()
		if !strings.Contains(body, `"items":[]`) {
			t.Fatalf("empty pagination: %s", body)
		}
	}
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(2))
	if body := request("GET", "/game?p=3&s=1", "", 200).Body.String(); !strings.Contains(body, `"items":[]`) || !strings.Contains(body, `"total":2`) {
		t.Fatal(body)
	}
	mock.ExpectQuery("SELECT COUNT").WillReturnError(errors.New("database down"))
	request("GET", "/game", "", 500)
	mock.ExpectExec("DELETE FROM t_game WHERE id IN").WithArgs(int64(1), int64(2)).WillReturnResult(sqlmock.NewResult(0, 2))
	request("DELETE", "/game/1,2,1", "", 200)
	for _, ids := range []string{"1,bad", "1,", "1,,2", strings.Repeat("1,", 100) + "1"} {
		request("DELETE", "/game/"+ids, "", 400)
	}
	expectCreate(mock)
	request("POST", "/game", `{"name":"Demo Game","description":"description"}`, 200)
	expectGet(mock, "Demo Game")
	response := request("GET", "/game/1", "", 200)
	var value map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	if value["createdAt"] != float64(testTime.UnixMilli()) || value["updatedAt"] != value["createdAt"] {
		t.Fatalf("millisecond timestamps: %v", value)
	}
	expectUpdate(mock)
	request("PATCH", "/game/1", `{"name":"Updated Game","description":"description"}`, 200)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE t_game").WithArgs("Renamed", nil, sqlmock.AnyArg(), int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
	expectGet(mock, "Renamed")
	mock.ExpectCommit()
	request("PATCH", "/game/1", `{"name":"Renamed"}`, 200)
	request("PUT", "/game/1", `{"name":"Unused"}`, 405)
	expectDelete(mock)
	if body := request("DELETE", "/game/1", "", 200).Body.String(); body != "" {
		t.Fatalf("delete body: %s", body)
	}
	expectMissing(mock)
	request("GET", "/game/1", "", 404)
	for _, method := range []string{"GET", "PATCH", "DELETE"} {
		request(method, "/game/bad", `{"name":"valid"}`, 400)
		request(method, "/game/0", `{"name":"valid"}`, 400)
	}
	for _, method := range []string{"POST", "PATCH"} {
		path := "/game"
		if method == "PATCH" {
			path += "/1"
		}
		for _, body := range []string{"", `{}`, `{"name":"valid"} {}`, `{"unknown":1}`} {
			request(method, path, body, 400)
		}
	}
	request("GET", "/missing", "", 404)
	failure := errors.New("database down")
	mock.ExpectQuery("SELECT").WillReturnError(failure)
	request("GET", "/game/1", "", 500)
	mock.ExpectBegin().WillReturnError(failure)
	request("POST", "/game", `{"name":"valid"}`, 500)
	mock.ExpectBegin().WillReturnError(failure)
	request("PATCH", "/game/1", `{"name":"valid"}`, 500)
	mock.ExpectExec("DELETE").WillReturnError(failure)
	request("DELETE", "/game/1", "", 500)
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(0, 0))
	expectMissing(mock)
	mock.ExpectRollback()
	request("PATCH", "/game/1", `{"name":"valid"}`, 404)
	mock.ExpectExec("DELETE").WillReturnResult(sqlmock.NewResult(0, 0))
	request("DELETE", "/game/1", "", 404)
}
