package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type ctxKey string

const (
	UserIDKey	ctxKey = "userID"
	GroupIDKey	ctxKey = "groupID"
)

// TODO: implement JWT tokens instead of base64 encoding
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing auth header", 401)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		decoded, err := base64.RawStdEncoding.DecodeString(tokenString)
		if err != nil {
			http.Error(w, "invalid token", 401)
			return
		}

		parts := strings.Split(string(decoded), ":")
		if len(parts) != 2 {
			http.Error(w, "invalid token format", 401)
			return
		}

		// getting user and group id from Split("userID:groupID", ":")
		userID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			http.Error(w, "internal error", 500)
			return
		}
		groupID, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			http.Error(w, "internal error", 500)
			return
		}

		// debug log
		fmt.Println("\n", userID, groupID)

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		ctx = context.WithValue(ctx, GroupIDKey, groupID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}