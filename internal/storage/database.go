package storage

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/huandu/go-sqlbuilder"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func RegisterUser(db *sqlx.DB, fullName string, groupName string) (*User, error) {
	tx, err := db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("error while starting transaction: %w", err)
	}
	defer func(tx *sqlx.Tx) {
		err := tx.Rollback()
		if err != nil {
		}
	}(tx)

	// trying to find existing group
	var groupID int64
	err = tx.Get(&groupID, "SELECT id FROM groups WHERE name = $1", groupName)

	// creating new group
	if err == sql.ErrNoRows {
		queryCreateGroup := `INSERT INTO groups (name) VALUES ($1) RETURNING id`
		err = tx.QueryRowx(queryCreateGroup, groupName).Scan(&groupID)
		if err != nil {
			return nil, fmt.Errorf("error while creating group: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("error while registering user: %w", err)
	}

	// creating user
	var newUserID int64
	queryCreateUser := `
		INSERT INTO users (group_id, full_name)
		VALUES ($1, $2)
		RETURNING id`

	err = tx.QueryRowx(queryCreateUser, groupID, fullName).Scan(&newUserID)
	if err != nil {
		return nil, fmt.Errorf("error while creating user: %w", err)
	}

	err = tx.Commit()
	if err != nil {
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

func GetSubjects(db *sqlx.DB, groupID, userID int64) ([]SubjectResponse, error) {
	var subjects []SubjectResponse
	query := `
			SELECT id, name, status
			FROM subjects
			WHERE group_id = $1 AND (group_connected = true OR creator_id = $2)
			ORDER BY created_at DESC
		`

	err := db.Select(&subjects, query, groupID, userID)
	if err != nil {
		return nil, err
	}

	// no subjects
	if subjects == nil {
		subjects = []SubjectResponse{}
	}

	return subjects, nil
}

type NavigationLog struct {
	UserID    int64     `db:"user_id"`
	TabName   string    `db:"tab_name"`
	CreatedAt time.Time `db:"created_at"`
}

// TODO: invoke when user changes screen
func LogNavigation(db *sqlx.DB, userID int64, tabName string) error {
	ib := sqlbuilder.NewInsertBuilder()
	ib.InsertInto("navigation_logs").
		Cols("user_id", "tab_name").
		Values(userID, tabName)
	query, args := ib.BuildWithFlavor(sqlbuilder.PostgreSQL)
	_, err := db.Exec(query, args...)
	return err
}

func GetAllNavigationLogs(db *sqlx.DB) ([]NavigationLog, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("user_id", "tab_name", "created_at").From("navigation_logs").OrderByDesc("created_at")
	query, args := sb.BuildWithFlavor(sqlbuilder.PostgreSQL)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("error closing rows: %v", err)
		}
	}(rows)

	var logs []NavigationLog
	for rows.Next() {
		var l NavigationLog
		if err := rows.Scan(&l.UserID, &l.TabName, &l.CreatedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}
