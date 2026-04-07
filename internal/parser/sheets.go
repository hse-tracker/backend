package parser

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// extract sheet ID from url
// example: https://docs.google.com/spreadsheets/d/1BxiMVs0X_5u.../edit
// depth 3
func ExtractSheetInfo(url string) (sheetID string, gid int64, err error) {
	// get sheet ID
	reID := regexp.MustCompile(`/d/([a-zA-Z0-9-_]+)`)
	matchesID := reID.FindStringSubmatch(url)
	if len(matchesID) < 2 {
		return "", 0, fmt.Errorf("invalid google sheets url")
	}
	sheetID = matchesID[1]

	// get gid
	reGid := regexp.MustCompile(`gid=([0-9]+)`)
	matchesGid := reGid.FindStringSubmatch(url)
	if len(matchesGid) >= 2 {
		parsedGid, _ := strconv.ParseInt(matchesGid[1], 10, 64)
		gid = parsedGid
	}

	// debug log
	log.Println("- - - ExtractSheetInfo - - -")
	log.Println(sheetID, gid)

	return sheetID, gid, nil
}

// get raw data (5 rows with formulas)
// depth 3
func FetchSheetData(ctx context.Context, sheetID string, gid int64, credsPath string) ([][]interface{}, error) {
	srv, err := sheets.NewService(ctx, option.WithCredentialsFile(credsPath))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Sheets client: %w", err)
	}

	// get metadata to get sheet name by gid
	doc, err := srv.Spreadsheets.Get(sheetID).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to get spreadsheet metadata: %w", err)
	}

	var sheetName string
	for _, sheet := range doc.Sheets {
		if sheet.Properties.SheetId == gid {
			sheetName = sheet.Properties.Title
			break
		}
	}

	// if sheet with provided gid not found, take 1st by default
	if sheetName == "" && len(doc.Sheets) > 0 {
		sheetName = doc.Sheets[0].Properties.Title
	}

	// get raw data
	readRange := fmt.Sprintf("'%s'!A1:Z5", sheetName)
	resp, err := srv.Spreadsheets.Values.Get(sheetID, readRange).ValueRenderOption("FORMULA").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve data from sheet: %w", err)
	}

	return resp.Values, nil
}

// convert raw sheets data to readable format for LLM
// depth 3
func FormatForLLM(values [][]interface{}) string {
	var sb strings.Builder
	rowValues := make([]string, 0, 26)

	for i, row := range values {
		sb.WriteString(fmt.Sprintf("Строка %d: ", i+1))

		rowValues = rowValues[:0]

		for j, val := range row {
			colLetter := string(rune('A' + j))
			rowValues = append(rowValues, fmt.Sprintf("%s=\"%v\"", colLetter, val))
		}
		sb.WriteString(strings.Join(rowValues, ", ") + "\n")
	}

	// debug log
	log.Println("- - - FormatForLLM - - -")
	log.Println(sb.String())

	return sb.String()
}

// get all raw sheet sheet data (without formulas)
func FetchSheetDataForSync(ctx context.Context, sheetID string, gid int64, credsPath string) ([][]interface{}, error) {
	srv, err := sheets.NewService(ctx, option.WithCredentialsFile(credsPath))
	if err != nil {
		return nil, fmt.Errorf("unable to get Sheets client: %w", err)
	}

	doc, err := srv.Spreadsheets.Get(sheetID).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to get spreadsheet metadata: %w", err)
	}

	var sheetName string
	for _, sheet := range doc.Sheets {
		if sheet.Properties.SheetId == gid {
			sheetName = sheet.Properties.Title
			break
		}
	}
	if sheetName == "" && len(doc.Sheets) > 0 {
		sheetName = doc.Sheets[0].Properties.Title
	}

	// medium data range
	readRange := fmt.Sprintf("'%s'!A1:AZ200", sheetName)

	// get sheet data with UNFORMATTED_VALUE
	resp, err := srv.Spreadsheets.Values.Get(sheetID, readRange).ValueRenderOption("UNFORMATTED_VALUE").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve data from sheet: %w", err)
	}

	return resp.Values, nil
}
