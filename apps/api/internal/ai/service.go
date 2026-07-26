package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type HTTPClient interface {
	Do(request *http.Request) (*http.Response, error)
}

type Service struct {
	client HTTPClient
}

func NewService(client HTTPClient) *Service {
	return &Service{client: client}
}

func NormalizeSession(session Session) (Session, error) {
	provider, ok := FindProvider(session.Provider)
	if !ok {
		return Session{}, errors.New("unsupported provider")
	}
	session.Provider = provider.ID
	session.Model = strings.TrimSpace(session.Model)
	if session.Model == "" {
		session.Model = provider.DefaultModel
	}
	if session.Model == "" || len(session.Model) > 200 || strings.ContainsAny(session.Model, "\r\n") {
		return Session{}, errors.New("model is required and must be at most 200 characters")
	}
	session.APIKey = strings.TrimSpace(session.APIKey)
	if provider.RequiresAPIKey && session.APIKey == "" {
		return Session{}, errors.New("api_key is required for this provider")
	}
	if len(session.APIKey) > 20_000 {
		return Session{}, errors.New("api_key is too long")
	}
	session.BaseURL = strings.TrimRight(strings.TrimSpace(session.BaseURL), "/")
	if session.BaseURL == "" {
		session.BaseURL = provider.DefaultBaseURL
	}
	if session.BaseURL == "" {
		return Session{}, errors.New("base_url is required for this provider")
	}
	parsed, err := url.Parse(session.BaseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return Session{}, errors.New("base_url must be an http(s) origin or path without credentials, query, or fragment")
	}
	if len(session.BaseURL) > 2_048 {
		return Session{}, errors.New("base_url is too long")
	}
	return session, nil
}

func (s *Service) Review(ctx context.Context, session Session, request ReviewRequest) (ReviewResult, error) {
	system, user, schema, err := BuildReviewPrompt(request)
	if err != nil {
		return ReviewResult{}, err
	}
	provider, _ := FindProvider(session.Provider)
	raw, err := s.generate(ctx, provider, session, "resume_review", system, user, schema)
	if err != nil {
		return ReviewResult{}, err
	}
	result, validationErr := decodeReview(raw, user)
	if validationErr == nil {
		return result, nil
	}
	if provider.StructuredOutput != "json_object" {
		return ReviewResult{}, validationErr
	}
	repaired, err := s.repair(ctx, provider, session, "resume_review", system, user, schema, raw, validationErr)
	if err != nil {
		return ReviewResult{}, err
	}
	return decodeReview(repaired, user)
}

func (s *Service) Rewrite(ctx context.Context, session Session, request RewriteRequest) (RewriteResult, error) {
	system, user, schema, err := BuildRewritePrompt(request)
	if err != nil {
		return RewriteResult{}, err
	}
	provider, _ := FindProvider(session.Provider)
	raw, err := s.generate(ctx, provider, session, "resume_rewrite", system, user, schema)
	if err != nil {
		return RewriteResult{}, err
	}
	result, validationErr := decodeRewrite(raw, user)
	if validationErr == nil {
		return result, nil
	}
	if provider.StructuredOutput != "json_object" {
		return RewriteResult{}, validationErr
	}
	repaired, err := s.repair(ctx, provider, session, "resume_rewrite", system, user, schema, raw, validationErr)
	if err != nil {
		return RewriteResult{}, err
	}
	return decodeRewrite(repaired, user)
}

func (s *Service) repair(ctx context.Context, provider Provider, session Session, name, system, original string, schema map[string]any, invalid []byte, validationErr error) ([]byte, error) {
	schemaJSON, _ := json.Marshal(schema)
	invalidText := string(invalid)
	if len(invalidText) > 30_000 {
		invalidText = invalidText[:30_000]
	}
	user := fmt.Sprintf("Correct the previous response. It failed validation: %s\nReturn only corrected JSON matching this schema: %s\nOriginal task and facts:\n%s\nPrevious response:\n%s", validationErr, schemaJSON, original, invalidText)
	return s.generate(ctx, provider, session, name, system, user, schema)
}

func (s *Service) generate(ctx context.Context, provider Provider, session Session, name, system, user string, schema map[string]any) ([]byte, error) {
	switch provider.Protocol {
	case "openai_responses":
		return s.openAIResponses(ctx, session, name, system, user, schema)
	case "anthropic_messages":
		return s.anthropicMessages(ctx, session, name, system, user, schema)
	case "gemini_generate_content":
		return s.geminiGenerateContent(ctx, session, system, user, schema)
	case "openai_chat":
		return s.openAIChat(ctx, session, system, user)
	case "ollama_chat":
		return s.ollamaChat(ctx, session, system, user, schema)
	default:
		return nil, errors.New("unsupported provider protocol")
	}
}

func (s *Service) openAIResponses(ctx context.Context, session Session, name, system, user string, schema map[string]any) ([]byte, error) {
	body := map[string]any{
		"model":        session.Model,
		"instructions": system,
		"input":        user,
		"text": map[string]any{"format": map[string]any{
			"type": "json_schema", "name": name, "strict": true, "schema": schema,
		}},
	}
	responseBody, err := s.postJSON(ctx, endpoint(session.BaseURL, "/responses"), bearerHeaders(session.APIKey), body)
	if err != nil {
		return nil, err
	}
	var response struct {
		Output []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("decode OpenAI response envelope: %w", err)
	}
	for _, output := range response.Output {
		for _, content := range output.Content {
			if content.Type == "output_text" && content.Text != "" {
				return []byte(content.Text), nil
			}
		}
	}
	return nil, errors.New("OpenAI response did not contain output_text")
}

func (s *Service) anthropicMessages(ctx context.Context, session Session, name, system, user string, schema map[string]any) ([]byte, error) {
	body := map[string]any{
		"model":      session.Model,
		"max_tokens": 4_096,
		"system":     system,
		"messages":   []map[string]any{{"role": "user", "content": user}},
		"tools": []map[string]any{{
			"name": name, "description": "Return validated resume editing results", "input_schema": schema,
		}},
		"tool_choice": map[string]any{"type": "tool", "name": name},
	}
	headers := map[string]string{"x-api-key": session.APIKey, "anthropic-version": "2023-06-01"}
	responseBody, err := s.postJSON(ctx, endpoint(session.BaseURL, "/messages"), headers, body)
	if err != nil {
		return nil, err
	}
	var response struct {
		Content []struct {
			Type  string          `json:"type"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("decode Anthropic response envelope: %w", err)
	}
	for _, content := range response.Content {
		if content.Type == "tool_use" && content.Name == name && len(content.Input) > 0 {
			return content.Input, nil
		}
	}
	return nil, errors.New("Anthropic response did not contain the requested tool result")
}

func (s *Service) geminiGenerateContent(ctx context.Context, session Session, system, user string, schema map[string]any) ([]byte, error) {
	model := strings.TrimPrefix(session.Model, "models/")
	body := map[string]any{
		"systemInstruction": map[string]any{"parts": []map[string]string{{"text": system}}},
		"contents":          []map[string]any{{"role": "user", "parts": []map[string]string{{"text": user}}}},
		"generationConfig": map[string]any{
			"temperature": 0.2, "responseMimeType": "application/json", "responseSchema": geminiSchema(schema),
		},
	}
	headers := map[string]string{"x-goog-api-key": session.APIKey}
	path := "/models/" + url.PathEscape(model) + ":generateContent"
	responseBody, err := s.postJSON(ctx, endpoint(session.BaseURL, path), headers, body)
	if err != nil {
		return nil, err
	}
	var response struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("decode Gemini response envelope: %w", err)
	}
	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 || response.Candidates[0].Content.Parts[0].Text == "" {
		return nil, errors.New("Gemini response did not contain generated text")
	}
	return []byte(response.Candidates[0].Content.Parts[0].Text), nil
}

func (s *Service) openAIChat(ctx context.Context, session Session, system, user string) ([]byte, error) {
	body := map[string]any{
		"model": session.Model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature":     0.2,
		"response_format": map[string]string{"type": "json_object"},
	}
	responseBody, err := s.postJSON(ctx, endpoint(session.BaseURL, "/chat/completions"), bearerHeaders(session.APIKey), body)
	if err != nil {
		return nil, err
	}
	var response struct {
		Choices []struct {
			Message struct {
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("decode chat completion envelope: %w", err)
	}
	if len(response.Choices) == 0 || len(response.Choices[0].Message.Content) == 0 {
		return nil, errors.New("chat completion did not contain message content")
	}
	content := response.Choices[0].Message.Content
	var text string
	if err := json.Unmarshal(content, &text); err == nil {
		return []byte(text), nil
	}
	if json.Valid(content) {
		return content, nil
	}
	return nil, errors.New("chat completion content was neither text nor JSON")
}

func (s *Service) ollamaChat(ctx context.Context, session Session, system, user string, schema map[string]any) ([]byte, error) {
	body := map[string]any{
		"model": session.Model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"stream":  false,
		"format":  schema,
		"options": map[string]any{"temperature": 0.2},
	}
	responseBody, err := s.postJSON(ctx, ollamaEndpoint(session.BaseURL), bearerHeaders(session.APIKey), body)
	if err != nil {
		return nil, err
	}
	var response struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return nil, fmt.Errorf("decode Ollama response envelope: %w", err)
	}
	if response.Message.Content == "" {
		return nil, errors.New("Ollama response did not contain message content")
	}
	return []byte(response.Message.Content), nil
}

func (s *Service) postJSON(ctx context.Context, target string, headers map[string]string, body any) ([]byte, error) {
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode provider request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(encoded))
	if err != nil {
		return nil, errors.New("construct provider request")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Forma/0.1")
	for key, value := range headers {
		if value != "" {
			request.Header.Set(key, value)
		}
	}
	response, err := s.client.Do(request)
	if err != nil {
		return nil, errors.New("provider request failed")
	}
	defer response.Body.Close()
	limited := io.LimitReader(response.Body, (2<<20)+1)
	responseBody, err := io.ReadAll(limited)
	if err != nil {
		return nil, errors.New("read provider response")
	}
	if len(responseBody) > 2<<20 {
		return nil, errors.New("provider response exceeded 2 MiB")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("provider returned HTTP %d", response.StatusCode)
	}
	return responseBody, nil
}

func bearerHeaders(apiKey string) map[string]string {
	if apiKey == "" {
		return nil
	}
	return map[string]string{"Authorization": "Bearer " + apiKey}
}

func endpoint(baseURL, path string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(baseURL, path) {
		return baseURL
	}
	return baseURL + path
}

func ollamaEndpoint(baseURL string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	baseURL = strings.TrimSuffix(baseURL, "/v1")
	return endpoint(baseURL, "/api/chat")
}

func geminiSchema(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		copy := make(map[string]any, len(typed))
		for key, child := range typed {
			if key == "additionalProperties" {
				continue
			}
			copy[key] = geminiSchema(child)
		}
		return copy
	case []string:
		return typed
	case []any:
		copy := make([]any, len(typed))
		for index, child := range typed {
			copy[index] = geminiSchema(child)
		}
		return copy
	default:
		return value
	}
}
