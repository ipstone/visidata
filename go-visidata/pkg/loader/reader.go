package loader

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strings"
)

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
			closeErr := f.Close()
			removeErr := os.Remove(name)
			return errors.Join(closeErr, removeErr)
		}, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	return f, f.Close, nil
}

func firstDataLine(f *os.File) (string, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || isCommentLine(line) {
			continue
		}
		return line, nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", nil
}

// isCommentLine is used by JSON/JSONL detection and parsing so that simple
// fixture-style comment lines beginning with # or // are skipped.
func isCommentLine(line string) bool {
	return strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//")
}
