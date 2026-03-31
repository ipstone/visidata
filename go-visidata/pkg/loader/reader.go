package loader

import (
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

func openReader(path string) (*os.File, func() error, error) {
	if path == "-" {
		return materializeReader("vdgo-stdin-*", os.Stdin)
	}

	if isRemotePath(path) {
		resp, err := http.Get(path) //nolint:gosec // user-requested URL fetch
		if err != nil {
			return nil, nil, err
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			defer resp.Body.Close()
			return nil, nil, errors.New(resp.Status)
		}
		return materializeMaybeCompressed(path, resp.Body)
	}

	if compressionKindForPath(path) != "" {
		f, err := os.Open(path)
		if err != nil {
			return nil, nil, err
		}
		return materializeMaybeCompressed(path, f)
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

func materializeMaybeCompressed(path string, reader io.ReadCloser) (*os.File, func() error, error) {
	if compressionKindForPath(path) == "" {
		return materializeReader("vdgo-data-*", reader)
	}

	decode, err := decompressedReader(path, reader)
	if err != nil {
		reader.Close()
		return nil, nil, err
	}
	return materializeReader("vdgo-data-*", combinedReadCloser{
		Reader: decode,
		closers: []io.Closer{
			decode,
			reader,
		},
	})
}

func materializeReader(pattern string, reader io.ReadCloser) (*os.File, func() error, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		reader.Close()
		return nil, nil, err
	}
	if _, err := io.Copy(f, reader); err != nil {
		name := f.Name()
		closeErr := errors.Join(f.Close(), reader.Close())
		removeErr := os.Remove(name)
		return nil, nil, errors.Join(err, closeErr, removeErr)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		name := f.Name()
		closeErr := errors.Join(f.Close(), reader.Close())
		removeErr := os.Remove(name)
		return nil, nil, errors.Join(err, closeErr, removeErr)
	}
	return f, func() error {
		name := f.Name()
		closeErr := f.Close()
		readerErr := reader.Close()
		removeErr := os.Remove(name)
		return errors.Join(closeErr, readerErr, removeErr)
	}, nil
}

func decompressedReader(path string, reader io.ReadCloser) (io.ReadCloser, error) {
	switch compressionKindForPath(path) {
	case "gzip":
		return gzip.NewReader(reader)
	case "bzip2":
		return io.NopCloser(bzip2.NewReader(reader)), nil
	case "xz":
		xr, err := xz.NewReader(reader)
		if err != nil {
			return nil, err
		}
		return io.NopCloser(xr), nil
	case "zstd":
		zr, err := zstd.NewReader(reader)
		if err != nil {
			return nil, err
		}
		return zr.IOReadCloser(), nil
	default:
		return reader, nil
	}
}

func stripCompressionSuffix(path string) string {
	return rewritePathSuffix(path, func(original, lower string) string {
		switch {
		case strings.HasSuffix(lower, ".tgz"):
			return original[:len(original)-4] + ".tar"
		case strings.HasSuffix(lower, ".tbz2"):
			return original[:len(original)-5] + ".tar"
		case strings.HasSuffix(lower, ".txz"):
			return original[:len(original)-4] + ".tar"
		case strings.HasSuffix(lower, ".tzst"):
			return original[:len(original)-5] + ".tar"
		}

		for _, ext := range []string{".gz", ".bz2", ".xz", ".zst"} {
			if strings.HasSuffix(lower, ext) {
				return original[:len(original)-len(ext)]
			}
		}
		return original
	})
}

func compressionKindForPath(path string) string {
	lower := strings.ToLower(pathForSuffix(path))
	switch {
	case strings.HasSuffix(lower, ".gz"), strings.HasSuffix(lower, ".tgz"):
		return "gzip"
	case strings.HasSuffix(lower, ".bz2"), strings.HasSuffix(lower, ".tbz2"):
		return "bzip2"
	case strings.HasSuffix(lower, ".xz"), strings.HasSuffix(lower, ".txz"):
		return "xz"
	case strings.HasSuffix(lower, ".zst"), strings.HasSuffix(lower, ".tzst"):
		return "zstd"
	default:
		return ""
	}
}

func isRemotePath(path string) bool {
	parsed, err := url.Parse(path)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func pathForSuffix(path string) string {
	if !isRemotePath(path) {
		return filepath.Clean(path)
	}
	parsed, err := url.Parse(path)
	if err != nil {
		return path
	}
	if parsed.Path == "" {
		return path
	}
	return parsed.Path
}

func rewritePathSuffix(path string, rewrite func(original, lower string) string) string {
	if !isRemotePath(path) {
		lower := strings.ToLower(path)
		rewritten := rewrite(path, lower)
		if rewritten == path {
			return path
		}
		return rewritten
	}

	parsed, err := url.Parse(path)
	if err != nil {
		return path
	}
	originalPath := parsed.Path
	lowerPath := strings.ToLower(originalPath)
	rewritten := rewrite(originalPath, lowerPath)
	if rewritten == originalPath {
		return path
	}
	parsed.Path = rewritten
	return parsed.String()
}

type combinedReadCloser struct {
	io.Reader
	closers []io.Closer
}

func (c combinedReadCloser) Close() error {
	var errs []error
	for _, closer := range c.closers {
		if closer == nil {
			continue
		}
		errs = append(errs, closer.Close())
	}
	return errors.Join(errs...)
}

// isCommentLine is used by JSON/JSONL detection and parsing so that simple
// fixture-style comment lines beginning with # or // are skipped.
func isCommentLine(line string) bool {
	return strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//")
}
