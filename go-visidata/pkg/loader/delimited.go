package loader

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ipstone/visidata/go-visidata/pkg/sheet"
)

func Load(path string) (*sheet.Sheet, error) {
	sourceName := path
	if path == "" {
		return nil, fmt.Errorf("no input path provided")
	}

	reader, closeFn, err := openReader(path)
	if err != nil {
		return nil, err
	}
	defer closeFn()

	delimiter, err := detectDelimiter(path, reader)
	if err != nil {
		return nil, err
	}

	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	csvReader := csv.NewReader(reader)
	csvReader.Comma = delimiter
	csvReader.FieldsPerRecord = -1
	csvReader.LazyQuotes = true

	headers, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("read headers: %w", err)
	}

	name := sourceName
	if path == "-" {
		name = "stdin"
	}

	sh := sheet.New(name, sourceName, headers)
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read rows: %w", err)
		}
		sh.AddRow(record)
	}

	sh.InferColumnKinds()
	return sh, nil
}

func openReader(path string) (*os.File, func() error, error) {
	if path == "-" {
		f, err := os.CreateTemp("", "vdgo-stdin-*")
		if err != nil {
			return nil, nil, err
		}
		if _, err := io.Copy(f, os.Stdin); err != nil {
			f.Close()
			os.Remove(f.Name())
			return nil, nil, err
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			f.Close()
			os.Remove(f.Name())
			return nil, nil, err
		}
		return f, func() error {
			name := f.Name()
			_ = f.Close()
			return os.Remove(name)
		}, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	return f, f.Close, nil
}

func detectDelimiter(path string, f *os.File) (rune, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".tsv":
		return '\t', nil
	case ".psv":
		return '|', nil
	case ".csv":
		return ',', nil
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return 0, err
		}
		return ',', nil
	}

	line := scanner.Text()
	tabCount := strings.Count(line, "\t")
	commaCount := strings.Count(line, ",")
	pipeCount := strings.Count(line, "|")

	best := ','
	bestCount := commaCount
	if tabCount > bestCount {
		best = '\t'
		bestCount = tabCount
	}
	if pipeCount > bestCount {
		best = '|'
	}

	return best, nil
}
