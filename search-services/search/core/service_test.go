package core

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

type MockWords struct {
	NormFunc func(ctx context.Context, s string) ([]string, error)
}

type MockDB struct {
	GetComicsDataFunc func(ctx context.Context) ([]ComicsKeyWords, error)
	GetImageURLFunc   func(ctx context.Context, id int) (string, error)
}

func (m *MockDB) GetComicsData(ctx context.Context) ([]ComicsKeyWords, error) {
	return m.GetComicsDataFunc(ctx)
}

func (m *MockDB) GetImageURL(ctx context.Context, id int) (string, error) {
	return m.GetImageURLFunc(ctx, id)
}

func (m *MockWords) Norm(ctx context.Context, s string) ([]string, error) {
	return m.NormFunc(ctx, s)
}

func TestSearch_MultipleCases(t *testing.T) {
	db := &MockDB{
		GetComicsDataFunc: func(ctx context.Context) ([]ComicsKeyWords, error) {
			return []ComicsKeyWords{
				{ID: 1, KeyWords: []string{"linux", "love", "tree"}},
				{ID: 2, KeyWords: []string{"earth", "fire", "govern"}},
				{ID: 3, KeyWords: []string{"earth", "lake", "planet"}},
			}, nil
		},
		GetImageURLFunc: func(ctx context.Context, id int) (string, error) {
			return "url", nil
		},
	}

	words := &MockWords{
		NormFunc: func(ctx context.Context, s string) ([]string, error) {
			return []string{s}, nil
		},
	}

	svc, _ := NewService(slog.Default(), db, words)

	tests := []struct {
		name     string
		phrase   string
		limit    int
		expected int
	}{
		{
			name:     "one result",
			phrase:   "linux",
			limit:    10,
			expected: 1,
		},
		{
			name:     "two results",
			phrase:   "earth",
			limit:    10,
			expected: 2,
		},
		{
			name:     "no results",
			phrase:   "unknown",
			limit:    10,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := svc.Search(context.Background(), SearchRequest{
				Phrase: tt.phrase,
				Limit:  tt.limit,
			})

			require.NoError(t, err)
			require.Len(t, res, tt.expected)
		})
	}
}

func TestSearch_NormError(t *testing.T) {
	words := &MockWords{
		NormFunc: func(ctx context.Context, s string) ([]string, error) {
			return nil, errors.New("norm 	fail")
		},
	}

	svc, _ := NewService(slog.Default(), &MockDB{}, words)

	_, err := svc.Search(context.Background(), SearchRequest{})

	require.Error(t, err)
}

func TestSearch_EmptyCorpus(t *testing.T) {
	db := &MockDB{
		GetComicsDataFunc: func(ctx context.Context) ([]ComicsKeyWords, error) {
			return []ComicsKeyWords{}, nil
		},
	}

	words := &MockWords{
		NormFunc: func(ctx context.Context, s string) ([]string, error) {
			return []string{"linux"}, nil
		},
	}

	svc, _ := NewService(slog.Default(), db, words)

	res, err := svc.Search(context.Background(), SearchRequest{})

	require.NoError(t, err)
	require.Empty(t, res)
}

func TestSearch_DBError(t *testing.T) {
	db := &MockDB{
		GetComicsDataFunc: func(ctx context.Context) ([]ComicsKeyWords, error) {
			return nil, errors.New("db fail")
		},
	}

	words := &MockWords{
		NormFunc: func(ctx context.Context, s string) ([]string, error) {
			return []string{"linux"}, nil
		},
	}

	svc, _ := NewService(slog.Default(), db, words)

	_, err := svc.Search(context.Background(), SearchRequest{})

	require.Error(t, err)
}

func TestSearch_ImageURLError(t *testing.T) {
	db := &MockDB{
		GetComicsDataFunc: func(ctx context.Context) ([]ComicsKeyWords, error) {
			return []ComicsKeyWords{
				{ID: 1, KeyWords: []string{"linux"}},
			}, nil
		},
		GetImageURLFunc: func(ctx context.Context, id int) (string, error) {
			return "", errors.New("get image URL fail")
		},
	}

	words := &MockWords{
		NormFunc: func(ctx context.Context, s string) ([]string, error) {
			return []string{"linux"}, nil
		},
	}

	svc, _ := NewService(slog.Default(), db, words)

	_, err := svc.Search(context.Background(), SearchRequest{})

	require.Error(t, err)
}
