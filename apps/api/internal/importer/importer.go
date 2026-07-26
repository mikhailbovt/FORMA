package importer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/resume"
)

func PreviewFile(ctx context.Context, filename, mediaType string, data []byte) (preview Preview, err error) {
	if err := ctx.Err(); err != nil {
		return Preview{}, err
	}
	if len(data) == 0 {
		return Preview{}, invalid("empty_file", "The uploaded file is empty.", nil)
	}
	if int64(len(data)) > MaxFileBytes {
		return Preview{}, invalid("file_too_large", "The uploaded file exceeds the 12 MiB import limit.", nil)
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			preview = Preview{}
			err = invalid("malformed_file", "The file could not be parsed safely.", nil)
		}
	}()

	hash := sha256.Sum256(data)
	extension := strings.ToLower(filepath.Ext(filename))
	trimmed := bytes.TrimSpace(data)

	var result draft
	switch {
	case bytes.HasPrefix(trimmed, []byte("%PDF-")) || extension == ".pdf":
		result, err = parsePDF(ctx, filename, data)
	case bytes.HasPrefix(data, []byte("PK\x03\x04")) || bytes.HasPrefix(data, []byte("PK\x05\x06")) || extension == ".docx" || extension == ".zip":
		result, err = parseArchive(ctx, filename, data)
	case len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') || extension == ".json" || strings.Contains(strings.ToLower(mediaType), "json"):
		result, err = parseJSON(filename, data)
	default:
		err = invalid("unsupported_format", "Supported imports are Forma JSON, JSON Resume, LinkedIn ZIP, DOCX, and text-layer PDF.", nil)
	}
	if err != nil {
		return Preview{}, err
	}

	result.candidate.Title = strings.TrimSpace(result.candidate.Title)
	if result.candidate.Title == "" {
		result.candidate.Title = importedTitle(filename, result.candidate.Document.Basics.Name)
	}
	setDocumentDefaults(&result.candidate.Document)
	if problems := result.candidate.NormalizeAndValidate(); len(problems) > 0 {
		return Preview{}, invalid("invalid_candidate", "The imported content does not fit Forma's resume schema.", nil)
	}
	sort.SliceStable(result.mappings, func(i, j int) bool {
		if result.mappings[i].Path == result.mappings[j].Path {
			return result.mappings[i].SourceLocator < result.mappings[j].SourceLocator
		}
		return result.mappings[i].Path < result.mappings[j].Path
	})
	sort.SliceStable(result.warnings, func(i, j int) bool {
		if result.warnings[i].Code == result.warnings[j].Code {
			return result.warnings[i].Path < result.warnings[j].Path
		}
		return result.warnings[i].Code < result.warnings[j].Code
	})
	if result.mappings == nil {
		result.mappings = make([]Mapping, 0)
	}
	if result.warnings == nil {
		result.warnings = make([]Warning, 0)
	}
	return Preview{
		Candidate:    result.candidate,
		Parser:       Parser{ID: result.parserID, Version: ParserVersion},
		SourceSHA256: hex.EncodeToString(hash[:]),
		Mappings:     result.mappings,
		Warnings:     result.warnings,
	}, nil
}

type draft struct {
	parserID  string
	candidate resume.Input
	mappings  []Mapping
	warnings  []Warning
}

func (d *draft) mapped(path, source, locator, status string) {
	d.mappings = append(d.mappings, Mapping{Path: path, Source: source, SourceLocator: locator, Status: status})
}

func (d *draft) warn(code, message, path string) {
	d.warnings = append(d.warnings, Warning{Code: code, Message: message, Path: path})
}

func setDocumentDefaults(document *resume.Document) {
	if document.Version == 0 {
		document.Version = 1
	}
	if strings.TrimSpace(document.Template) == "" {
		document.Template = "forma"
	}
	if strings.TrimSpace(document.PageSize) == "" {
		document.PageSize = "A4"
	}
	if strings.TrimSpace(document.Language) == "" {
		document.Language = "en"
	}
	if len(document.Order) == 0 {
		document.Order = []string{"summary", "experience", "projects", "education", "skills", "certifications", "languages"}
	}
}

func importedTitle(filename, name string) string {
	if name = strings.TrimSpace(name); name != "" {
		return truncate(name+" resume", 200)
	}
	base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	base = strings.TrimSpace(base)
	if base == "" {
		base = "Imported"
	}
	return truncate(base+" resume", 200)
}

func stableID(prefix, source string, index int) string {
	hash := sha256.Sum256([]byte(prefix + "\x00" + source + "\x00" + strconv.Itoa(index)))
	return prefix + "-" + hex.EncodeToString(hash[:6])
}

func truncate(value string, maximum int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) > maximum {
		return string(runes[:maximum])
	}
	return string(runes)
}
