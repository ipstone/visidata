package loader

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/ipstone/visidata/go-visidata/pkg/sheet"
)

func loadJSON(path string, reader *os.File, fileFormat format) (*sheet.Sheet, error) {
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	name := path
	if path == "-" {
		name = "stdin"
	}

	records, err := decodeJSONRecords(reader, fileFormat)
	if err != nil {
		return nil, err
	}

	return recordsToSheet(name, path, records), nil
}

func decodeJSONRecords(reader io.Reader, fileFormat format) ([]any, error) {
	if fileFormat == formatJSONL {
		records, err := decodeJSONLines(reader)
		if err == nil {
			return records, nil
		}

		if seeker, ok := reader.(io.Seeker); ok {
			if _, seekErr := seeker.Seek(0, io.SeekStart); seekErr != nil {
				return nil, seekErr
			}
		}
	}

	return decodeJSONDocument(reader)
}

func decodeJSONLines(reader io.Reader) ([]any, error) {
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var records []any
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || isCommentLine(line) {
			continue
		}

		var value any
		if err := json.Unmarshal([]byte(line), &value); err != nil {
			return nil, fmt.Errorf("decode jsonl line %d: %w", lineNo, err)
		}
		records = append(records, value)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func decodeJSONDocument(reader io.Reader) ([]any, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	content = stripLeadingJSONComments(content)

	var value any
	if err := json.Unmarshal(content, &value); err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}

	switch typed := value.(type) {
	case []any:
		return typed, nil
	default:
		return []any{typed}, nil
	}
}

func stripLeadingJSONComments(content []byte) []byte {
	for {
		content = bytes.TrimLeft(content, " \t\r\n")
		switch {
		case bytes.HasPrefix(content, []byte("#")):
			if idx := bytes.IndexByte(content, '\n'); idx >= 0 {
				content = content[idx+1:]
				continue
			}
			return nil
		case bytes.HasPrefix(content, []byte("//")):
			if idx := bytes.IndexByte(content, '\n'); idx >= 0 {
				content = content[idx+1:]
				continue
			}
			return nil
		default:
			return content
		}
	}
}

func recordsToSheet(name, source string, records []any) *sheet.Sheet {
	headers := collectHeaders(records)
	sh := sheet.New(name, source, headers)

	for _, record := range records {
		sh.AddRow(recordToRow(headers, record))
	}

	sh.InferColumnKinds()
	return sh
}

func collectHeaders(records []any) []string {
	keys := make(map[string]struct{})
	for _, record := range records {
		if obj, ok := record.(map[string]any); ok {
			for key := range obj {
				keys[key] = struct{}{}
			}
		} else {
			keys["value"] = struct{}{}
		}
	}

	headers := make([]string, 0, len(keys))
	for key := range keys {
		headers = append(headers, key)
	}
	sort.Strings(headers)

	if len(headers) == 0 {
		headers = []string{"value"}
	}
	return headers
}

func recordToRow(headers []string, record any) []string {
	row := make([]string, len(headers))

	if obj, ok := record.(map[string]any); ok {
		for i, header := range headers {
			row[i] = stringifyJSONValue(obj[header])
		}
		return row
	}

	if len(headers) > 0 {
		row[0] = stringifyJSONValue(record)
	}
	return row
}

func stringifyJSONValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case json.Number:
		return typed.String()
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		return string(encoded)
	}
}
