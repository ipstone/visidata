package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ipstone/visidata/go-visidata/pkg/sheet"
)

func loadDir(path string, opts Options) (*sheet.Sheet, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}

	headers := []string{"directory", "filename", "ext", "size", "modtime"}
	sh := sheet.New(path, path, headers)

	for _, entry := range entries {
		name := entry.Name()
		if !opts.ShowHidden && strings.HasPrefix(name, ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("stat directory entry %q: %w", name, err)
		}

		sh.AddRow([]string{
			filepath.Clean(path) + string(os.PathSeparator),
			name,
			dirEntryExt(entry),
			strconv.FormatInt(info.Size(), 10),
			info.ModTime().Format(time.RFC3339),
		})
	}

	sh.InferColumnKinds()
	return sh, nil
}

func dirEntryExt(entry os.DirEntry) string {
	if entry.IsDir() {
		return "/"
	}
	return strings.TrimPrefix(filepath.Ext(entry.Name()), ".")
}
