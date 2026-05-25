package initiator

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type MockIndexBuilder struct {
	mu               sync.Mutex
	calls            int
	RebuildIndexFunc func(ctx context.Context) error
}

func (m *MockIndexBuilder) RebuildIndex(ctx context.Context) error {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()

	if m.RebuildIndexFunc != nil {
		return m.RebuildIndexFunc(ctx)
	}
	return nil
}

func (m *MockIndexBuilder) getCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func TestInitiator_InitInitial(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func() *MockIndexBuilder
		expectError bool
	}{
		{
			name: "success build",
			setupMock: func() *MockIndexBuilder {
				return &MockIndexBuilder{
					RebuildIndexFunc: func(ctx context.Context) error {
						return nil
					},
				}
			},
			expectError: false,
		},
		{
			name: "error build",
			setupMock: func() *MockIndexBuilder {
				return &MockIndexBuilder{
					RebuildIndexFunc: func(ctx context.Context) error {
						return errors.New("builder failed")
					},
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockBuilder := tt.setupMock()
			init := New(mockBuilder, time.Minute, slog.Default())

			err := init.InitInitial(context.Background())

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, 1, mockBuilder.getCalls())
		})
	}
}

func TestInitiator_Start_ContextCancellation(t *testing.T) {
	mockBuilder := &MockIndexBuilder{}

	init := New(mockBuilder, time.Hour, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())

	cancel()

	init.Start(ctx)

	require.Equal(t, 0, mockBuilder.getCalls())
}

func TestInitiator_Start_TimerTicks(t *testing.T) {
	mockBuilder := &MockIndexBuilder{}

	shortTTL := 10 * time.Millisecond
	init := New(mockBuilder, shortTTL, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		init.Start(ctx)
	}()

	time.Sleep(35 * time.Millisecond)

	cancel()

	wg.Wait()

	calls := mockBuilder.getCalls()
	require.GreaterOrEqual(t, calls, 1, "timer should have triggered RebuildIndex at least once")
}

func TestInitiator_Start_TimerTickWithErrorIgnored(t *testing.T) {
	mockBuilder := &MockIndexBuilder{
		RebuildIndexFunc: func(ctx context.Context) error {
			return errors.New("simulated tick error")
		},
	}

	shortTTL := 10 * time.Millisecond
	init := New(mockBuilder, shortTTL, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		init.Start(ctx)
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()
	wg.Wait()

	require.GreaterOrEqual(t, mockBuilder.getCalls(), 1)
}
