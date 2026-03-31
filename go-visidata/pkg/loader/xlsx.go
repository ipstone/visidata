package loader

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"

	"github.com/ipstone/visidata/go-visidata/pkg/sheet"
)

func loadXLSX(path string, reader *os.File, opts Options) (*sheet.Sheet, error) {
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	workbook, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer workbook.Close()

	sheetName := opts.Table
	if sheetName == "" {
		sheets := workbook.GetSheetList()
		if len(sheets) == 0 {
			return nil, fmt.Errorf("xlsx workbook has no sheets")
		}
		sheetName = sheets[0]
	}

	rows, err := workbook.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("read xlsx sheet %q: %w", sheetName, err)
	}
	if len(rows) == 0 {
		return sheet.New(fmt.Sprintf("%s:%s", filepath.Base(path), sheetName), path, nil), nil
	}

	headers := append([]string(nil), rows[0]...)
	if len(headers) == 0 {
		headers = []string{"col1"}
	}

	sh := sheet.New(fmt.Sprintf("%s:%s", filepath.Base(path), sheetName), path, headers)
	for _, row := range rows[1:] {
		sh.AddRow(padRecord(row, len(sh.Columns)))
	}
	sh.InferColumnKinds()
	return sh, nil
}

func padRecord(row []string, width int) []string {
	record := append([]string(nil), row...)
	for len(record) < width {
		record = append(record, "")
	}
	if len(record) > width {
		record = record[:width]
	}
	return record
}
