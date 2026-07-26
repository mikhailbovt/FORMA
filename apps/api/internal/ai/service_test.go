package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestProviderAdapters(t *testing.T) {
	t.Parallel()
	valid := validReviewJSON
	tests := []struct {
		name         string
		provider     string
		expectedPath string
		respond      func(http.ResponseWriter)
	}{
		{name: "OpenAI Responses", provider: "openai", expectedPath: "/responses", respond: func(w http.ResponseWriter) {
			writeTestJSON(w, map[string]any{"output": []any{map[string]any{"content": []any{map[string]any{"type": "output_text", "text": valid}}}}})
		}},
		{name: "Anthropic Messages", provider: "anthropic", expectedPath: "/messages", respond: func(w http.ResponseWriter) {
			writeTestJSON(w, map[string]any{"content": []any{map[string]any{"type": "tool_use", "name": "resume_review", "input": json.RawMessage(valid)}}})
		}},
		{name: "Gemini generateContent", provider: "gemini", expectedPath: "/models/test-model:generateContent", respond: func(w http.ResponseWriter) {
			writeTestJSON(w, map[string]any{"candidates": []any{map[string]any{"content": map[string]any{"parts": []any{map[string]any{"text": valid}}}}}})
		}},
		{name: "OpenAI-compatible chat", provider: "deepseek", expectedPath: "/chat/completions", respond: func(w http.ResponseWriter) {
			writeTestJSON(w, map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": valid}}}})
		}},
		{name: "Ollama chat", provider: "ollama", expectedPath: "/api/chat", respond: func(w http.ResponseWriter) {
			writeTestJSON(w, map[string]any{"message": map[string]any{"content": valid}})
		}},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Path != test.expectedPath {
					t.Errorf("path = %q, want %q", request.URL.Path, test.expectedPath)
				}
				if request.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Content-Type = %q", request.Header.Get("Content-Type"))
				}
				var body map[string]any
				if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
					t.Errorf("decode request: %v", err)
				}
				encoded, _ := json.Marshal(body)
				if strings.Contains(string(encoded), "Alice Example") || strings.Contains(string(encoded), "alice@example.com") || strings.Contains(string(encoded), "secret") {
					t.Errorf("provider request leaked personal data: %s", encoded)
				}
				if request.URL.RawQuery != "" {
					t.Errorf("provider URL unexpectedly contains query data: %q", request.URL.RawQuery)
				}
				assertAdapterRequest(t, test.provider, request, body)
				test.respond(writer)
			}))
			defer server.Close()

			service := NewService(server.Client())
			baseURL := server.URL
			if test.provider == "ollama" {
				baseURL += "/v1"
			}
			result, err := service.Review(context.Background(), Session{
				Provider: test.provider, Model: "test-model", APIKey: "secret", BaseURL: baseURL,
			}, ReviewRequest{Resume: json.RawMessage(`{"basics":{"name":"Alice Example","email":"alice@example.com","headline":"Engineer"},"summary":"Builds reliable APIs"}`)})
			if err != nil {
				t.Fatalf("Review() error = %v", err)
			}
			if result.Summary != "Strong base" || len(result.Assessments) != 4 {
				t.Fatalf("Review() = %#v, want summary and four semantic assessments", result)
			}
		})
	}
}

func assertAdapterRequest(t *testing.T, provider string, request *http.Request, body map[string]any) {
	t.Helper()
	switch provider {
	case "openai":
		text, ok := body["text"].(map[string]any)
		if !ok || text["format"] == nil || request.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("invalid OpenAI Responses request: body=%#v headers=%v", body, request.Header)
		}
	case "anthropic":
		if body["tools"] == nil || body["tool_choice"] == nil || request.Header.Get("x-api-key") != "secret" || request.Header.Get("anthropic-version") == "" {
			t.Errorf("invalid Anthropic Messages request: body=%#v headers=%v", body, request.Header)
		}
	case "gemini":
		if body["generationConfig"] == nil || body["systemInstruction"] == nil || request.Header.Get("x-goog-api-key") != "secret" {
			t.Errorf("invalid Gemini request: body=%#v headers=%v", body, request.Header)
		}
	case "deepseek":
		if body["response_format"] == nil || body["messages"] == nil || request.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("invalid OpenAI-compatible request: body=%#v headers=%v", body, request.Header)
		}
	case "ollama":
		if body["format"] == nil || body["stream"] != false {
			t.Errorf("invalid Ollama request: body=%#v", body)
		}
	}
}

func TestJSONObjectProviderGetsOneRepairRetry(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			writeTestJSON(writer, map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": `{"not":"the schema"}`}}}})
			return
		}
		repaired := `{"summary":"Fixed","assessments":` + validAssessmentsJSON + `,"suggestions":[],"warnings":[]}`
		writeTestJSON(writer, map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": repaired}}}})
	}))
	defer server.Close()

	service := NewService(server.Client())
	_, err := service.Review(context.Background(), Session{
		Provider: "deepseek", Model: "editable-model", APIKey: "secret", BaseURL: server.URL,
	}, ReviewRequest{Resume: json.RawMessage(`{"summary":"Built APIs"}`)})
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("provider calls = %d, want exactly 2", calls.Load())
	}
}

func writeTestJSON(writer http.ResponseWriter, value any) {
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(value)
}
