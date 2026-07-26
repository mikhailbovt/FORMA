package httpapi

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/ai"
	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/importer"
)

func TestPreviewImportAcceptsOneMultipartFile(t *testing.T) {
	body, contentType := importMultipart(t, map[string][]byte{
		"resume.json": []byte(`{"basics":{"name":"Ada Lovelace","label":"Engineer"},"skills":[{"name":"Backend","keywords":["Go"]}]}`),
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/imports/preview", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()
	(&API{}).previewImport(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var envelope struct {
		Data importer.Preview `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.Parser.ID != "json_resume" || envelope.Data.Candidate.Document.Basics.Name != "Ada Lovelace" || envelope.Data.SourceSHA256 == "" {
		t.Fatalf("preview = %#v", envelope.Data)
	}
}

func TestPreviewImportRouteIsWired(t *testing.T) {
	body, contentType := importMultipart(t, map[string][]byte{
		"resume.json": []byte(`{"basics":{"name":"Ada Lovelace"}}`),
	})
	router, sessions := testRouter(t, &fakeResumeStore{})
	defer sessions.Close()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/imports/preview", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"id":"json_resume"`) {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestPreviewImportRouteHasIndependentUploadLimit(t *testing.T) {
	file := largeValidDOCX(t)
	if int64(len(file)) >= importer.MaxFileBytes || len(file) < 11<<20 {
		t.Fatalf("test fixture size = %d, want 11 MiB <= size < %d", len(file), importer.MaxFileBytes)
	}
	body, contentType := importMultipart(t, map[string][]byte{"resume.docx": file})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	sessions := ai.NewSessionStore(time.Minute)
	defer sessions.Close()
	router := New(Dependencies{
		Logger: logger, DB: healthyDB{}, Resumes: &fakeResumeStore{}, Sessions: sessions,
		CORSOrigin: "http://localhost:3000", MaxBodyBytes: 1 << 10,
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/imports/preview", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"id":"docx"`) {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	jsonBody := `{"title":"` + strings.Repeat("x", 2<<10) + `","document":{}}`
	jsonRequest := httptest.NewRequest(http.MethodPost, "/api/v1/resumes", strings.NewReader(jsonBody))
	jsonRequest.Header.Set("Content-Type", "application/json")
	jsonResponse := httptest.NewRecorder()
	router.ServeHTTP(jsonResponse, jsonRequest)
	if jsonResponse.Code != http.StatusRequestEntityTooLarge || !strings.Contains(jsonResponse.Body.String(), "body_too_large") {
		t.Fatalf("configured JSON limit not applied: status = %d, body = %s", jsonResponse.Code, jsonResponse.Body.String())
	}
}

func TestPreviewImportNeverFetchesJSONURLs(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/imports/preview", strings.NewReader(`{"url":"https://linkedin.com/in/example"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	(&API{}).previewImport(response, request)
	if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), "url_import_not_supported") || !strings.Contains(response.Body.String(), "never fetches") {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestPreviewImportLimitsMultipartFieldCount(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for index := 0; index < maxImportMultipartParts+1; index++ {
		if err := writer.WriteField("metadata", "ignored"); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/imports/preview", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	(&API{}).previewImport(response, request)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "too_many_parts") {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestPreviewImportRejectsMissingMultipleAndOversizedFiles(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		_ = writer.WriteField("format", "auto")
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodPost, "/api/v1/imports/preview", &body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		response := httptest.NewRecorder()
		(&API{}).previewImport(response, request)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "file_required") {
			t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
		}
	})

	t.Run("multiple", func(t *testing.T) {
		body, contentType := importMultipart(t, map[string][]byte{"one.json": []byte(`{"basics":{"name":"One"}}`), "two.json": []byte(`{"basics":{"name":"Two"}}`)})
		request := httptest.NewRequest(http.MethodPost, "/api/v1/imports/preview", body)
		request.Header.Set("Content-Type", contentType)
		response := httptest.NewRecorder()
		(&API{}).previewImport(response, request)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "multiple_files") {
			t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
		}
	})

	t.Run("oversized", func(t *testing.T) {
		body, contentType := importMultipart(t, map[string][]byte{"large.json": bytes.Repeat([]byte("x"), int(importer.MaxFileBytes)+1)})
		request := httptest.NewRequest(http.MethodPost, "/api/v1/imports/preview", body)
		request.Header.Set("Content-Type", contentType)
		response := httptest.NewRecorder()
		(&API{}).previewImport(response, request)
		if response.Code != http.StatusRequestEntityTooLarge || !strings.Contains(response.Body.String(), "file_too_large") {
			t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
		}
	})
}

func importMultipart(t *testing.T, files map[string][]byte) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for filename, data := range files {
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return &body, writer.FormDataContentType()
}

func largeValidDOCX(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	document, err := writer.CreateHeader(&zip.FileHeader{Name: "word/document.xml", Method: zip.Store})
	if err != nil {
		t.Fatal(err)
	}
	const documentXML = `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>Ada Lovelace</w:t></w:r></w:p></w:body></w:document>`
	if _, err := document.Write([]byte(documentXML)); err != nil {
		t.Fatal(err)
	}
	padding, err := writer.CreateHeader(&zip.FileHeader{Name: "ignored-padding.bin", Method: zip.Store})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.CopyN(padding, zeroReader{}, importer.MaxFileBytes-(64<<10)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

type zeroReader struct{}

func (zeroReader) Read(buffer []byte) (int, error) {
	clear(buffer)
	return len(buffer), nil
}
