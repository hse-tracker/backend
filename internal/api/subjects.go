package api

import (
	"fmt"
	"encoding/json"
	"mockBackend/internal/storage"
	"net/http"

	"github.com/jmoiron/sqlx"
)

type CreateSubjectRequest struct {
	Name	string	`json:"name"`
	URL		string	`json:"url"`
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
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":		"subject created",
			"subject_id":	subjectID,
			"status":		"processing",
		})
	}
}

type SubjectResponse struct {
	ID		int64	`json:"id"`
	Name	string	`json:"name"`
	Status	string	`json:"status"`
}

func GetSubjectsHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(UserIDKey).(int64)
		groupID := r.Context().Value(GroupIDKey).(int64)

		var subjects []SubjectResponse
		query := `
			SELECT id, name, status
			FROM subjects 
			WHERE group_id = $1 AND (group_connected = true OR creator_id = $2)
			ORDER BY created_at DESC
		`

		err := db.Select(&subjects, query, groupID, userID)
		if err != nil {
			fmt.Println("error fetching subjects:", err)
			http.Error(w, "internal error", 500)
			return
		}

		// no subjects
		if subjects == nil {
			subjects =[]SubjectResponse{}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(subjects)
	}
}