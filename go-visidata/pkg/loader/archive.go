package loader

import (
	"archive/tar"
	"archive/zip"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/ipstone/visidata/go-visidata/pkg/sheet"
)

func loadArchive(path string, reader *os.File) (*sheet.Sheet, error) {
	switch normalizedExt(path) {
	case ".zip":
		return loadZipArchive(path, reader)
	default:
		return loadTarArchive(path, reader)
	}
}

func loadZipArchive(path string, reader *os.File) (*sheet.Sheet, error) {
	info, err := reader.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat zip archive: %w", err)
	}
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	zr, err := zip.NewReader(reader, info.Size())
	if err != nil {
		return nil, fmt.Errorf("read zip archive: %w", err)
	}

	sh := archiveSheet(path)
	for _, file := range zr.File {
		sh.AddRawRow([]string{
			file.Name,
			fmt.Sprintf("%d", file.UncompressedSize64),
			fmt.Sprintf("%d", file.CompressedSize64),
			file.Modified.Format(timeLayout),
			strconv.FormatBool(file.FileInfo().IsDir()),
		})
	}
	sh.InferColumnKinds()
	return sh, nil
}

func loadTarArchive(path string, reader *os.File) (*sheet.Sheet, error) {
	if _, err := reader.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	tr := tar.NewReader(reader)
	sh := archiveSheet(path)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar archive: %w", err)
		}
		sh.AddRawRow([]string{
			header.Name,
			fmt.Sprintf("%d", header.Size),
			"",
			header.ModTime.Format(timeLayout),
			strconv.FormatBool(header.FileInfo().IsDir()),
		})
	}
	sh.InferColumnKinds()
	return sh, nil
}

func archiveSheet(path string) *sheet.Sheet {
	sh := sheet.New(path, path, []string{"path", "size", "compressed_size", "modtime", "is_dir"})
	sh.Columns[1].Kind = sheet.KindInt
	sh.Columns[2].Kind = sheet.KindInt
	sh.Columns[3].Kind = sheet.KindDate
	sh.Columns[4].Kind = sheet.KindBool
	return sh
}

const timeLayout = "2006-01-02T15:04:05Z07:00"
