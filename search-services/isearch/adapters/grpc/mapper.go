package grpc

import (
	"where-is-my-comic-service/search-services/isearch/core"
	isearchpb "where-is-my-comic-service/search-services/proto/isearch"
)

func toProtoComics(src []core.Comics) []*isearchpb.Comics {
	out := make([]*isearchpb.Comics, 0, len(src))

	for _, c := range src {
		out = append(out, &isearchpb.Comics{
			Id:  c.ID,
			Url: c.URL,
		})
	}

	return out
}
