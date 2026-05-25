package grpc

import (
	"context"
	"log/slog"

	"google.golang.org/protobuf/types/known/emptypb"
	"where-is-my-comic-service/search-services/isearch/core"
	isearchpb "where-is-my-comic-service/search-services/proto/isearch"
)

func NewServer(service core.ISearcher) *Server {
	return &Server{service: service}
}

type Server struct {
	isearchpb.UnimplementedISearchServer
	service core.ISearcher
}

func (s *Server) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) { return nil, nil }

func (s *Server) ISearch(ctx context.Context, in *isearchpb.ISearchRequest) (*isearchpb.ISearchResponse, error) {
	iSearchReply, err := s.service.ISearch(ctx, core.ISearchRequest{
		Phrase: in.Phrase,
		Limit:  int(in.GetLimit()),
	})
	if err != nil {
		slog.Error("Error", "err", err)
		return nil, MapGRPCError(err)
	}
	return &isearchpb.ISearchResponse{
		Comics: toProtoComics(iSearchReply),
	}, nil
}
