package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	searchpb "where-is-my-comic-service/search-services/proto/search"
	"where-is-my-comic-service/search-services/search/core"
)

type MockSearcher struct {
	SearchFunc func(ctx context.Context, req core.SearchRequest) ([]core.Comics, error)
}

func (m *MockSearcher) Search(ctx context.Context, req core.SearchRequest) ([]core.Comics, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, req)
	}
	return nil, nil
}

func TestServer_Ping(t *testing.T) {
	srv := NewServer(&MockSearcher{})

	res, err := srv.Ping(context.Background(), &emptypb.Empty{})

	require.NoError(t, err)
	require.Nil(t, res)
}

func TestServer_Search(t *testing.T) {
	tests := []struct {
		name          string
		inPhrase      string
		inLimit       int32
		mockReturn    []core.Comics
		mockErr       error
		expectedCode  codes.Code
		expectedCount int
	}{
		{
			name:          "success search",
			inPhrase:      "linux",
			inLimit:       10,
			mockReturn:    []core.Comics{{ID: 1, URL: "url1"}, {ID: 2, URL: "url2"}},
			mockErr:       nil,
			expectedCode:  codes.OK,
			expectedCount: 2,
		},
		{
			name:          "core service error",
			inPhrase:      "error-trigger",
			inLimit:       5,
			mockReturn:    nil,
			mockErr:       errors.New("something went wrong in core"),
			expectedCode:  codes.Internal,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockSearcher := &MockSearcher{
				SearchFunc: func(ctx context.Context, req core.SearchRequest) ([]core.Comics, error) {
					require.Equal(t, tt.inPhrase, req.Phrase)
					require.Equal(t, int(tt.inLimit), req.Limit)
					return tt.mockReturn, tt.mockErr
				},
			}

			srv := NewServer(mockSearcher)

			req := &searchpb.SearchRequest{
				Phrase: tt.inPhrase,
				Limit:  tt.inLimit,
			}

			res, err := srv.Search(context.Background(), req)

			if tt.expectedCode == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, res)

				require.Len(t, res.Comics, tt.expectedCount)
			} else {
				require.Error(t, err)
				require.Nil(t, res)

				st, ok := status.FromError(err)
				require.True(t, ok, "error should be a grpc status error")
				require.Equal(t, tt.expectedCode, st.Code())
			}
		})
	}
}
