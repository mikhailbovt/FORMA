package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
)

type errorEnvelope struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeData(writer http.ResponseWriter, status int, value any) {
	writeJSON(writer, status, map[string]any{"data": value})
}

func writeError(writer http.ResponseWriter, request *http.Request, status int, code, message string, fields map[string]string) {
	writeJSON(writer, status, errorEnvelope{Error: apiError{
		Code: code, Message: message, Fields: fields, RequestID: middleware.GetReqID(request.Context()),
	}})
}

func decodeJSON(writer http.ResponseWriter, request *http.Request, target any) error {
	contentType := request.Header.Get("Content-Type")
	if contentType == "" {
		return &requestError{Status: http.StatusUnsupportedMediaType, Code: "unsupported_media_type", Message: "Content-Type must be application/json"}
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "application/json" {
		return &requestError{Status: http.StatusUnsupportedMediaType, Code: "unsupported_media_type", Message: "Content-Type must be application/json"}
	}
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			return &requestError{Status: http.StatusRequestEntityTooLarge, Code: "body_too_large", Message: "Request body is too large"}
		}
		if errors.Is(err, io.EOF) {
			return &requestError{Status: http.StatusBadRequest, Code: "invalid_json", Message: "Request body must contain a JSON object"}
		}
		return &requestError{Status: http.StatusBadRequest, Code: "invalid_json", Message: "Request body contains invalid JSON"}
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return &requestError{Status: http.StatusBadRequest, Code: "invalid_json", Message: "Request body must contain a single JSON object"}
	}
	return nil
}

type requestError struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
}

func (e *requestError) Error() string { return e.Message }

func writeRequestError(writer http.ResponseWriter, request *http.Request, err error) {
	var typed *requestError
	if errors.As(err, &typed) {
		writeError(writer, request, typed.Status, typed.Code, typed.Message, typed.Fields)
		return
	}
	writeError(writer, request, http.StatusBadRequest, "bad_request", strings.TrimSpace(err.Error()), nil)
}
