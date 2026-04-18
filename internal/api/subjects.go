package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"errors"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"github.com/hse-tracker/backend/internal/config"
	"github.com/hse-tracker/backend/internal/parser"
	"github.com/hse-tracker/backend/internal/storage"
)

type CreateSubjectRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func CreateSubjectHandler(db *sqlx.DB, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(UserIDKey).(int64)
		groupID := r.Context().Value(GroupIDKey).(int64)

		log.Printf("[API] CreateSubject: UserID=%d, GroupID=%d", userID, groupID)

		var req CreateSubjectRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "invalid json request", http.StatusBadRequest)
			return
		}

		subjectID, err := storage.AddSubject(db, userID, groupID, req.Name, req.URL)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			if errors.Is(err, storage.ErrSubjectAlreadyExists) {
				w.WriteHeader(http.StatusConflict) // 409 Conflict
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": "Предмет с такой ссылкой уже был добавлен в вашу группу ранее.",
				})
				return
			}
			
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		go parser.ProcessNewSubject(db, subjectID, req.URL, cfg.GoogleCredsPath, cfg.DeepSeekKey)

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

		log.Printf("[API] GetSubjects: UserID=%d, GroupID=%d", userID, groupID)

		subjects, err := storage.GetSubjects(db, groupID, userID)
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

// returns whole grade structure of subject
func GetSubjectHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(UserIDKey).(int64)
		groupID := r.Context().Value(GroupIDKey).(int64)

		// get subject ID from URL
		subjectIDStr := chi.URLParam(r, "id")
		subjectID, err := strconv.ParseInt(subjectIDStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid subject id", http.StatusBadRequest)
			return
		}

		log.Printf("[API] GetSubjectStructure: ID=%d, UserID=%d", subjectID, userID)

		// get grade struct from db
		resp, err := storage.GetSubjectStructure(db, subjectID, userID, groupID)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "subject not found", http.StatusNotFound)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// return json to frontend
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			return
		}
	}
}

// deletes subject by id
func DeleteSubjectHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupID := r.Context().Value(GroupIDKey).(int64)

		subjectIDStr := chi.URLParam(r, "id")
		subjectID, err := strconv.ParseInt(subjectIDStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid subject id", http.StatusBadRequest)
			return
		}

		log.Printf("[API] DeleteSubject: ID=%d, GroupID=%d", subjectID, groupID)

		err = storage.DeleteSubject(db, subjectID, groupID)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "subject not found or access denied", http.StatusNotFound)
				return
			}
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "subject deleted",
		})
	}
}
