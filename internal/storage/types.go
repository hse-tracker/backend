package storage

import (
	"time"
)

type Group struct {
	ID			int64		`db:"id"`
	Name		string		`db:"name"`
	CreatedAt	time.Time	`db:"created_at"`
}


type User struct {
	ID			int64		`db:"id"`
	GroupID		int64		`db:"group_id"`
	FullName	string		`db:"full_name"`
	Language	string		`db:"language"`
	CreatedAt	time.Time	`db:"created_at"`
	UpdatedAt	time.Time	`db:"updated_at"`
}

type Subject struct {
	ID				int64		`db:"id"`
	CreatorID		int64		`db:"creator_id"`
	GroupID			int64		`db:"group_id"`
	Name			string		`db:"name"`
	URL				string		`db:"url"`
	GroupConnected	bool		`db:"group_connected"`
	Notifications	bool		`db:"notifications"`
	Status			string		`db:"status"`
	ErrorMessage	string		`db:"error_message"`
	LastHash		string		`db:"last_hash"`
	LastParsedAt	time.Time	`db:"last_parsed_at"`
	CreatedAt		time.Time	`db:"created_at"`
}

type GradeStructure struct {
	ID			int64	`db:"id"`
	SubjectID	int64	`db:"subject_id"`
	ParentID	int64	`db:"parent_id"`
	Name		string	`db:"name"`
	Type		string	`db:"type"`
	ColumnIndex	int		`db:"column_index"`
}

type StudentGrade struct {
	ID					int64		`db:"id"`
	UserID				int64		`db:"user_id"`
	GradeStructureID	int64		`db:"grade_structure_id"`
	Value				string		`db:"value"`
	UpdatedAt			time.Time	`db:"updated_at"`
}
