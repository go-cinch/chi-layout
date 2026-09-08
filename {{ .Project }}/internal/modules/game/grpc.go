package game

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	gamepb "{{ .Computed.module_name_final }}/api/{{ .Computed.proto_service_name_final }}"
	"{{ .Computed.module_name_final }}/internal/modules"
)

var _ modules.GRPCModule = (*Module)(nil)

func (m *Module) GRPC(registrar grpc.ServiceRegistrar) {
	gamepb.RegisterGameServiceServer(registrar, &grpcHandler{module: m})
}

type grpcHandler struct {
	gamepb.UnimplementedGameServiceServer
	module *Module
}

func (h *grpcHandler) GetGame(ctx context.Context, req *gamepb.GetGameRequest) (*gamepb.Game, error) {
	value, err := h.module.Get(ctx, req.GetId())
	if err != nil {
		return nil, grpcError(ctx, "get game "+strconv.FormatInt(req.GetId(), 10), err)
	}
	return toProto(value), nil
}
func (h *grpcHandler) CreateGame(ctx context.Context, req *gamepb.CreateGameRequest) (*gamepb.Game, error) {
	value, err := h.module.Create(ctx, Input{Name: req.GetName(), Description: req.GetDescription()})
	if err != nil {
		return nil, grpcError(ctx, "create game", err)
	}
	return toProto(value), nil
}
func (h *grpcHandler) UpdateGame(ctx context.Context, req *gamepb.UpdateGameRequest) (*gamepb.Game, error) {
	if req == nil {
		return nil, grpcError(ctx, "update game", ErrInvalid)
	}
	value, err := h.module.Update(ctx, req.GetId(), UpdateInput{Name: req.Name, Description: req.Description})
	if err != nil {
		return nil, grpcError(ctx, "update game "+strconv.FormatInt(req.GetId(), 10), err)
	}
	return toProto(value), nil
}
func (h *grpcHandler) BatchDeleteGames(ctx context.Context, req *gamepb.BatchDeleteGamesRequest) (*emptypb.Empty, error) {
	if err := h.module.Delete(ctx, req.GetIds()...); err != nil {
		return nil, grpcError(ctx, fmt.Sprintf("delete games %v", req.GetIds()), err)
	}
	return &emptypb.Empty{}, nil
}
func toProto(value *Game) *gamepb.Game {
	return &gamepb.Game{Id: value.ID, Name: value.Name, Description: value.Description, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}
func grpcError(ctx context.Context, operation string, err error) error {
	switch {
	case errors.Is(err, ErrInvalid), errors.Is(err, ErrIDs):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, ErrNotFound):
		return status.Error(codes.NotFound, ErrNotFound.Error())
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	default:
		slog.ErrorContext(ctx, operation+" failed: "+err.Error())
		return status.Error(codes.Internal, "internal server error")
	}
}

func (h *grpcHandler) ListGames(ctx context.Context, req *gamepb.ListGamesRequest) (*gamepb.ListGamesResponse, error) {
	input := ListInput{Name: req.GetName()}
	if req != nil {
		input.Page = req.P
		input.PageSize = req.S
	}
	value, err := h.module.List(ctx, input)
	if err != nil {
		return nil, grpcError(ctx, "list games", err)
	}
	reply := &gamepb.ListGamesResponse{Items: make([]*gamepb.Game, 0, len(value.Items)), Total: value.Total, P: value.Page, S: value.PageSize}
	for i := range value.Items {
		reply.Items = append(reply.Items, toProto(&value.Items[i]))
	}
	return reply, nil
}

func (h *grpcHandler) DeleteGame(ctx context.Context, req *gamepb.DeleteGameRequest) (*emptypb.Empty, error) {
	if err := h.module.Delete(ctx, req.GetId()); err != nil {
		return nil, grpcError(ctx, "delete game "+strconv.FormatInt(req.GetId(), 10), err)
	}
	return &emptypb.Empty{}, nil
}
