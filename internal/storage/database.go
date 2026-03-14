package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func RegisterUser(db *sqlx.DB, fullName string, groupName string) (*User, error) {
	tx, err := db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("error while starting transaction: %w", err)
	}
	defer tx.Rollback()

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
		ID:			newUserID,
		GroupID:	groupID,
		FullName:	fullName,
		Language:	"ru",
		CreatedAt:	time.Now(),
		UpdatedAt:	time.Now(),
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
