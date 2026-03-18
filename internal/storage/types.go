package storage

import (
	"time"
)

// Users group. Example "name":"БПИ228"
type Group struct {
	ID        int64     `db:"id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
}

// User. Example "full_name":"Иванов Иван Иванович"
type User struct {
	ID        int64     `db:"id"`
	GroupID   int64     `db:"group_id"`
	FullName  string    `db:"full_name"`
	Language  string    `db:"language"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// Users/Groups subject. Example "name":"UX design"
type Subject struct {
	ID              int64     `db:"id"`
	CreatorID       int64     `db:"creator_id"`
	GroupID         int64     `db:"group_id"`
	Name            string    `db:"name"`
	URL             string    `db:"url"`
	GroupConnected  bool      `db:"group_connected"`
	Notifications   bool      `db:"notifications"`
	Status          string    `db:"status"`
	ErrorMessage    string    `db:"error_message"`
	LastHash        string    `db:"last_hash"`
	LastParsedAt    time.Time `db:"last_parsed_at"`
	CreatedAt       time.Time `db:"created_at"`
	NameColumnIndex *int      `db:"name_column_index"`
    DataStartRow    *int      `db:"data_start_row"`
}

type SubjectResponse struct {
	ID     int64   `json:"id" db:"id"`
	Name   string  `json:"name" db:"name"`
	Status string  `json:"status" db:"status"`
	Grade  *string `json:"grade" db:"grade"`
}

// Represents one(!) assessment point
type GradeStructure struct {
	ID int64 `db:"id"`

	// link to subject
	SubjectID int64 `db:"subject_id"`

	// link to parent GradeStructure node
	// can be null if root
	ParentID *int64 `db:"parent_id"`

	// name of current GradeStructure node
	// example: "Итог", "ДЗ"
	Name string `db:"name"`

	// (???) could be removed in future
	Type string `db:"type"`

	// represents column index in google sheets
	// example: (0..25) = ("A".."Z"), (26..51) = ("AA".."AZ"), etc
	ColumnIndex int `db:"column_index"`

	Weight *float64 `db:"weight"`
	DisplayFormula string `db:"display_formula"`
}

// Represent value of each GradeStructure node
type StudentGrade struct {
	ID               int64     `db:"id"`
	UserID           int64     `db:"user_id"`
	GradeStructureID int64     `db:"grade_structure_id"`
	Value            string    `db:"value"`
	UpdatedAt        time.Time `db:"updated_at"`
}

// struct for mapping string from JOIN
type GradeRow struct {
	ID             int64    `db:"id"`
	ParentID       *int64   `db:"parent_id"`
	Name           string   `db:"name"`
	Type           string   `db:"type"`
	Weight         *float64 `db:"weight"`
	DisplayFormula *string  `db:"display_formula"`
	Value          *string  `db:"value"` // nullable
}

// node for frontend
type GradeNodeResponse struct {
	ID             int64                `json:"id"`
	Name           string               `json:"name"`
	Type           string               `json:"type"`
	Weight         *float64             `json:"weight"`
	DisplayFormula *string              `json:"display_formula"`
	Value          *string              `json:"value"`
	Children       []*GradeNodeResponse `json:"children"`
}

// final subject response
type SubjectStructureResponse struct {
	ID        int64                `json:"id"`
	Name      string               `json:"name"`
	Status    string               `json:"status"`
	Structure[]*GradeNodeResponse `json:"structure"`
}