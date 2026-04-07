package parser

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

// wrapper for processing subject, updates subject status to db
// depth 1
func ProcessNewSubject(db *sqlx.DB, subjectID int64, url, credsPath, deepseekKey string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Parser] PANIC recovered: %v", r)
			db.Exec(`UPDATE subjects SET status = 'error', error_message = 'Internal error during parsing' WHERE id = $1`, subjectID)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	err := runParsingPipeline(ctx, db, subjectID, url, credsPath, deepseekKey)

	status := "ready"
	errMsg := ""
	if err != nil {
		status = "error"
		errMsg = err.Error()
		log.Printf("[Parser] Error parsing subject %d: %v", subjectID, err)
	}

	_, dbErr := db.Exec(`UPDATE subjects SET status = $1, error_message = $2 WHERE id = $3`, status, errMsg, subjectID)
	if dbErr != nil {
		log.Printf("[Parser] Failed to update subject status: %v", dbErr)
	}

}

// main pipeline for project processing
// depth 2
func runParsingPipeline(ctx context.Context, db *sqlx.DB, subjectID int64, url, credsPath, deepseekKey string) error {
	log.Printf("[Parser] Starting pipeline for SubjectID=%d, URL=%s", subjectID, url)

	// get sheet ID from raw link
	sheetID, gid, err := ExtractSheetInfo(url)
	if err != nil {
		return err
	}

	// get raw data from Google Sheets API
	log.Printf("[Parser] Fetching data from Google Sheets (SheetID: %s)", sheetID)
	rawValues, err := FetchSheetData(ctx, sheetID, gid, credsPath)
	if err != nil {
		return err
	}

	// format raw data for LLM
	log.Printf("[Parser] Sending data to DeepSeek for analysis...")
	tableText := FormatForLLM(rawValues)

	// send request to deepseek
	llmStruct, err := AnalyzeTableWithDeepSeek(ctx, deepseekKey, tableText)
	if err != nil {
		return err
	}

	// save column indexes to db
	_, err = db.ExecContext(ctx,
		`UPDATE subjects SET name_column_index = $1, data_start_row = $2 WHERE id = $3`,
		llmStruct.StudentNameColumnIndex, llmStruct.DataStartRow, subjectID)
	if err != nil {
		return fmt.Errorf("db update error: %w", err)
	}

	// save grade struct
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, node := range llmStruct.Structure {
		if err := insertNode(tx, subjectID, nil, node); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// recursive function to save grade structure nodes
// depth 3+
func insertNode(tx *sqlx.Tx, subjectID int64, parentID *int64, node LLMGradeNode) error {
	var newID int64
	query := `INSERT INTO grade_structures (subject_id, parent_id, name, type, column_index, weight, display_formula)
              VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`

	err := tx.QueryRowx(query, subjectID, parentID, node.Name, node.Type, node.ColumnIndex, node.Weight, node.DisplayFormula).Scan(&newID)
	if err != nil {
		return fmt.Errorf("insert node '%s' error: %w", node.Name, err)
	}

	// recursion for children
	for _, child := range node.Children {
		if err := insertNode(tx, subjectID, &newID, child); err != nil {
			return err
		}
	}
	return nil
}
