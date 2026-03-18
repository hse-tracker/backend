package parser

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type Notifier interface {
	SendMessage(userID int64, text string) error
}

type SyncSubject struct {
	ID              int64  `db:"id"`
	GroupID         int64  `db:"group_id"`
	Name            string `db:"name"`
	URL             string `db:"url"`
	NameColumnIndex int    `db:"name_column_index"`
	DataStartRow    int    `db:"data_start_row"`
}

type SyncUser struct {
	ID       int64  `db:"id"`
	FullName string `db:"full_name"`
}

type SyncNode struct {
	ID          int64  `db:"id"`
	Name        string `db:"name"`
	ColumnIndex int    `db:"column_index"`
}

// infinite update cycle
func StartSyncWorker(db *sqlx.DB, credsPath string, bot Notifier, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("[SyncWorker] PANIC recovered: %v", r)
					}
				}()

				log.Println("[SyncWorker] starting sync cycle...")
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				if err := syncAllGrades(ctx, db, credsPath, bot); err != nil {
					log.Printf("[SyncWorker] error while sync: %v\n", err)
				}
				cancel()
				log.Println("[SyncWorker] sync cycle ended.")
			}()
		}
	}()
}

func syncAllGrades(ctx context.Context, db *sqlx.DB, credsPath string, bot Notifier) error {
	// get subjects with available grade structure
	var subjects []SyncSubject
	query := `
		SELECT id, group_id, name, url, name_column_index, data_start_row
		FROM subjects
		WHERE status = 'ready' AND name_column_index IS NOT NULL AND data_start_row IS NOT NULL`

	if err := db.SelectContext(ctx, &subjects, query); err != nil {
		return fmt.Errorf("error while getting parsed subjects: %w", err)
	}

	for _, sub := range subjects {
		if err := syncSingleSubject(ctx, db, sub, credsPath, bot); err != nil {
			log.Printf("[SyncWorker] error while updating single subject (ID: %d): %v\n", sub.ID, err)
		} else {
			log.Printf("[SyncWorker] subjectID: %d update success\n", sub.ID)
		}

		time.Sleep(5 * time.Second)
	}

	return nil
}

func syncSingleSubject(ctx context.Context, db *sqlx.DB, sub SyncSubject, credsPath string, bot Notifier) error {
	log.Printf("[Sync] Syncing subject '%s' (ID: %d)", sub.Name, sub.ID)

	sheetID, gid, err := ExtractSheetInfo(sub.URL)
	if err != nil {
		return err
	}

	// get all raw sheet sheet data without formulas
	rawValues, err := FetchSheetDataForSync(ctx, sheetID, gid, credsPath)
	if err != nil {
		return err
	}

	// get all students from group
	var users []SyncUser
	if err := db.SelectContext(ctx, &users, `SELECT id, full_name FROM users WHERE group_id = $1`, sub.GroupID); err != nil {
		return err
	}

	// map "normalize name" -> User ID
	userMap := make(map[string]int64)
	for _, u := range users {
		userMap[normalizeName(u.FullName)] = u.ID
	}

	// get grade structure nodes for subject (only with columnID not null)
	var nodes []SyncNode
	if err := db.SelectContext(ctx, &nodes, `SELECT id, name, column_index FROM grade_structures WHERE subject_id = $1 AND column_index IS NOT NULL`, sub.ID); err != nil {
		return err
	}

	// iterating sheet rows starting with data_start_row index
	for rowIndex := sub.DataStartRow; rowIndex < len(rawValues); rowIndex++ {
		row := rawValues[rowIndex]
		if len(row) <= sub.NameColumnIndex {
			continue // empty row
		}

		// get cell with student name
		studentName := cellString(row[sub.NameColumnIndex])
		if studentName == "" {
			continue
		}

		// trying to find user in db
		userID, exists := userMap[normalizeName(studentName)]
		if !exists {
			continue // user is not in db
		}

		log.Printf("[Sync] Processing student: %s (ID: %d)", studentName, userID)

		// check users grades
		for _, node := range nodes {
			if len(row) <= node.ColumnIndex {
				continue // empty cell
			}

			newVal := cellString(row[node.ColumnIndex])
			if newVal == "" {
				continue
			}

			// get previous grade
			var oldVal string
			err := db.GetContext(ctx, &oldVal, `SELECT value FROM student_grades WHERE user_id = $1 AND grade_structure_id = $2`, userID, node.ID)

			isNew := err == sql.ErrNoRows
			isChanged := err == nil && oldVal != newVal

			if isNew || isChanged {
				// write or update grade
				upsertQuery := `
					INSERT INTO student_grades (user_id, grade_structure_id, value, updated_at)
					VALUES ($1, $2, $3, NOW())
					ON CONFLICT (user_id, grade_structure_id) DO UPDATE
					SET value = EXCLUDED.value, updated_at = NOW()`

				_, dbErr := db.ExecContext(ctx, upsertQuery, userID, node.ID, newVal)
				if dbErr != nil {
					log.Printf("[SyncWorker] Ошибка сохранения оценки юзера %d: %v", userID, dbErr)
					continue
				}

				if isChanged {
					msg := fmt.Sprintf("🔔 Обновлена оценка!\n\nПредмет: **%s**\nЭлемент: **%s**\nБыло: %s ➡️ Стало: **%s**", sub.Name, node.Name, oldVal, newVal)
					bot.SendMessage(userID, msg)
					time.Sleep(1 * time.Second)
				} else if isNew {
					msg := fmt.Sprintf("🔔 Выставлена новая оценка!\n\nПредмет: **%s**\nЭлемент: **%s**\nОценка: **%s**", sub.Name, node.Name, newVal)
					bot.SendMessage(userID, msg)
					time.Sleep(1 * time.Second)
				}
			}
		}
	}

	// update last_parsed value
	db.ExecContext(ctx, `UPDATE subjects SET last_parsed_at = NOW() WHERE id = $1`, sub.ID)
	return nil
}

// normalizes full_name to lowercase, trims all redundant spaces
// example: " Иванов   Иван Иванович " -> "иванов иван иванович"
func normalizeName(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(name), " "))
}

// converts Google API value to string
func cellString(val interface{}) string {
	if val == nil {
		return ""
	}

	// UNFORMATTED_VALUE returns float64 or string
	str := fmt.Sprintf("%v", val)
	return strings.TrimSpace(str)
}
