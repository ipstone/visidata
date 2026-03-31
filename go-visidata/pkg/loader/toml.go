package loader

import (
	"fmt"
	"io"
	"os"

	"github.com/BurntSushi/toml"

	"github.com/ipstone/visidata/go-visidata/pkg/sheet"
)

func loadTOML(path string, reader *os.File) (*sheet.Sheet, error) {
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	var value any
	if _, err := toml.NewDecoder(reader).Decode(&value); err != nil {
		return nil, fmt.Errorf("decode toml: %w", err)
	}

	return recordsToSheet(path, path, structuredRecords(value)), nil
}
