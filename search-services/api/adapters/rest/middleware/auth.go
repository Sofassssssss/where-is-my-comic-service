package middleware

import (
	"net/http"
	"strings"

	"where-is-my-comic-service/search-services/api/adapters/rest"
)

type TokenVerifier interface {
	Verify(token string) error
}

func Auth(next http.HandlerFunc, verifier TokenVerifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}
		if !strings.HasPrefix(authHeader, "Token ") {
			http.Error(w, "invalid authorization header", http.StatusUnauthorized)
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Token ")
		err := verifier.Verify(tokenString)
		if err != nil {
			httpStatus, message := rest.HttpError(err)
			http.Error(w, message, httpStatus)
			return
		}
		next(w, r)
	}
}
