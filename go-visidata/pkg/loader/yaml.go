package loader

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/ipstone/visidata/go-visidata/pkg/sheet"
)

func loadYAML(path string, reader *os.File) (*sheet.Sheet, error) {
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	decoder := yaml.NewDecoder(reader)
	var records []any
	for {
		var value any
		if err := decoder.Decode(&value); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decode yaml: %w", err)
		}
		records = append(records, structuredRecords(value)...)
	}

	return recordsToSheet(path, path, records), nil
}
