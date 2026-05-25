package core

import (
	"context"
)

type Searcher interface {
	Search(ctx context.Context, request SearchRequest) ([]Comics, error)
}

type DB interface {
	GetComicsData(ctx context.Context) ([]ComicsKeyWords, error)
	GetImageURL(ctx context.Context, ComicsID int) (string, error)
}

type Words interface {
	Norm(ctx context.Context, phrase string) ([]string, error)
}
