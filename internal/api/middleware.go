package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const (
	UserIDKey  ctxKey = "userID"
	GroupIDKey ctxKey = "groupID"
)

func AuthMiddleware(secret string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing auth header", http.StatusUnauthorized)
				return
			}

			const bearerPrefix = "Bearer "
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				http.Error(w, "invalid auth header format", http.StatusUnauthorized)
				return
			}
			tokenString := strings.TrimPrefix(authHeader, bearerPrefix)
			tokenString = strings.TrimSpace(tokenString)
			if tokenString == "" {
				http.Error(w, "empty token", http.StatusUnauthorized)
				return
			}

			claims := &TokenClaims{}
			token, err := jwt.ParseWithClaims(
				tokenString,
				claims,
				func(t *jwt.Token) (interface{}, error) {
					if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
						return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
					}
					return []byte(secret), nil
				},
				jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			)
			if err != nil {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}
			if !token.Valid {
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, GroupIDKey, claims.GroupID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
