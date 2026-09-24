package game

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/encoding/protojson"
	gamepb "{{ .Computed.module_name_final }}/api/{{ .Computed.proto_service_name_final }}"
	"{{ .Computed.module_name_final }}/internal/common/config"
	"{{ .Computed.module_name_final }}/internal/common/rpc"
)

func TestGRPC(t *testing.T) {
	m, mock := newTestModule(t)
	srv, err := rpc.NewServer(&config.Config{}, m)
	if err != nil {
		t.Fatal(err)
	}
	listener := bufconn.Listen(1024 * 1024)
	go func() { _ = srv.Serve(listener) }()
	t.Cleanup(func() { srv.Stop(); _ = listener.Close() })
	conn, err := grpc.NewClient("passthrough:///game", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	client := gamepb.NewGameServiceClient(conn)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	expectList(mock, "%Demo%", 1, 0)
	defaultPage, err := client.ListGames(ctx, &gamepb.ListGamesRequest{Name: "Demo"})
	if err != nil || defaultPage.GetP() != 1 || defaultPage.GetS() != 1 {
		t.Fatalf("default pagination: %v %v", defaultPage, err)
	}
	expectList(mock, "%Demo%", 20, 0)
	request := new(gamepb.ListGamesRequest)
	if err := protojson.Unmarshal([]byte(`{"name":"Demo","p":1,"s":20}`), request); err != nil {
		t.Fatal(err)
	}
	listed, err := client.ListGames(ctx, request)
	if err != nil || listed.GetP() != 1 || listed.GetS() != 20 || listed.GetTotal() != 1 || len(listed.GetItems()) != 1 || listed.GetItems()[0].GetName() != "Demo Game" {
		t.Fatalf("list: %v %v", listed, err)
	}
	for _, req := range []*gamepb.ListGamesRequest{
		{P: pagePointer(0)}, {S: pagePointer(0)}, {P: pagePointer(-1)}, {P: pagePointer(10001)}, {S: pagePointer(10001)},
	} {
		reply, err := client.ListGames(ctx, req)
		if err != nil || len(reply.GetItems()) != 0 {
			t.Fatalf("RPC empty pagination: %v %v", reply, err)
		}
		if req.P != nil && reply.GetP() != req.GetP() {
			t.Fatal("RPC page was reset")
		}
		if req.S != nil && reply.GetS() != req.GetS() {
			t.Fatal("RPC size was reset")
		}
	}
	mock.ExpectQuery("SELECT COUNT").WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(2))
	end, err := client.ListGames(ctx, &gamepb.ListGamesRequest{P: pagePointer(3), S: pagePointer(1)})
	if err != nil || len(end.GetItems()) != 0 || end.GetTotal() != 2 {
		t.Fatalf("RPC last page: %v %v", end, err)
	}
	mock.ExpectExec("DELETE FROM t_game WHERE id IN").WithArgs(int64(1), int64(2)).WillReturnResult(sqlmock.NewResult(0, 2))
	if _, err := client.BatchDeleteGames(ctx, &gamepb.BatchDeleteGamesRequest{Ids: []int64{1, 2}}); err != nil {
		t.Fatal(err)
	}
	expectCreate(mock)
	value, err := client.CreateGame(ctx, &gamepb.CreateGameRequest{Name: "Demo Game", Description: "description"})
	if err != nil || value.GetId() != 1 || value.GetName() != "Demo Game" {
		t.Fatalf("create: %v %v", value, err)
	}
	expectGet(mock, "Demo Game")
	value, err = client.GetGame(ctx, &gamepb.GetGameRequest{Id: 1})
	if err != nil || value.GetCreatedAt() != testTime.UnixMilli() || value.GetUpdatedAt() != testTime.UnixMilli() {
		t.Fatalf("get: %v %v", value, err)
	}
	data, err := protojson.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["createdAt"] != strconv.FormatInt(testTime.UnixMilli(), 10) {
		t.Fatalf("millisecond int64 JSON: %s", data)
	}
	expectUpdate(mock)
	value, err = client.UpdateGame(ctx, &gamepb.UpdateGameRequest{Id: 1, Name: stringPointer("Updated Game"), Description: stringPointer("description")})
	if err != nil || value.GetName() != "Updated Game" {
		t.Fatalf("update: %v %v", value, err)
	}
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE t_game").WithArgs(nil, "", sqlmock.AnyArg(), int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT .* FROM t_game").WithArgs(int64(1)).WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name", "description"}).AddRow(1, testTime, testTime, "Updated Game", ""))
	mock.ExpectCommit()
	value, err = client.UpdateGame(ctx, &gamepb.UpdateGameRequest{Id: 1, Description: stringPointer("")})
	if err != nil || value.GetName() != "Updated Game" || value.GetDescription() != "" {
		t.Fatalf("optional fields: %v %v", value, err)
	}
	expectDelete(mock)
	if _, err := client.DeleteGame(ctx, &gamepb.DeleteGameRequest{Id: 1}); err != nil {
		t.Fatal(err)
	}
	expectDelete(mock)
	if _, err := client.BatchDeleteGames(ctx, &gamepb.BatchDeleteGamesRequest{Ids: []int64{1}}); err != nil {
		t.Fatal(err)
	}
	expectMissing(mock)
	if _, err := client.GetGame(ctx, &gamepb.GetGameRequest{Id: 1}); status.Code(err) != codes.NotFound {
		t.Fatal(err)
	}
	for _, call := range []func() error{
		func() error { _, err := client.GetGame(ctx, &gamepb.GetGameRequest{}); return err },
		func() error { _, err := client.CreateGame(ctx, &gamepb.CreateGameRequest{}); return err },
		func() error { _, err := client.UpdateGame(ctx, &gamepb.UpdateGameRequest{}); return err },
		func() error { _, err := client.DeleteGame(ctx, &gamepb.DeleteGameRequest{}); return err },
	} {
		if err := call(); status.Code(err) != codes.InvalidArgument {
			t.Fatal(err)
		}
	}
	mock.ExpectQuery("SELECT").WillReturnError(errors.New("secret database detail"))
	if _, err := client.GetGame(ctx, &gamepb.GetGameRequest{Id: 1}); status.Code(err) != codes.Internal || status.Convert(err).Message() != "internal server error" {
		t.Fatal(err)
	}
	mock.ExpectExec("DELETE").WillReturnResult(sqlmock.NewResult(0, 0))
	if _, err := client.BatchDeleteGames(ctx, &gamepb.BatchDeleteGamesRequest{Ids: []int64{1}}); status.Code(err) != codes.NotFound {
		t.Fatal(err)
	}
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
		if got := grpcError(ctx, "get game", cause); status.Code(got) != status.Code(status.FromContextError(cause).Err()) {
			t.Fatal(got)
		}
	}
}

func TestInt64DatesRoundTrip(t *testing.T) {
	original := &gamepb.Game{CreatedAt: math.MaxInt64, UpdatedAt: math.MinInt64}
	data, err := protojson.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var restored gamepb.Game
	if err := protojson.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.CreatedAt != original.CreatedAt || restored.UpdatedAt != original.UpdatedAt {
		t.Fatal("ProtoJSON int64 precision lost")
	}
}
