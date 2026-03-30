package loader

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/ipstone/visidata/go-visidata/pkg/sheet"
)

func loadSQLite(path string, opts Options) (*sheet.Sheet, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	defer db.Close()

	tableName := opts.Table
	if tableName == "" {
		tableName, err = firstSQLiteTable(db)
		if err != nil {
			return nil, err
		}
	}

	// This loader needs every column from the selected table so the resulting
	// sheet mirrors the full table schema.
	query := fmt.Sprintf("SELECT * FROM %s", quoteSQLiteIdentifier(tableName))
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query sqlite table %q: %w", tableName, err)
	}
	defer rows.Close()

	headers, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("sqlite columns: %w", err)
	}

	name := fmt.Sprintf("%s:%s", filepath.Base(path), tableName)
	sh := sheet.New(name, path, headers)

	for rows.Next() {
		values, err := scanSQLiteRow(rows, len(headers))
		if err != nil {
			return nil, err
		}
		sh.AddRow(values)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sqlite rows: %w", err)
	}

	sh.InferColumnKinds()
	return sh, nil
}

func firstSQLiteTable(db *sql.DB) (string, error) {
	row := db.QueryRow(`
		SELECT name
		FROM sqlite_master
		WHERE type = 'table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
		LIMIT 1
	`)

	var tableName string
	if err := row.Scan(&tableName); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("sqlite database has no user tables")
		}
		return "", fmt.Errorf("list sqlite tables: %w", err)
	}
	return tableName, nil
}

func scanSQLiteRow(rows *sql.Rows, width int) ([]string, error) {
	raw := make([]any, width)
	dest := make([]any, width)
	for i := range raw {
		dest[i] = &raw[i]
	}

	if err := rows.Scan(dest...); err != nil {
		return nil, fmt.Errorf("scan sqlite row: %w", err)
	}

	record := make([]string, width)
	for i, value := range raw {
		record[i] = stringifySQLiteValue(value)
	}
	return record, nil
}

func stringifySQLiteValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case []byte:
		return string(typed)
	default:
		return fmt.Sprint(typed)
	}
}

func quoteSQLiteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
