package adminserver

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/hse-tracker/backend/internal/storage"
)

type RegisterUserRequest struct {
	UserID       int64  `json:"user_id" example:"123456789"`
	FirstName    string `json:"first_name" example:"Dima"`
	MiddleName   string `json:"middle_name" example:"Dmitrievich"`
	LastName     string `json:"last_name" example:"Malay"`
	FacultyID    int    `json:"faculty_id" example:"1"`
	ProgramID    int    `json:"program_id" example:"1"`
	CourseNumber int    `json:"course_number" example:"3"`
}

type CreateSubjectRequest struct {
	Name     string `json:"name" example:"Databases"`
	Type     int    `json:"type" example:"1"`
	TableURL string `json:"table_url" example:"https://docs.google.com/spreadsheets/d/"`
}

type AssignTableRequest struct {
	TableID  int    `json:"table_id" example:"1"`
	TableURL string `json:"table_url,omitempty" example:"https://docs.google.com/spreadsheets/d/"`
}

func getUserIDFromHeader(r *http.Request) int64 {
	uidStr := r.Header.Get("X-Telegram-User-ID")
	if uidStr == "" {
		return 0
	}
	uid, _ := strconv.ParseInt(uidStr, 10, 64)
	return uid
}

// @Summary Get list of faculties
// @Description Returns all available faculties for registration filter.
// @Tags meta
// @Produce  json
// @Success 200 {array} storage.Faculty
// @Failure 500 {string} string "Internal Server Error"
// @Router /faculties [get]
func (s *Server) getFacultiesHandler(w http.ResponseWriter, _ *http.Request) {
	faculties, err := s.db.GetFaculties()
	if err != nil {
		log.Printf("ERROR: Failed to get faculties: %v", err)
		http.Error(w, "Could not retrieve faculties", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(faculties)
	if err != nil {
		return
	}
}

// @Summary Get list of programs
// @Description Returns programs filtered by faculty_id.
// @Tags meta
// @Produce  json
// @Param   faculty_id query int true "Faculty ID"
// @Success 200 {array} storage.Program
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /programs [get]
func (s *Server) getProgramsHandler(w http.ResponseWriter, r *http.Request) {
	fidStr := r.URL.Query().Get("faculty_id")
	fid, err := strconv.Atoi(fidStr)
	if err != nil {
		http.Error(w, "Invalid faculty_id", http.StatusBadRequest)
		return
	}

	programs, err := s.db.GetPrograms(fid)
	if err != nil {
		log.Printf("ERROR: Failed to get programs: %v", err)
		http.Error(w, "Could not retrieve programs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(programs)
	if err != nil {
		return
	}
}

// @Summary Register or Update User
// @Description Registers a new user or updates existing profile info.
// @Tags user
// @Accept  json
// @Produce  json
// @Param   request body RegisterUserRequest true "User Info"
// @Success 200 {object} map[string]string
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /user/register [post]
func (s *Server) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	if req.UserID == 0 || req.FirstName == "" || req.CourseNumber < 1 || req.CourseNumber > 6 {
		http.Error(w, "Invalid parameters", http.StatusBadRequest)
		return
	}

	u := storage.User{
		ID:                   req.UserID,
		FirstName:            req.FirstName,
		MiddleName:           req.MiddleName,
		LastName:             req.LastName,
		ProgramID:            &req.ProgramID,
		CourseNumber:         req.CourseNumber,
		Language:             "ru",
		NotificationsEnabled: true,
	}

	if err := s.db.UpsertUser(u); err != nil {
		log.Printf("ERROR: Failed to register user %d: %v", req.UserID, err)
		http.Error(w, "Registration failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	if err != nil {
		return
	}
}

// @Summary Get User Profile
// @Description Returns current user info and settings.
// @Tags user
// @Produce  json
// @Param X-Telegram-User-ID header int true "Telegram User ID"
// @Success 200 {object} storage.User
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "User not found"
// @Router /user/me [get]
func (s *Server) getUserProfileHandler(w http.ResponseWriter, r *http.Request) {
	uid := getUserIDFromHeader(r)
	if uid == 0 {
		http.Error(w, "Missing X-Telegram-User-ID header", http.StatusUnauthorized)
		return
	}

	u, err := s.db.GetUser(uid)
	if err != nil {
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}
	if u == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(u)
	if err != nil {
		return
	}
}

// @Summary Get Available Subjects
// @Description Returns list of subjects (Program-specific or Global).
// @Tags subjects
// @Produce  json
// @Param X-Telegram-User-ID header int true "Telegram User ID"
// @Success 200 {array} storage.Subject
// @Failure 500 {string} string "Internal Server Error"
// @Router /subjects [get]
func (s *Server) getSubjectsHandler(w http.ResponseWriter, r *http.Request) {
	uid := getUserIDFromHeader(r)

	user, err := s.db.GetUser(uid)
	if err != nil || user == nil || user.ProgramID == nil {
		http.Error(w, "User not found or program not set", http.StatusForbidden)
		return
	}

	subjects, err := s.db.GetAvailableSubjects(*user.ProgramID)
	if err != nil {
		log.Printf("ERROR: Failed to get subjects: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(subjects)
	if err != nil {
		return
	}
}

// @Summary Create New Subject
// @Description User creates a new subject and links a table to it.
// @Tags subjects
// @Accept  json
// @Produce  json
// @Param X-Telegram-User-ID header int true "Telegram User ID"
// @Param request body CreateSubjectRequest true "Subject Data"
// @Success 201 {object} map[string]interface{}
// @Router /subjects/create [post]
func (s *Server) createSubjectHandler(w http.ResponseWriter, r *http.Request) {
	uid := getUserIDFromHeader(r)
	var req CreateSubjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	user, _ := s.db.GetUser(uid)
	if user == nil {
		http.Error(w, "User not found", http.StatusForbidden)
		return
	}

	var programID *int
	if req.Type == 1 {
		programID = user.ProgramID
	}

	subID, err := s.db.CreateSubject(req.Name, req.Type, programID)
	if err != nil {
		log.Printf("ERROR: CreateSubject: %v", err)
		http.Error(w, "Failed to create subject", http.StatusInternalServerError)
		return
	}

	tableID, err := s.db.FindOrCreateTable(subID, req.TableURL, uid)
	if err != nil {
		log.Printf("ERROR: FindOrCreateTable: %v", err)
		http.Error(w, "Failed to register table", http.StatusInternalServerError)
		return
	}

	_ = s.db.SubscribeUser(uid, tableID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(map[string]interface{}{
		"subject_id": subID,
		"table_id":   tableID,
	})
	if err != nil {
		return
	}
}

// @Summary Get Tables for Subject
// @Description Returns list of tables added by users for this subject.
// @Tags subjects
// @Produce  json
// @Param subject_id path int true "Subject ID"
// @Router /subjects/{subject_id}/tables [get]
func (s *Server) getSubjectTablesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write([]byte("[]"))
	if err != nil {
		return
	} // TODO: Implement GetTablesBySubjectID in storage
}

// @Summary Assign Table to User
// @Description Subscribe user to an existing table (or add new link to existing subject).
// @Tags subjects
// @Accept json
// @Produce json
// @Param X-Telegram-User-ID header int true "Telegram User ID"
// @Param subject_id path int true "Subject ID"
// @Param request body AssignTableRequest true "Table Info"
// @Router /subjects/{subject_id}/assign [post]
func (s *Server) assignTableHandler(w http.ResponseWriter, r *http.Request) {
	uid := getUserIDFromHeader(r)
	subIDStr := chi.URLParam(r, "subject_id")
	subID, _ := strconv.Atoi(subIDStr)

	var req AssignTableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	var targetTableID int

	if req.TableID > 0 {
		targetTableID = req.TableID
	} else if req.TableURL != "" {
		id, err := s.db.FindOrCreateTable(subID, req.TableURL, uid)
		if err != nil {
			http.Error(w, "Failed to create table", http.StatusInternalServerError)
			return
		}
		targetTableID = id
	} else {
		http.Error(w, "Provide table_id or table_url", http.StatusBadRequest)
		return
	}

	if err := s.db.SubscribeUser(uid, targetTableID); err != nil {
		http.Error(w, "Failed to subscribe", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(map[string]string{"status": "subscribed"})
	if err != nil {
		return
	}
}

type LogNavigationRequest struct {
	TabName string `json:"tab_name"`
}

func (s *Server) logNavigationHandler(w http.ResponseWriter, r *http.Request) {
	uid := getUserIDFromHeader(r)
	var req LogNavigationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if err := s.db.LogNavigation(uid, req.TabName); err != nil {
		http.Error(w, "failed to log", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
