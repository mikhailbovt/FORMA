package importer

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"strings"
)

const (
	maxArchiveEntries        = 512
	maxArchiveBytes   uint64 = 64 << 20
	maxArchiveEntry   uint64 = 8 << 20
)

type safeArchive struct {
	files map[string]*zip.File
}

func parseArchive(ctx context.Context, filename string, data []byte) (draft, error) {
	archive, err := openSafeArchive(data)
	if err != nil {
		return draft{}, err
	}
	if _, ok := archive.files["word/document.xml"]; ok {
		return parseDOCX(ctx, filename, archive)
	}
	return parseLinkedInArchive(ctx, filename, archive)
}

func openSafeArchive(data []byte) (*safeArchive, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, invalid("invalid_archive", "The ZIP or DOCX archive is malformed.", err)
	}
	if len(reader.File) == 0 || len(reader.File) > maxArchiveEntries {
		return nil, invalid("archive_entry_limit", "The archive contains too many entries.", nil)
	}
	files := make(map[string]*zip.File, len(reader.File))
	var total uint64
	for _, file := range reader.File {
		cleanName, ok := safeArchivePath(file.Name)
		if !ok {
			return nil, invalid("unsafe_archive_path", "The archive contains an unsafe file path.", nil)
		}
		if file.FileInfo().IsDir() {
			continue
		}
		total += file.UncompressedSize64
		if total > maxArchiveBytes {
			return nil, invalid("archive_expansion_limit", "The archive expands beyond the 64 MiB safety limit.", nil)
		}
		if file.UncompressedSize64 > 1<<20 && file.CompressedSize64 > 0 && file.UncompressedSize64/file.CompressedSize64 > 200 {
			return nil, invalid("archive_ratio_limit", "The archive contains a suspiciously compressed entry.", nil)
		}
		key := strings.ToLower(cleanName)
		if _, exists := files[key]; exists {
			return nil, invalid("duplicate_archive_path", "The archive contains duplicate file paths.", nil)
		}
		files[key] = file
	}
	return &safeArchive{files: files}, nil
}

func safeArchivePath(name string) (string, bool) {
	name = strings.ReplaceAll(name, "\\", "/")
	if strings.HasPrefix(name, "/") || strings.Contains(name, "\x00") {
		return "", false
	}
	clean := path.Clean(name)
	if clean == "." || clean == ".." || path.IsAbs(clean) || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") || isWindowsDrivePath(clean) {
		return "", false
	}
	return clean, true
}

func isWindowsDrivePath(name string) bool {
	return len(name) >= 2 && ((name[0] >= 'a' && name[0] <= 'z') || (name[0] >= 'A' && name[0] <= 'Z')) && name[1] == ':'
}

func (archive *safeArchive) read(file *zip.File, maximum uint64) ([]byte, error) {
	if file == nil {
		return nil, fmt.Errorf("archive entry is missing")
	}
	if maximum == 0 || maximum > maxArchiveEntry {
		maximum = maxArchiveEntry
	}
	if file.UncompressedSize64 > maximum {
		return nil, invalid("archive_entry_too_large", "An archive entry exceeds its safety limit.", nil)
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, int64(maximum)+1))
	if err != nil {
		return nil, err
	}
	if uint64(len(data)) > maximum {
		return nil, invalid("archive_entry_too_large", "An archive entry exceeds its safety limit.", nil)
	}
	return data, nil
}
