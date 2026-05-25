package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	updatepb "where-is-my-comic-service/search-services/proto/update"
	"where-is-my-comic-service/search-services/update/core"
)

type MockUpdater struct {
	StatusFunc func(ctx context.Context) core.ServiceStatus
	UpdateFunc func(ctx context.Context) (core.Update, error)
	StatsFunc  func(ctx context.Context) (core.ServiceStats, error)
	DropFunc   func(ctx context.Context) error
}

func (m *MockUpdater) Status(ctx context.Context) core.ServiceStatus {
	if m.StatusFunc != nil {
		return m.StatusFunc(ctx)
	}
	return core.StatusIdle
}

func (m *MockUpdater) Update(ctx context.Context) (core.Update, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx)
	}
	return core.Update{}, nil
}

func (m *MockUpdater) Stats(ctx context.Context) (core.ServiceStats, error) {
	if m.StatsFunc != nil {
		return m.StatsFunc(ctx)
	}
	return core.ServiceStats{}, nil
}

func (m *MockUpdater) Drop(ctx context.Context) error {
	if m.DropFunc != nil {
		return m.DropFunc(ctx)
	}
	return nil
}

func TestServer_Ping(t *testing.T) {
	srv := NewServer(&MockUpdater{})
	res, err := srv.Ping(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.Nil(t, res)
}

func TestServer_Status(t *testing.T) {
	mockUpdater := &MockUpdater{
		StatusFunc: func(ctx context.Context) core.ServiceStatus {
			return core.StatusRunning
		},
	}
	srv := NewServer(mockUpdater)

	res, err := srv.Status(context.Background(), &emptypb.Empty{})

	require.NoError(t, err)
	require.NotNil(t, res)
}

func TestServer_Update(t *testing.T) {
	tests := []struct {
		name         string
		setupMock    func() *MockUpdater
		expectedCode codes.Code
		expectedRes  *updatepb.UpdateReply
	}{
		{
			name: "already running",
			setupMock: func() *MockUpdater {
				return &MockUpdater{
					StatusFunc: func(ctx context.Context) core.ServiceStatus {
						return core.StatusRunning
					},
				}
			},
			expectedCode: codes.Unavailable,
		},
		{
			name: "core service error",
			setupMock: func() *MockUpdater {
				return &MockUpdater{
					StatusFunc: func(ctx context.Context) core.ServiceStatus {
						return core.StatusIdle
					},
					UpdateFunc: func(ctx context.Context) (core.Update, error) {
						return core.Update{}, errors.New("core error")
					},
				}
			},
			expectedCode: codes.Internal,
		},
		{
			name: "success update",
			setupMock: func() *MockUpdater {
				return &MockUpdater{
					StatusFunc: func(ctx context.Context) core.ServiceStatus {
						return core.StatusIdle
					},
					UpdateFunc: func(ctx context.Context) (core.Update, error) {
						return core.Update{ComicsInserted: 42}, nil
					},
				}
			},
			expectedCode: codes.OK,
			expectedRes:  &updatepb.UpdateReply{ComicsInserted: 42},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(tt.setupMock())
			res, err := srv.Update(context.Background(), &emptypb.Empty{})

			if tt.expectedCode == codes.OK {
				require.NoError(t, err)
				require.Equal(t, tt.expectedRes.ComicsInserted, res.ComicsInserted)
			} else {
				require.Error(t, err)
				require.Nil(t, res)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, tt.expectedCode, st.Code())
			}
		})
	}
}

func TestServer_Stats(t *testing.T) {
	tests := []struct {
		name         string
		setupMock    func() *MockUpdater
		expectedCode codes.Code
		expectedRes  *updatepb.StatsReply
	}{
		{
			name: "core service error",
			setupMock: func() *MockUpdater {
				return &MockUpdater{
					StatsFunc: func(ctx context.Context) (core.ServiceStats, error) {
						return core.ServiceStats{}, errors.New("db error")
					},
				}
			},
			expectedCode: codes.Internal,
		},
		{
			name: "success stats",
			setupMock: func() *MockUpdater {
				return &MockUpdater{
					StatsFunc: func(ctx context.Context) (core.ServiceStats, error) {
						return core.ServiceStats{
							DBStats: core.DBStats{
								WordsTotal:    450,
								WordsUnique:   340,
								ComicsFetched: 200,
							},
							ComicsTotal: 200,
						}, nil
					},
				}
			},
			expectedCode: codes.OK,
			expectedRes: &updatepb.StatsReply{
				WordsTotal:    450,
				WordsUnique:   340,
				ComicsFetched: 200,
				ComicsTotal:   200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(tt.setupMock())
			res, err := srv.Stats(context.Background(), &emptypb.Empty{})

			if tt.expectedCode == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.Equal(t, tt.expectedRes.WordsTotal, res.WordsTotal)
				require.Equal(t, tt.expectedRes.WordsUnique, res.WordsUnique)
				require.Equal(t, tt.expectedRes.ComicsFetched, res.ComicsFetched)
				require.Equal(t, tt.expectedRes.ComicsTotal, res.ComicsTotal)
			} else {
				require.Error(t, err)
				require.Nil(t, res)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, tt.expectedCode, st.Code())
			}
		})
	}
}

func TestServer_Drop(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func() *MockUpdater
		expectError bool
	}{
		{
			name: "success drop",
			setupMock: func() *MockUpdater {
				return &MockUpdater{
					DropFunc: func(ctx context.Context) error {
						return nil
					},
				}
			},
			expectError: false,
		},
		{
			name: "error drop",
			setupMock: func() *MockUpdater {
				return &MockUpdater{
					DropFunc: func(ctx context.Context) error {
						return errors.New("failed to drop")
					},
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(tt.setupMock())
			res, err := srv.Drop(context.Background(), &emptypb.Empty{})

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
			}
		})
	}
}
