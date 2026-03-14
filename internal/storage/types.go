package storage

import (
	"time"
)

// Users group. Example "name":"БПИ228"
type Group struct {
	ID			int64		`db:"id"`
	Name		string		`db:"name"`
	CreatedAt	time.Time	`db:"created_at"`
}

// User. Example "full_name":"Иванов Иван Иванович"
type User struct {
	ID			int64		`db:"id"`
	GroupID		int64		`db:"group_id"`
	FullName	string		`db:"full_name"`
	Language	string		`db:"language"`
	CreatedAt	time.Time	`db:"created_at"`
	UpdatedAt	time.Time	`db:"updated_at"`
}

// Users/Groups subject. Example "name":"UX design"
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

// Represents one(!) assessment point
type GradeStructure struct {
	ID			int64	`db:"id"`

	// link to subject
	SubjectID	int64	`db:"subject_id"`

	// link to parent GradeStructure node
	// can be null if root
	ParentID	int64	`db:"parent_id"`

	// name of current GradeStructure node
	// example: "Итог", "ДЗ"
	Name		string	`db:"name"`

	// (???) could be removed in future
	Type		string	`db:"type"`

	// represents column index in google sheets
	// example: (0..25) = ("A".."Z"), (26..51) = ("AA".."AZ"), etc
	ColumnIndex	int		`db:"column_index"`
}

// Represent value of each GradeStructure node
type StudentGrade struct {
	ID					int64		`db:"id"`
	UserID				int64		`db:"user_id"`
	GradeStructureID	int64		`db:"grade_structure_id"`
	Value				string		`db:"value"`
	UpdatedAt			time.Time	`db:"updated_at"`
}
