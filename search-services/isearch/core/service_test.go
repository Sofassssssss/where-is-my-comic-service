package core

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

type MockDB struct {
	GetComicsDataFunc func(ctx context.Context) ([]ComicsKeyWords, error)
	GetImageURLFunc   func(ctx context.Context, id int) (string, error)
}

func (m *MockDB) GetComicsData(ctx context.Context) ([]ComicsKeyWords, error) {
	if m.GetComicsDataFunc != nil {
		return m.GetComicsDataFunc(ctx)
	}
	return nil, nil
}

func (m *MockDB) GetImageURL(ctx context.Context, id int) (string, error) {
	if m.GetImageURLFunc != nil {
		return m.GetImageURLFunc(ctx, id)
	}
	return "", nil
}

type MockWords struct {
	NormFunc func(ctx context.Context, phrase string) ([]string, error)
}

func (m *MockWords) Norm(ctx context.Context, phrase string) ([]string, error) {
	if m.NormFunc != nil {
		return m.NormFunc(ctx, phrase)
	}
	return nil, nil
}

func TestRemoveDuplicates(t *testing.T) {
	input := []string{"linux", "apple", "linux", "banana", "apple"}
	expected := []string{"apple", "banana", "linux"}

	result := removeDuplicates(input)
	require.Equal(t, expected, result)
}

func TestTokenizer(t *testing.T) {
	input := "hello world from go"
	expected := []string{"hello", "world", "from", "go"}

	result := tokenizer(input)
	require.Equal(t, expected, result)
}

func TestBuildSearchIndex(t *testing.T) {
	comicsData := []ComicsKeyWords{
		{ID: 1, KeyWords: []string{"linux", "world", "love", "linux", "orange"}},
		{ID: 2, KeyWords: []string{"tree", "orange"}},
	}

	index, err := buildSearchIndex(comicsData)
	require.NoError(t, err)
	require.NotNil(t, index)

	require.Len(t, index.InvertedIndex["linux"], 1)
	require.Equal(t, 1, index.InvertedIndex["linux"][0].DocID)
	require.Equal(t, 2, index.InvertedIndex["linux"][0].TF)

	require.Len(t, index.InvertedIndex["orange"], 2)

	require.Equal(t, 2, index.TotalDocs)
	require.Len(t, index.DocLengths, 2)
	require.Greater(t, index.AvgDocLength, 0.0)
	require.Contains(t, index.IDF, "linux")
}

func TestBuildSearchIndex_Empty(t *testing.T) {
	index, err := buildSearchIndex([]ComicsKeyWords{})
	require.NoError(t, err)
	require.Empty(t, index.InvertedIndex)
	require.Empty(t, index.IDF)
}

func TestService_RebuildIndex_Success(t *testing.T) {
	db := &MockDB{
		GetComicsDataFunc: func(ctx context.Context) ([]ComicsKeyWords, error) {
			return []ComicsKeyWords{{ID: 1, KeyWords: []string{"test"}}}, nil
		},
	}
	svc, _ := NewService(slog.Default(), db, &MockWords{})

	err := svc.RebuildIndex(context.Background())
	require.NoError(t, err)

	loadedIndex := svc.index.Load()
	require.NotNil(t, loadedIndex)
	require.Contains(t, loadedIndex.InvertedIndex, "test")
}

func TestService_RebuildIndex_Concurrent(t *testing.T) {
	db := &MockDB{
		GetComicsDataFunc: func(ctx context.Context) ([]ComicsKeyWords, error) {
			return []ComicsKeyWords{{ID: 1, KeyWords: []string{"test"}}}, nil
		},
	}
	svc, _ := NewService(slog.Default(), db, &MockWords{})

	svc.rebuilding.Store(true)

	err := svc.RebuildIndex(context.Background())
	require.NoError(t, err)

	require.Nil(t, svc.index.Load())
}

func TestService_RebuildIndex_DBError(t *testing.T) {
	db := &MockDB{
		GetComicsDataFunc: func(ctx context.Context) ([]ComicsKeyWords, error) {
			return nil, errors.New("db error")
		},
	}
	svc, _ := NewService(slog.Default(), db, &MockWords{})

	err := svc.RebuildIndex(context.Background())
	require.Error(t, err)
	require.Nil(t, svc.index.Load())
}

func TestService_ISearch_IndexNotReady(t *testing.T) {
	words := &MockWords{
		NormFunc: func(ctx context.Context, phrase string) ([]string, error) {
			return []string{"test"}, nil
		},
	}
	svc, _ := NewService(slog.Default(), &MockDB{}, words)

	res, err := svc.ISearch(context.Background(), ISearchRequest{Phrase: "test", Limit: 10})

	require.ErrorIs(t, err, ErrIndexNotReady)
	require.Nil(t, res)
}

func TestService_ISearch_Success(t *testing.T) {
	db := &MockDB{
		GetImageURLFunc: func(ctx context.Context, id int) (string, error) {
			return "https://imgs.xkcd.com/comics/test_image.png", nil
		},
	}
	words := &MockWords{
		NormFunc: func(ctx context.Context, phrase string) ([]string, error) {
			return []string{phrase}, nil
		},
	}
	svc, _ := NewService(slog.Default(), db, words)

	comicsData := []ComicsKeyWords{
		{ID: 1, KeyWords: []string{"linux", "kernel", "system"}},
		{ID: 2, KeyWords: []string{"windows", "kernel"}},
		{ID: 3, KeyWords: []string{"apple", "mac", "os"}},
		{ID: 4, KeyWords: []string{"android", "mobile", "phone"}},
		{ID: 5, KeyWords: []string{"ubuntu", "debian", "linux"}},
	}
	index, _ := buildSearchIndex(comicsData)
	svc.index.Store(index)

	req := ISearchRequest{Phrase: "kernel", Limit: 1}
	res, err := svc.ISearch(context.Background(), req)

	require.NoError(t, err)

	require.Len(t, res, 1)

	require.Equal(t, int64(2), res[0].ID)
	require.Equal(t, "https://imgs.xkcd.com/comics/test_image.png", res[0].URL)
}

func TestService_ISearch_WordsError(t *testing.T) {
	words := &MockWords{
		NormFunc: func(ctx context.Context, phrase string) ([]string, error) {
			return nil, errors.New("norm fail")
		},
	}
	svc, _ := NewService(slog.Default(), &MockDB{}, words)

	_, err := svc.ISearch(context.Background(), ISearchRequest{Phrase: "test"})
	require.Error(t, err)
}

func TestService_ISearch_DBErrorOnURL(t *testing.T) {
	db := &MockDB{
		GetImageURLFunc: func(ctx context.Context, id int) (string, error) {
			return "", errors.New("db url fail")
		},
	}
	words := &MockWords{
		NormFunc: func(ctx context.Context, phrase string) ([]string, error) {
			return []string{"linux"}, nil
		},
	}
	svc, _ := NewService(slog.Default(), db, words)

	index, _ := buildSearchIndex([]ComicsKeyWords{{ID: 1, KeyWords: []string{"linux"}}})
	svc.index.Store(index)

	_, err := svc.ISearch(context.Background(), ISearchRequest{Phrase: "linux", Limit: 10})
	require.Error(t, err)
}
