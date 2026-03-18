package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func RegisterUser(db *sqlx.DB, tgID int64, fullName string, groupName string) (*User, error) {
	tx, err := db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("error while starting transaction: %w", err)
	}
	defer tx.Rollback()

	var groupID int64
	err = tx.Get(&groupID, "SELECT id FROM groups WHERE name = $1", groupName)
	if err == sql.ErrNoRows {
		queryCreateGroup := `INSERT INTO groups (name) VALUES ($1) RETURNING id`
		err = tx.QueryRowx(queryCreateGroup, groupName).Scan(&groupID)
		if err != nil {
			return nil, fmt.Errorf("error while creating group: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error while checking group: %w", err)
	}

	var newUserID int64
	queryCreateUser := `
		INSERT INTO users (id, group_id, full_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (id) DO UPDATE 
		SET group_id = EXCLUDED.group_id, full_name = EXCLUDED.full_name, updated_at = CURRENT_TIMESTAMP
		RETURNING id`

	err = tx.QueryRowx(queryCreateUser, tgID, groupID, fullName).Scan(&newUserID)
	if err != nil {
		return nil, fmt.Errorf("error while creating/updating user: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("error while commiting transaction: %w", err)
	}

	return &User{
		ID:        newUserID,
		GroupID:   groupID,
		FullName:  fullName,
		Language:  "ru",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// TODO: add groupConnected handler
func AddSubject(db *sqlx.DB, creatorID int64, groupID int64,
	name string, url string) (int64, error) {

	var newSubjectID int64
	query := `
		INSERT INTO subjects (creator_id, group_id, name, url, group_connected, status)
		VALUES ($1, $2, $3, $4, $5, 'processing')
		RETURNING id`

	err := db.QueryRowx(query, creatorID, groupID, name, url, true).Scan(&newSubjectID)
	if err != nil {
		return 0, fmt.Errorf("error inserting subject: %w", err)
	}
	return newSubjectID, nil
}

// gets ALL subjects
func GetSubjects(db *sqlx.DB, groupID, userID int64) ([]SubjectResponse, error) {
	var subjects[]SubjectResponse
	
	query := `
		SELECT 
			s.id, 
			s.name, 
			s.status,
			sg.value AS grade
		FROM subjects s
		LEFT JOIN grade_structures gs ON gs.subject_id = s.id AND gs.parent_id IS NULL
		LEFT JOIN student_grades sg ON sg.grade_structure_id = gs.id AND sg.user_id = $2
		WHERE s.group_id = $1 AND (s.group_connected = true OR s.creator_id = $2)
		ORDER BY s.created_at DESC
	`

	err := db.Select(&subjects, query, groupID, userID)
	if err != nil {
		return nil, err
	}

	if subjects == nil {
		subjects =[]SubjectResponse{}
	}

	return subjects, nil
}

// get single subject
func GetSubjectStructure(db *sqlx.DB, subjectID, userID, groupID int64) (*SubjectStructureResponse, error) {
	// check if subject exists and connected to students group
	var subj SubjectResponse
	querySubj := `SELECT id, name, status FROM subjects WHERE id = $1 AND group_id = $2`
	if err := db.Get(&subj, querySubj, subjectID, groupID); err != nil {
		return nil, err // sql.ErrNoRows
	}

	// get whole grade structure and add values of current student
	queryGrades := `
		SELECT 
			gs.id, gs.parent_id, gs.name, gs.type, gs.weight, gs.display_formula,
			sg.value
		FROM grade_structures gs
		LEFT JOIN student_grades sg ON gs.id = sg.grade_structure_id AND sg.user_id = $2
		WHERE gs.subject_id = $1
		ORDER BY gs.id ASC
	`
	
	var rows[]GradeRow
	if err := db.Select(&rows, queryGrades, subjectID, userID); err != nil {
		return nil, err
	}

	// - convert array to tree -
	
	// initialise map for quick search nodes by ID
	nodesMap := make(map[int64]*GradeNodeResponse)
	for _, r := range rows {
		nodesMap[r.ID] = &GradeNodeResponse{
			ID:             r.ID,
			Name:           r.Name,
			Type:           r.Type,
			Weight:         r.Weight,
			DisplayFormula: r.DisplayFormula,
			Value:          r.Value,
			Children:       make([]*GradeNodeResponse, 0), // must be [], not null
		}
	}

	tree := make([]*GradeNodeResponse, 0)
	for _, r := range rows {
		node := nodesMap[r.ID]
		if r.ParentID == nil {
			// root
			tree = append(tree, node)
		} else {
			// child
			if parent, exists := nodesMap[*r.ParentID]; exists {
				parent.Children = append(parent.Children, node)
			}
		}
	}

	return &SubjectStructureResponse{
		ID:        subj.ID,
		Name:      subj.Name,
		Status:    subj.Status,
		Structure: tree,
	}, nil
}

// deletes subject by id
func DeleteSubject(db *sqlx.DB, subjectID, groupID int64) error {
	res, err := db.Exec(`DELETE FROM subjects WHERE id = $1 AND group_id = $2`, subjectID, groupID)
	if err != nil {
		return err
	}
	
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	
	return nil
}
