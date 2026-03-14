package api

import (
	"fmt"
	"net/http"
	"encoding/json"
	"encoding/base64"

	"github.com/jmoiron/sqlx"
	"mockBackend/internal/storage"
)

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
func RegisterHandler(db *sqlx.DB) http.HandlerFunc {
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

		// TODO: integrate jwt tokens
		rawToken := fmt.Sprintf("%d:%d", user.ID, user.GroupID)
		mockToken := base64.StdEncoding.EncodeToString([]byte(rawToken))

		// sending response
		resp := RegisterResponse{
			Token: mockToken,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}
}
