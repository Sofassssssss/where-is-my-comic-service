package grpc

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	updatepb "where-is-my-comic-service/search-services/proto/update"
	"where-is-my-comic-service/search-services/update/core"
)

func NewServer(service core.Updater) *Server {
	return &Server{service: service}
}

type Server struct {
	updatepb.UnimplementedUpdateServer
	service core.Updater
}

func (s *Server) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) { return nil, nil }

func (s *Server) Status(ctx context.Context, _ *emptypb.Empty) (*updatepb.StatusReply, error) {
	serviceStatus := s.service.Status(ctx)
	return &updatepb.StatusReply{
		Status: ToProtoStatus(serviceStatus),
	}, nil
}

func (s *Server) Update(ctx context.Context, _ *emptypb.Empty) (*updatepb.UpdateReply, error) {
	if s.service.Status(ctx) == core.StatusRunning {
		return nil, status.Error(codes.Unavailable, "update is already running")
	}
	updateReply, err := s.service.Update(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update")
	}
	return &updatepb.UpdateReply{
		ComicsInserted: int64(updateReply.ComicsInserted),
	}, nil
}

func (s *Server) Stats(ctx context.Context, _ *emptypb.Empty) (*updatepb.StatsReply, error) {
	serviceStats, err := s.service.Stats(ctx)
	if err != nil {
		slog.Error("Error while getting stats", "err", err)
		return nil, status.Error(codes.Internal, "failed to get stats")
	}
	return &updatepb.StatsReply{
		WordsTotal:    int64(serviceStats.WordsTotal),
		WordsUnique:   int64(serviceStats.WordsUnique),
		ComicsFetched: int64(serviceStats.ComicsFetched),
		ComicsTotal:   int64(serviceStats.ComicsTotal),
	}, nil
}

func (s *Server) Drop(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	err := s.service.Drop(ctx)
	return &emptypb.Empty{}, err
}
