package importer

import (
	"errors"

	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/resume"
)

const (
	ParserVersion        = "1.0.0"
	MaxUploadBytes int64 = 16 << 20
	MaxFileBytes   int64 = 12 << 20
	MaxTextBytes   int64 = 1 << 20
)

type Parser struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type Mapping struct {
	Path          string `json:"path"`
	Source        string `json:"source"`
	SourceLocator string `json:"source_locator,omitempty"`
	Status        string `json:"status"`
}

type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

type Preview struct {
	Candidate    resume.Input `json:"candidate"`
	Parser       Parser       `json:"parser"`
	SourceSHA256 string       `json:"source_sha256"`
	Mappings     []Mapping    `json:"mappings"`
	Warnings     []Warning    `json:"warnings"`
}

type ImportError struct {
	Code    string
	Message string
	Err     error
}

func (e *ImportError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

func (e *ImportError) Unwrap() error { return e.Err }

func ErrorCode(err error) string {
	var importError *ImportError
	if errors.As(err, &importError) {
		return importError.Code
	}
	return "import_failed"
}

func invalid(code, message string, err error) error {
	return &ImportError{Code: code, Message: message, Err: err}
}
