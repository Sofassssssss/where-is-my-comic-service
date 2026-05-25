package core

import (
	"context"
)

type Normalizer interface {
	Norm(context.Context, string) ([]string, error)
}

type Pinger interface {
	Ping(context.Context) error
}

type Updater interface {
	Update(context.Context) (UpdateResult, error)
	Stats(context.Context) (UpdateStats, error)
	Status(context.Context) (UpdateStatus, error)
	Drop(context.Context) error
}

type Searcher interface {
	Search(ctx context.Context, req SearchRequest) ([]Comics, error)
}

type ISearcher interface {
	ISearch(ctx context.Context, req ISearchRequest) ([]Comics, error)
}
