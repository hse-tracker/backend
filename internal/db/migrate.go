package db

import (
	_ "embed"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

//go:embed migrations/initial_up.sql
var initialMigration string

// checks and creates tables in db
func RunMigrations(db *sqlx.DB) error {
	log.Println("[DB] checking and creating tables...")
	
	_, err := db.Exec(initialMigration)
	if err != nil {
		return fmt.Errorf("error while executing file initial_up.sql: %w", err)
	}

	log.Println("[DB] tables created")
	return nil
}