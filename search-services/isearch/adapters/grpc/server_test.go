package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"

	"where-is-my-comic-service/search-services/isearch/core"
	isearchpb "where-is-my-comic-service/search-services/proto/isearch"
)

type MockISearcher struct {
	ISearchFunc func(ctx context.Context, req core.ISearchRequest) ([]core.Comics, error)
}

func (m *MockISearcher) ISearch(ctx context.Context, req core.ISearchRequest) ([]core.Comics, error) {
	if m.ISearchFunc != nil {
		return m.ISearchFunc(ctx, req)
	}
	return nil, nil
}

func TestServer_Ping(t *testing.T) {
	srv := NewServer(&MockISearcher{})

	res, err := srv.Ping(context.Background(), &emptypb.Empty{})

	require.NoError(t, err)
	require.Nil(t, res)
}

func TestServer_ISearch(t *testing.T) {
	tests := []struct {
		name          string
		inPhrase      string
		inLimit       int32
		mockReturn    []core.Comics
		mockErr       error
		expectError   bool
		expectedCount int
	}{
		{
			name:     "success search",
			inPhrase: "linux",
			inLimit:  10,
			mockReturn: []core.Comics{
				{ID: 1, URL: "http://xkcd.com/comics/test_image.png"},
				{ID: 2, URL: "http://xkcd.com/comics/test_image_1.png"},
			},
			mockErr:       nil,
			expectError:   false,
			expectedCount: 2,
		},
		{
			name:          "core service error",
			inPhrase:      "error-trigger",
			inLimit:       5,
			mockReturn:    nil,
			mockErr:       errors.New("some core error"),
			expectError:   true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockSearcher := &MockISearcher{
				ISearchFunc: func(ctx context.Context, req core.ISearchRequest) ([]core.Comics, error) {
					require.Equal(t, tt.inPhrase, req.Phrase)
					require.Equal(t, int(tt.inLimit), req.Limit)
					return tt.mockReturn, tt.mockErr
				},
			}

			srv := NewServer(mockSearcher)

			req := &isearchpb.ISearchRequest{
				Phrase: tt.inPhrase,
				Limit:  tt.inLimit,
			}

			res, err := srv.ISearch(context.Background(), req)

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, res)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.Len(t, res.Comics, tt.expectedCount)
			}
		})
	}
}
