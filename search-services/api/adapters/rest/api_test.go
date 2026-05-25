package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"where-is-my-comic-service/search-services/api/core"
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

type MockPinger struct {
	PingFunc func(ctx context.Context) error
}

func (m *MockPinger) Ping(ctx context.Context) error {
	if m.PingFunc != nil {
		return m.PingFunc(ctx)
	}
	return nil
}

type MockAuthenticator struct {
	LoginFunc func(user, password string) (string, error)
}

func (m *MockAuthenticator) Login(user, password string) (string, error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(user, password)
	}
	return "", nil
}

type MockUpdater struct {
	UpdateFunc func(ctx context.Context) (core.UpdateResult, error)
	StatsFunc  func(ctx context.Context) (core.UpdateStats, error)
	StatusFunc func(ctx context.Context) (core.UpdateStatus, error)
	DropFunc   func(ctx context.Context) error
}

func (m *MockUpdater) Update(ctx context.Context) (core.UpdateResult, error) {
	return m.UpdateFunc(ctx)
}
func (m *MockUpdater) Stats(ctx context.Context) (core.UpdateStats, error) { return m.StatsFunc(ctx) }
func (m *MockUpdater) Status(ctx context.Context) (core.UpdateStatus, error) {
	return m.StatusFunc(ctx)
}
func (m *MockUpdater) Drop(ctx context.Context) error { return m.DropFunc(ctx) }

type MockSearcher struct {
	SearchFunc func(ctx context.Context, req core.SearchRequest) ([]core.Comics, error)
}

func (m *MockSearcher) Search(ctx context.Context, req core.SearchRequest) ([]core.Comics, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, req)
	}
	return nil, nil
}

func TestNewPingHandler(t *testing.T) {
	pingers := map[string]core.Pinger{
		"service_ok": &MockPinger{
			PingFunc: func(ctx context.Context) error { return nil },
		},
		"service_fail": &MockPinger{
			PingFunc: func(ctx context.Context) error { return errors.New("timeout") },
		},
	}

	handler := NewPingHandler(slog.Default(), pingers)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp PingResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	require.Equal(t, "ok", resp.Replies["service_ok"])
	require.Equal(t, "unavailable", resp.Replies["service_fail"])
}

func TestNewLoginHandler(t *testing.T) {
	authMock := &MockAuthenticator{
		LoginFunc: func(user, password string) (string, error) {
			if user == "admin" && password == "admin" {
				return "valid-token", nil
			}
			return "", errors.New("invalid credentials")
		},
	}
	handler := NewLoginHandler(slog.Default(), authMock)

	t.Run("success", func(t *testing.T) {
		body := `{"name": "admin", "password": "admin"}`
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "text/plain", w.Header().Get("Content-Type"))
		require.Equal(t, "valid-token", w.Body.String())
	})

	t.Run("bad json", func(t *testing.T) {
		body := `{"name": "admin", "password": `
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestNewUpdateHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		updater := &MockUpdater{
			UpdateFunc: func(ctx context.Context) (core.UpdateResult, error) {
				return core.UpdateResult{ComicsInserted: 5}, nil
			},
		}
		handler := NewUpdateHandler(slog.Default(), updater)

		req := httptest.NewRequest(http.MethodPost, "/update", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp UpdateResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)
		require.Equal(t, 5, resp.ComicsInserted)
	})
}

func TestNewDropHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		updater := &MockUpdater{
			DropFunc: func(ctx context.Context) error {
				return nil
			},
		}
		handler := NewDropHandler(slog.Default(), updater)

		req := httptest.NewRequest(http.MethodDelete, "/drop", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})
}

func TestNewISearchHandler(t *testing.T) {
	isearcher := &MockISearcher{
		ISearchFunc: func(ctx context.Context, req core.ISearchRequest) ([]core.Comics, error) {
			if req.Phrase == "error-trigger" {
				return nil, errors.New("isearch failed")
			}
			return []core.Comics{{ID: 1, URL: "url1"}}, nil
		},
	}
	handler := NewISearchHandler(slog.Default(), isearcher)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		checkBody      func(t *testing.T, body *bytes.Buffer)
	}{
		{
			name:           "success search with default limit",
			url:            "/isearch?phrase=linux",
			expectedStatus: http.StatusOK,
			checkBody: func(t *testing.T, body *bytes.Buffer) {
				var resp ISearchResponse
				require.NoError(t, json.NewDecoder(body).Decode(&resp))
				require.Equal(t, 1, resp.Total)
				require.Len(t, resp.Comics, 1)
			},
		},
		{
			name:           "success search with custom limit",
			url:            "/isearch?phrase=linux&limit=5",
			expectedStatus: http.StatusOK,
			checkBody:      func(t *testing.T, body *bytes.Buffer) {},
		},
		{
			name:           "empty phrase",
			url:            "/isearch",
			expectedStatus: http.StatusBadRequest,
			checkBody:      func(t *testing.T, body *bytes.Buffer) {},
		},
		{
			name:           "invalid limit format",
			url:            "/isearch?phrase=linux&limit=abc",
			expectedStatus: http.StatusBadRequest,
			checkBody:      func(t *testing.T, body *bytes.Buffer) {},
		},
		{
			name:           "invalid limit zero",
			url:            "/isearch?phrase=linux&limit=0",
			expectedStatus: http.StatusBadRequest,
			checkBody:      func(t *testing.T, body *bytes.Buffer) {},
		},
		{
			name:           "core logic error",
			url:            "/isearch?phrase=error-trigger",
			expectedStatus: http.StatusInternalServerError,
			checkBody:      func(t *testing.T, body *bytes.Buffer) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)
			tt.checkBody(t, w.Body)
		})
	}
}

func TestNewUpdateStatsHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		updater := &MockUpdater{
			StatsFunc: func(ctx context.Context) (core.UpdateStats, error) {
				return core.UpdateStats{
					WordsTotal:    450,
					WordsUnique:   340,
					ComicsFetched: 200,
					ComicsTotal:   200,
				}, nil
			},
		}
		handler := NewUpdateStatsHandler(slog.Default(), updater)

		req := httptest.NewRequest(http.MethodGet, "/stats", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp StatsResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)

		require.Equal(t, 450, resp.WordsTotal)
		require.Equal(t, 340, resp.WordsUnique)
		require.Equal(t, 200, resp.ComicsFetched)
		require.Equal(t, 200, resp.ComicsTotal)
	})

	t.Run("core error", func(t *testing.T) {
		updater := &MockUpdater{
			StatsFunc: func(ctx context.Context) (core.UpdateStats, error) {
				return core.UpdateStats{}, errors.New("db offline")
			},
		}
		handler := NewUpdateStatsHandler(slog.Default(), updater)

		req := httptest.NewRequest(http.MethodGet, "/stats", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		require.NotEqual(t, http.StatusOK, w.Code)
	})
}

func TestNewUpdateStatusHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		updater := &MockUpdater{
			StatusFunc: func(ctx context.Context) (core.UpdateStatus, error) {
				return core.StatusUpdateRunning, nil
			},
		}
		handler := NewUpdateStatusHandler(slog.Default(), updater)

		req := httptest.NewRequest(http.MethodGet, "/status", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var resp StatusResponse
		err := json.NewDecoder(w.Body).Decode(&resp)
		require.NoError(t, err)

		require.Equal(t, string(core.StatusUpdateRunning), resp.Status)
	})

	t.Run("core error", func(t *testing.T) {
		updater := &MockUpdater{
			StatusFunc: func(ctx context.Context) (core.UpdateStatus, error) {
				return "", errors.New("unexpected error")
			},
		}
		handler := NewUpdateStatusHandler(slog.Default(), updater)

		req := httptest.NewRequest(http.MethodGet, "/status", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		require.NotEqual(t, http.StatusOK, w.Code)
	})
}
