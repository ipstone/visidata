package loader

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ipstone/visidata/go-visidata/pkg/sheet"
)

func loadMarkdown(path string, reader *os.File) (*sheet.Sheet, error) {
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	lines, err := readMarkdownLines(reader)
	if err != nil {
		return nil, err
	}
	headers, rows, err := firstMarkdownTable(lines)
	if err != nil {
		return nil, err
	}

	sh := sheet.New(path, path, headers)
	for _, row := range rows {
		sh.AddRow(row)
	}
	sh.InferColumnKinds()
	return sh, nil
}

func readMarkdownLines(reader io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, strings.TrimRight(scanner.Text(), "\r"))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func firstMarkdownTable(lines []string) ([]string, [][]string, error) {
	for i := 0; i+1 < len(lines); i++ {
		header := markdownCells(lines[i])
		if len(header) == 0 || !isMarkdownSeparator(lines[i+1]) {
			continue
		}

		rows := make([][]string, 0)
		for j := i + 2; j < len(lines); j++ {
			row := markdownCells(lines[j])
			if len(row) == 0 {
				break
			}
			if len(row) < len(header) {
				for len(row) < len(header) {
					row = append(row, "")
				}
			}
			if len(row) > len(header) {
				row = row[:len(header)]
			}
			rows = append(rows, row)
		}
		return header, rows, nil
	}
	return nil, nil, fmt.Errorf("markdown file does not contain a table")
}

func markdownCells(line string) []string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || !strings.Contains(trimmed, "|") {
		return nil
	}
	parts := strings.Split(trimmed, "|")
	if len(parts) >= 2 && strings.TrimSpace(parts[0]) == "" {
		parts = parts[1:]
	}
	if len(parts) >= 1 && strings.TrimSpace(parts[len(parts)-1]) == "" {
		parts = parts[:len(parts)-1]
	}
	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cells = append(cells, strings.TrimSpace(part))
	}
	return cells
}

func isMarkdownSeparator(line string) bool {
	cells := markdownCells(line)
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		cell = strings.TrimSpace(cell)
		cell = strings.Trim(cell, ":")
		if cell == "" {
			return false
		}
		for _, r := range cell {
			if r != '-' {
				return false
			}
		}
	}
	return true
}
