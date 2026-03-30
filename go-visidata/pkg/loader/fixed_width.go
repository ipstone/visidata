package loader

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/ipstone/visidata/go-visidata/pkg/sheet"
)

type fixedColumnRange struct {
	start int
	end   int
}

func loadFixed(path string, reader *os.File, opts Options) (*sheet.Sheet, error) {
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	lines, err := readFixedLines(reader)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("fixed-width file is empty")
	}

	headerCount := opts.Header
	if headerCount < 0 {
		headerCount = 0
	}
	hasHeader := headerCount > 0

	ranges := columnize(lines, hasHeader)
	if len(ranges) == 0 {
		ranges = []fixedColumnRange{{start: 0, end: maxRuneLen(lines)}}
	}

	headers := fixedHeaders(lines, ranges, hasHeader)
	sh := sheet.New(path, path, headers)

	startRow := 0
	if hasHeader {
		startRow = 1
	}
	for _, line := range lines[startRow:] {
		sh.AddRawRow(extractFixedRow(line, ranges))
	}

	sh.InferColumnKinds()
	return sh, nil
}

func readFixedLines(reader io.Reader) ([]string, error) {
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

func columnize(lines []string, hasHeader bool) []fixedColumnRange {
	if len(lines) == 0 {
		return nil
	}

	detectLines := lines
	if hasHeader {
		detectLines = lines[:1]
	}
	runeLines := toRuneLines(lines)
	detectRuneLines := toRuneLines(detectLines)

	colStarts := make([]int, 0)
	for i := 0; i < maxRuneLen(detectLines); i++ {
		allSpace := true
		prevAllSpace := true
		for _, row := range detectRuneLines {
			if i < len(row) && !isSpace(row[i]) {
				allSpace = false
			}
			if i > 0 && i-1 < len(row) && !isSpace(row[i-1]) {
				prevAllSpace = false
			}
		}
		if !allSpace && (i == 0 || prevAllSpace) {
			colStarts = append(colStarts, i)
		}
	}
	if len(colStarts) == 0 {
		return nil
	}

	allNonspaces := make(map[int]struct{})
	for _, row := range runeLines {
		for i, r := range row {
			if !isSpace(r) {
				allNonspaces[i] = struct{}{}
			}
		}
	}

	ranges := make([]fixedColumnRange, 0, len(colStarts))
	for idx, start := range colStarts {
		end := start
		if idx+1 < len(colStarts) {
			nextStart := colStarts[idx+1]
			for pos := start; pos < nextStart; pos++ {
				if _, ok := allNonspaces[pos]; ok {
					end = pos + 1
				}
			}
		} else {
			for pos := range allNonspaces {
				if pos >= start && pos+1 > end {
					end = pos + 1
				}
			}
		}
		ranges = append(ranges, fixedColumnRange{start: start, end: end})
	}

	return slices.DeleteFunc(ranges, func(r fixedColumnRange) bool { return r.end <= r.start })
}

func fixedHeaders(lines []string, ranges []fixedColumnRange, hasHeader bool) []string {
	if !hasHeader {
		return make([]string, len(ranges))
	}

	headers := extractFixedRow(lines[0], ranges)
	for i := range headers {
		headers[i] = strings.TrimSpace(headers[i])
	}
	return headers
}

func extractFixedRow(line string, ranges []fixedColumnRange) []string {
	runes := []rune(line)
	row := make([]string, len(ranges))
	for i, r := range ranges {
		start := min(r.start, len(runes))
		end := min(r.end, len(runes))
		if end < start {
			end = start
		}
		row[i] = string(runes[start:end])
	}
	return row
}

func toRuneLines(lines []string) [][]rune {
	rows := make([][]rune, len(lines))
	for i, line := range lines {
		rows[i] = []rune(line)
	}
	return rows
}

func maxRuneLen(lines []string) int {
	maxLen := 0
	for _, line := range lines {
		if l := len([]rune(line)); l > maxLen {
			maxLen = l
		}
	}
	return maxLen
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}
