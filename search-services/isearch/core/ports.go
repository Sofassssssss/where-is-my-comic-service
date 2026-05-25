package core

import (
	"context"
)

type ISearcher interface {
	ISearch(ctx context.Context, request ISearchRequest) ([]Comics, error)
}

type DB interface {
	GetComicsData(ctx context.Context) ([]ComicsKeyWords, error)
	GetImageURL(ctx context.Context, ComicsID int) (string, error)
}

type Words interface {
	Norm(ctx context.Context, phrase string) ([]string, error)
}

type IndexBuilder interface {
	RebuildIndex(ctx context.Context) error
}
