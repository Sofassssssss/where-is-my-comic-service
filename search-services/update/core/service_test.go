package core

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

type MockDB struct {
	AddFunc   func(ctx context.Context, comics Comics) error
	IDsFunc   func(ctx context.Context) ([]int, error)
	StatsFunc func(ctx context.Context) (DBStats, error)
	DropFunc  func(ctx context.Context) error
}

func (m *MockDB) Add(ctx context.Context, comics Comics) error { return m.AddFunc(ctx, comics) }
func (m *MockDB) IDs(ctx context.Context) ([]int, error)       { return m.IDsFunc(ctx) }
func (m *MockDB) Stats(ctx context.Context) (DBStats, error)   { return m.StatsFunc(ctx) }
func (m *MockDB) Drop(ctx context.Context) error               { return m.DropFunc(ctx) }

type MockXKCD struct {
	GetFunc    func(ctx context.Context, ID int) (XKCDInfo, error)
	LastIDFunc func(ctx context.Context) (int, error)
}

func (m *MockXKCD) Get(ctx context.Context, id int) (XKCDInfo, error) { return m.GetFunc(ctx, id) }
func (m *MockXKCD) LastID(ctx context.Context) (int, error)           { return m.LastIDFunc(ctx) }

type MockWords struct {
	NormLeaveDuplicatesFunc func(ctx context.Context, phrase string) ([]string, error)
}

func (m *MockWords) NormLeaveDuplicates(ctx context.Context, phrase string) ([]string, error) {
	return m.NormLeaveDuplicatesFunc(ctx, phrase)
}

type MockPublisher struct {
	PublishUpdateFunc func(ctx context.Context) error
}

func (m *MockPublisher) PublishUpdate(ctx context.Context) error {
	return m.PublishUpdateFunc(ctx)
}

func TestNewService(t *testing.T) {
	svc, err := NewService(slog.Default(), &MockDB{}, &MockXKCD{}, &MockWords{}, &MockPublisher{}, 5)
	require.NoError(t, err)
	require.NotNil(t, svc)
	require.Equal(t, 5, svc.concurrency)

	_, err = NewService(slog.Default(), &MockDB{}, &MockXKCD{}, &MockWords{}, &MockPublisher{}, 0)
	require.Error(t, err)
}

func TestGetComicsToInsert(t *testing.T) {
	tests := []struct {
		name              string
		comicsAmount      int
		insertedComicsIDs []int
		expected          []int
	}{
		{
			name:              "all new",
			comicsAmount:      3,
			insertedComicsIDs: []int{},
			expected:          []int{1, 2, 3},
		},
		{
			name:              "some exist",
			comicsAmount:      5,
			insertedComicsIDs: []int{2, 4},
			expected:          []int{1, 3, 5},
		},
		{
			name:              "all exist",
			comicsAmount:      3,
			insertedComicsIDs: []int{1, 2, 3},
			expected:          []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := getComicsToInsert(tt.comicsAmount, tt.insertedComicsIDs)
			require.Equal(t, tt.expected, res)
		})
	}
}

func TestService_Update_Success(t *testing.T) {
	db := &MockDB{
		IDsFunc: func(ctx context.Context) ([]int, error) {
			return []int{1}, nil
		},
		AddFunc: func(ctx context.Context, comics Comics) error {
			return nil
		},
	}

	xkcd := &MockXKCD{
		LastIDFunc: func(ctx context.Context) (int, error) {
			return 3, nil
		},
		GetFunc: func(ctx context.Context, ID int) (XKCDInfo, error) {
			return XKCDInfo{ID: ID, Title: "Test", URL: "url1"}, nil
		},
	}

	words := &MockWords{
		NormLeaveDuplicatesFunc: func(ctx context.Context, phrase string) ([]string, error) {
			return []string{"test"}, nil
		},
	}

	pub := &MockPublisher{
		PublishUpdateFunc: func(ctx context.Context) error {
			return nil
		},
	}

	svc, _ := NewService(slog.Default(), db, xkcd, words, pub, 2)

	updateRes, err := svc.Update(context.Background())
	require.NoError(t, err)

	require.Equal(t, 2, updateRes.ComicsInserted)
	require.Equal(t, int32(0), svc.isBusy.Load())
}

func TestService_Update_AlreadyRunning(t *testing.T) {
	svc, _ := NewService(slog.Default(), &MockDB{}, &MockXKCD{}, &MockWords{}, &MockPublisher{}, 1)

	svc.isBusy.Store(1)

	_, err := svc.Update(context.Background())
	require.ErrorIs(t, err, ErrUpdateRunning)
}

func TestService_Update_ErrorsIgnoredByWorkers(t *testing.T) {

	db := &MockDB{
		IDsFunc: func(ctx context.Context) ([]int, error) { return []int{}, nil },
		AddFunc: func(ctx context.Context, comics Comics) error {
			if comics.ID == 2 {
				return errors.New("db error")
			}
			return nil
		},
	}

	xkcd := &MockXKCD{
		LastIDFunc: func(ctx context.Context) (int, error) { return 2, nil },
		GetFunc: func(ctx context.Context, ID int) (XKCDInfo, error) {
			return XKCDInfo{ID: ID, Title: "Test", URL: "url1"}, nil
		},
	}

	words := &MockWords{
		NormLeaveDuplicatesFunc: func(ctx context.Context, phrase string) ([]string, error) {
			return []string{"test"}, nil
		},
	}

	pub := &MockPublisher{
		PublishUpdateFunc: func(ctx context.Context) error { return nil },
	}

	svc, _ := NewService(slog.Default(), db, xkcd, words, pub, 2)

	updateRes, err := svc.Update(context.Background())
	require.NoError(t, err)

	require.Equal(t, 1, updateRes.ComicsInserted)
}

func TestService_Stats(t *testing.T) {
	db := &MockDB{
		StatsFunc: func(ctx context.Context) (DBStats, error) {
			return DBStats{}, nil
		},
	}
	xkcd := &MockXKCD{
		LastIDFunc: func(ctx context.Context) (int, error) {
			return 42, nil
		},
	}

	svc, _ := NewService(slog.Default(), db, xkcd, &MockWords{}, &MockPublisher{}, 1)

	stats, err := svc.Stats(context.Background())
	require.NoError(t, err)
	require.Equal(t, 42, stats.ComicsTotal)
}

func TestService_Status(t *testing.T) {
	svc, _ := NewService(slog.Default(), &MockDB{}, &MockXKCD{}, &MockWords{}, &MockPublisher{}, 1)

	require.Equal(t, StatusIdle, svc.Status(context.Background()))

	svc.isBusy.Store(1)
	require.Equal(t, StatusRunning, svc.Status(context.Background()))
}

func TestService_Drop(t *testing.T) {
	db := &MockDB{
		DropFunc: func(ctx context.Context) error { return nil },
	}
	pub := &MockPublisher{
		PublishUpdateFunc: func(ctx context.Context) error { return nil },
	}

	svc, _ := NewService(slog.Default(), db, &MockXKCD{}, &MockWords{}, pub, 1)

	err := svc.Drop(context.Background())
	require.NoError(t, err)
}
