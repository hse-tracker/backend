package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hse-tracker/backend/internal/storage"
	"github.com/jmoiron/sqlx"
)

type TokenClaims struct {
	UserID  int64 `json:"user_id"`
	GroupID int64 `json:"group_id"`
	jwt.RegisteredClaims
}

// user register request data type
type RegisterRequest struct {
	FullName string `json:"full_name"`
	Group    string `json:"group"`
}

// user register response data type
type RegisterResponse struct {
	Token string `json:"token"`
}

// register handler
func RegisterHandler(db *sqlx.DB, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// decoding JSON
		var req RegisterRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		// TODO: save user to DB
		user, err := storage.RegisterUser(db, req.FullName, req.Group)
		if err != nil {
			fmt.Printf("db error while registering user: %s\n", err)
			http.Error(w, "internal error", 500)
		}
		fmt.Printf("user created: \n%s, \n%d\n", user.FullName, user.GroupID)

		claims := TokenClaims{
			UserID:  user.ID,
			GroupID: user.GroupID,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(72 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signedToken, err := token.SignedString([]byte(secret))
		if err != nil {
			fmt.Printf("jwt error: %s\n", err)
			http.Error(w, "internal error", 500)
			return
		}

		// sending response
		resp := RegisterResponse{
			Token: signedToken,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(resp)
		if err != nil {
			return
		}
	}
}
