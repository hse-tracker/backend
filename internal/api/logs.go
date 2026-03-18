package api

import (
	"encoding/json"
	"net/http"
	"log"

	"github.com/hse-tracker/backend/internal/storage"
	"github.com/jmoiron/sqlx"
)

type LogNavigationRequest struct {
	TabName string `json:"tab_name"`
}

func LogNavigationHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(UserIDKey).(int64)

		var req LogNavigationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json request", http.StatusBadRequest)
			return
		}

		err := storage.LogNavigation(db, userID, req.TabName)
		if err != nil {
			log.Printf(" [API] error while writing navigation log: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}