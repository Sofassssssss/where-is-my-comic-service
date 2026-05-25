package aaa

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"where-is-my-comic-service/search-services/api/core"

	"github.com/golang-jwt/jwt"
)

const secretKey = "something secret here" // token sign key
const adminRole = "superuser"             // token subject

// AAA Authentication, Authorization, Accounting
type AAA struct {
	users    map[string]string
	tokenTTL time.Duration
	log      *slog.Logger
}

func New(tokenTTL time.Duration, log *slog.Logger) (AAA, error) {
	const adminUser = "ADMIN_USER"
	const adminPass = "ADMIN_PASSWORD"
	user, ok := os.LookupEnv(adminUser)
	if !ok {
		return AAA{}, fmt.Errorf("could not get admin user from enviroment")
	}
	password, ok := os.LookupEnv(adminPass)
	if !ok {
		return AAA{}, fmt.Errorf("could not get admin password from enviroment")
	}

	return AAA{
		users:    map[string]string{user: password},
		tokenTTL: tokenTTL,
		log:      log,
	}, nil
}

func (a AAA) Login(name, password string) (string, error) {
	if value, ok := a.users[name]; !ok || value != password {
		return "", core.ErrUnauthorized
	} else {
		claims := jwt.MapClaims{
			"sub": adminRole,
			"exp": time.Now().Add(a.tokenTTL).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(secretKey))
		if err != nil {
			return "", err
		}
		return tokenString, nil
	}
}

func (a AAA) Verify(tokenString string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		var ve *jwt.ValidationError
		if errors.As(err, &ve) {
			if ve.Errors&jwt.ValidationErrorExpired != 0 {
				return core.ErrUnauthorized
			}
		}
		return err
	}
	if !token.Valid {
		return core.ErrUnauthorized
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return core.ErrUnauthorized
	}
	if claims["sub"] != adminRole {
		return core.ErrUnauthorized
	}
	return nil
}
