package grpc

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	searchpb "where-is-my-comic-service/search-services/proto/search"
	"where-is-my-comic-service/search-services/search/core"
)

func NewServer(service core.Searcher) *Server {
	return &Server{service: service}
}

type Server struct {
	searchpb.UnimplementedSearchServer
	service core.Searcher
}

func (s *Server) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) { return nil, nil }

func (s *Server) Search(ctx context.Context, in *searchpb.SearchRequest) (*searchpb.SearchResponse, error) {
	searchReply, err := s.service.Search(ctx, core.SearchRequest{
		Phrase: in.Phrase,
		Limit:  int(in.GetLimit()),
	})
	if err != nil {
		slog.Error("Error", "err", err)
		return nil, status.Error(codes.Internal, "failed to search")
	}
	return &searchpb.SearchResponse{
		Comics: toProtoComics(searchReply),
	}, nil
}
