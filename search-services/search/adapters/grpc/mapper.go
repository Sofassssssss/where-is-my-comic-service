package grpc

import (
	searchpb "where-is-my-comic-service/search-services/proto/search"
	"where-is-my-comic-service/search-services/search/core"
)

func toProtoComics(src []core.Comics) []*searchpb.Comics {
	out := make([]*searchpb.Comics, 0, len(src))

	for _, c := range src {
		out = append(out, &searchpb.Comics{
			Id:  c.ID,
			Url: c.URL,
		})
	}

	return out
}
