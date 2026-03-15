package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hse-tracker/backend/internal/storage"

	"github.com/jmoiron/sqlx"
)

type CreateSubjectRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func CreateSubjectHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(UserIDKey).(int64)
		groupID := r.Context().Value(GroupIDKey).(int64)

		var req CreateSubjectRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "invalid json request", 400)
			return
		}

		subjectID, err := storage.AddSubject(db, userID, groupID, req.Name, req.URL)
		if err != nil {
			http.Error(w, "internal error", 500)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(map[string]interface{}{
			"message":    "subject created",
			"subject_id": subjectID,
			"status":     "processing",
		})
		if err != nil {
			return
		}
	}
}

// returns list of users subject
func GetSubjectsHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(UserIDKey).(int64)
		groupID := r.Context().Value(GroupIDKey).(int64)

		subjects, err := storage.GetSubjects(db, userID, groupID)
		if err != nil {
			fmt.Println("error fetching subjects:", err)
			http.Error(w, "internal error", 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(subjects)
		if err != nil {
			return
		}
	}
}
