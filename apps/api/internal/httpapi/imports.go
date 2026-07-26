package httpapi

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/mikhailbovt/FORMA/apps/api/internal/importer"
)

const maxImportMultipartParts = 16

func (api *API) previewImport(writer http.ResponseWriter, request *http.Request) {
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil {
		writeError(writer, request, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be multipart/form-data.", nil)
		return
	}
	if mediaType == "application/json" || strings.HasSuffix(mediaType, "+json") {
		writeError(writer, request, http.StatusUnprocessableEntity, "url_import_not_supported", "Forma never fetches LinkedIn profile URLs. Save the URL as a profile link, or upload your own LinkedIn data-export ZIP or PDF.", nil)
		return
	}
	if mediaType != "multipart/form-data" {
		writeError(writer, request, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be multipart/form-data.", nil)
		return
	}

	request.Body = http.MaxBytesReader(writer, request.Body, importer.MaxUploadBytes)
	reader, err := request.MultipartReader()
	if err != nil {
		writeError(writer, request, http.StatusBadRequest, "invalid_multipart", "The multipart request is malformed.", nil)
		return
	}

	var filename, partMediaType string
	var fileData []byte
	fileSeen := false
	partsSeen := 0
	for {
		part, nextErr := reader.NextPart()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			status, code, message := multipartReadError(nextErr)
			writeError(writer, request, status, code, message, nil)
			return
		}
		partsSeen++
		if partsSeen > maxImportMultipartParts {
			_ = part.Close()
			writeError(writer, request, http.StatusBadRequest, "too_many_parts", "The multipart request contains too many fields.", nil)
			return
		}
		if part.FormName() != "file" {
			_, _ = io.Copy(io.Discard, io.LimitReader(part, 8<<10))
			_ = part.Close()
			continue
		}
		if fileSeen {
			_ = part.Close()
			writeError(writer, request, http.StatusBadRequest, "multiple_files", "Upload exactly one file field.", nil)
			return
		}
		fileSeen = true
		filename = strings.TrimSpace(part.FileName())
		partMediaType = part.Header.Get("Content-Type")
		fileData, err = io.ReadAll(io.LimitReader(part, importer.MaxFileBytes+1))
		_ = part.Close()
		if err != nil {
			status, code, message := multipartReadError(err)
			writeError(writer, request, status, code, message, nil)
			return
		}
		if int64(len(fileData)) > importer.MaxFileBytes {
			writeError(writer, request, http.StatusRequestEntityTooLarge, "file_too_large", "The uploaded file exceeds the 12 MiB import limit.", nil)
			return
		}
	}
	if !fileSeen || filename == "" {
		writeError(writer, request, http.StatusBadRequest, "file_required", "Upload one file in the multipart field named file.", nil)
		return
	}

	preview, err := importer.PreviewFile(request.Context(), filename, partMediaType, fileData)
	if err != nil {
		code := importer.ErrorCode(err)
		writeError(writer, request, importHTTPStatus(code), code, err.Error(), nil)
		return
	}
	writeData(writer, http.StatusOK, preview)
}

func multipartReadError(err error) (int, string, string) {
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) || strings.Contains(strings.ToLower(err.Error()), "request body too large") {
		return http.StatusRequestEntityTooLarge, "upload_too_large", "The multipart upload exceeds the 16 MiB request limit."
	}
	return http.StatusBadRequest, "invalid_multipart", "The multipart request is malformed."
}

func importHTTPStatus(code string) int {
	switch code {
	case "file_too_large", "json_too_large", "archive_entry_limit", "archive_entry_too_large", "archive_expansion_limit", "archive_ratio_limit", "pdf_page_limit", "pdf_text_limit":
		return http.StatusRequestEntityTooLarge
	case "unsupported_format":
		return http.StatusUnsupportedMediaType
	default:
		return http.StatusUnprocessableEntity
	}
}
