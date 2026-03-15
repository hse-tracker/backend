package storage

import (
	"database/sql"
	"log"
	"time"

	"github.com/huandu/go-sqlbuilder"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

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
