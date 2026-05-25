package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"where-is-my-comic-service/search-services/api/core"
)

type MockVerifier struct {
	VerifyFunc func(token string) error
}

func (m *MockVerifier) Verify(token string) error {
	if m.VerifyFunc != nil {
		return m.VerifyFunc(token)
	}
	return nil
}

func TestAuth(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	})

	tests := []struct {
		name           string
		authHeader     string
		setupMock      func() *MockVerifier
		expectedStatus int
		expectNext     bool
	}{
		{
			name:       "missing authorization header",
			authHeader: "",
			setupMock: func() *MockVerifier {
				return &MockVerifier{}
			},
			expectedStatus: http.StatusUnauthorized,
			expectNext:     false,
		},
		{
			name:       "invalid token prefix",
			authHeader: "Bearer some-token",
			setupMock: func() *MockVerifier {
				return &MockVerifier{}
			},
			expectedStatus: http.StatusUnauthorized,
			expectNext:     false,
		},
		{
			name:       "verifier returns error",
			authHeader: "Token invalid-token",
			setupMock: func() *MockVerifier {
				return &MockVerifier{
					VerifyFunc: func(token string) error {
						require.Equal(t, "invalid-token", token)
						return core.ErrUnauthorized
					},
				}
			},

			expectedStatus: http.StatusUnauthorized,
			expectNext:     false,
		},
		{
			name:       "success authorization",
			authHeader: "Token valid-token",
			setupMock: func() *MockVerifier {
				return &MockVerifier{
					VerifyFunc: func(token string) error {
						require.Equal(t, "valid-token", token)
						return nil
					},
				}
			},
			expectedStatus: http.StatusOK,
			expectNext:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockVerifier := tt.setupMock()
			handler := Auth(nextHandler, mockVerifier)

			req := httptest.NewRequest(http.MethodGet, "/protected-route", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			require.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectNext {
				require.Equal(t, "success", w.Body.String())
			} else {
				require.NotContains(t, w.Body.String(), "success")
			}
		})
	}
}
