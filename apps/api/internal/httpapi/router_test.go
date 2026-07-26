package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/ai"
	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/resume"
	"github.com/google/uuid"
)

type healthyDB struct{ err error }

func (db healthyDB) Ping(context.Context) error { return db.err }

type fakeResumeStore struct {
	created resume.Input
	item    resume.Resume
}

type fakeAIService struct {
	review ai.ReviewResult
}

func (service fakeAIService) Review(context.Context, ai.Session, ai.ReviewRequest) (ai.ReviewResult, error) {
	return service.review, nil
}

func (fakeAIService) Rewrite(context.Context, ai.Session, ai.RewriteRequest) (ai.RewriteResult, error) {
	return ai.RewriteResult{}, nil
}

func (store *fakeResumeStore) List(context.Context, int, int) ([]resume.Resume, error) {
	return []resume.Resume{store.item}, nil
}
func (store *fakeResumeStore) Create(_ context.Context, input resume.Input) (resume.Resume, error) {
	store.created = input
	store.item = resume.Resume{ID: uuid.New(), Title: input.Title, Document: input.Document, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	return store.item, nil
}
func (store *fakeResumeStore) Get(context.Context, uuid.UUID) (resume.Resume, error) {
	return store.item, nil
}
func (store *fakeResumeStore) Update(_ context.Context, id uuid.UUID, input resume.Input) (resume.Resume, error) {
	store.item.ID, store.item.Title, store.item.Document = id, input.Title, input.Document
	return store.item, nil
}
func (store *fakeResumeStore) Delete(context.Context, uuid.UUID) error { return nil }
func (store *fakeResumeStore) Duplicate(context.Context, uuid.UUID) (resume.Resume, error) {
	copy := store.item
	copy.ID = uuid.New()
	return copy, nil
}

func testRouter(t *testing.T, store resume.Store) (http.Handler, *ai.SessionStore) {
	t.Helper()
	sessions := ai.NewSessionStore(time.Minute)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(Dependencies{
		Logger: logger, DB: healthyDB{}, Resumes: store, Sessions: sessions,
		CORSOrigin: "http://localhost:3000", MaxBodyBytes: 2 << 20,
	}), sessions
}

func TestResumeCreateNormalizesDocument(t *testing.T) {
	store := &fakeResumeStore{}
	router, sessions := testRouter(t, store)
	defer sessions.Close()

	request := httptest.NewRequest(http.MethodPost, "/api/v1/resumes", strings.NewReader(`{
		"title":"Backend resume",
		"document":{"basics":{"headline":"Go Engineer"},"experience":[],"projects":[],"education":[],"skills":[],"certifications":[],"languages":[],"custom_sections":[]}
	}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if store.created.Document.Version != 1 || store.created.Document.Template != "forma" || store.created.Document.PageSize != "A4" || store.created.Document.Language != "en" {
		t.Fatalf("document defaults = %#v", store.created.Document)
	}
	if response.Header().Get("Location") == "" {
		t.Fatal("missing Location header")
	}
}

func TestAISessionCookieDoesNotExposeKey(t *testing.T) {
	router, sessions := testRouter(t, &fakeResumeStore{})
	defer sessions.Close()

	put := httptest.NewRequest(http.MethodPut, "/api/v1/ai/session", strings.NewReader(`{
		"provider":"custom","model":"private-model","api_key":"super-secret","base_url":"https://models.example.test/v1"
	}`))
	put.Header.Set("Content-Type", "application/json")
	putResponse := httptest.NewRecorder()
	router.ServeHTTP(putResponse, put)
	if putResponse.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, body = %s", putResponse.Code, putResponse.Body.String())
	}
	if strings.Contains(putResponse.Body.String(), "super-secret") {
		t.Fatalf("PUT response leaked key: %s", putResponse.Body.String())
	}
	cookies := putResponse.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("session cookie = %#v", cookies)
	}

	get := httptest.NewRequest(http.MethodGet, "/api/v1/ai/session", nil)
	get.AddCookie(cookies[0])
	getResponse := httptest.NewRecorder()
	router.ServeHTTP(getResponse, get)
	if getResponse.Code != http.StatusOK || !strings.Contains(getResponse.Body.String(), `"has_api_key":true`) || strings.Contains(getResponse.Body.String(), "super-secret") {
		t.Fatalf("GET status = %d, body = %s", getResponse.Code, getResponse.Body.String())
	}
}

func TestAISessionCanKeepExistingKey(t *testing.T) {
	router, sessions := testRouter(t, &fakeResumeStore{})
	defer sessions.Close()

	first := httptest.NewRequest(http.MethodPut, "/api/v1/ai/session", strings.NewReader(`{"provider":"openai","model":"gpt-5.6-terra","api_key":"super-secret"}`))
	first.Header.Set("Content-Type", "application/json")
	firstResponse := httptest.NewRecorder()
	router.ServeHTTP(firstResponse, first)
	if firstResponse.Code != http.StatusOK {
		t.Fatalf("first PUT status = %d, body = %s", firstResponse.Code, firstResponse.Body.String())
	}

	second := httptest.NewRequest(http.MethodPut, "/api/v1/ai/session", strings.NewReader(`{"provider":"openai","model":"gpt-5.6-terra"}`))
	second.Header.Set("Content-Type", "application/json")
	second.AddCookie(firstResponse.Result().Cookies()[0])
	secondResponse := httptest.NewRecorder()
	router.ServeHTTP(secondResponse, second)
	if secondResponse.Code != http.StatusOK || !strings.Contains(secondResponse.Body.String(), `"has_api_key":true`) {
		t.Fatalf("second PUT status = %d, body = %s", secondResponse.Code, secondResponse.Body.String())
	}
}

func TestAIReviewRequiresExplicitSession(t *testing.T) {
	router, sessions := testRouter(t, &fakeResumeStore{})
	defer sessions.Close()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/ai/review", strings.NewReader(`{"resume":{"summary":"Built APIs"}}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "ai_not_configured") {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

func TestAIReviewReturnsServerScoredQualityAndSeparateSuggestions(t *testing.T) {
	store := &fakeResumeStore{}
	sessions := ai.NewSessionStore(time.Minute)
	defer sessions.Close()
	evidence := "Built reliable APIs for finance teams."
	service := fakeAIService{review: ai.ReviewResult{
		Summary: "Strong foundation.",
		Assessments: []ai.Assessment{
			{RuleID: "semantic.impact_strength", Verdict: "pass", Evidence: evidence, Confidence: 0.95, Reason: "Shows contribution and audience."},
			{RuleID: "semantic.clarity_specificity", Verdict: "pass", Evidence: evidence, Confidence: 0.95, Reason: "Uses direct and specific language."},
			{RuleID: "semantic.target_relevance", Verdict: "pass", Evidence: evidence, Confidence: 0.95, Reason: "Relevant to the supplied target role."},
			{RuleID: "semantic.voice_coherence", Verdict: "pass", Evidence: evidence, Confidence: 0.95, Reason: "Uses a consistent professional voice."},
		},
	}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := New(Dependencies{
		Logger: logger, DB: healthyDB{}, Resumes: store, Sessions: sessions, AI: service,
		CORSOrigin: "http://localhost:3000", MaxBodyBytes: 2 << 20,
	})

	configure := httptest.NewRequest(http.MethodPut, "/api/v1/ai/session", strings.NewReader(`{"provider":"custom","model":"test-model","base_url":"https://models.example.test/v1"}`))
	configure.Header.Set("Content-Type", "application/json")
	configureResponse := httptest.NewRecorder()
	router.ServeHTTP(configureResponse, configure)
	if configureResponse.Code != http.StatusOK {
		t.Fatalf("configure status = %d, body = %s", configureResponse.Code, configureResponse.Body.String())
	}

	review := httptest.NewRequest(http.MethodPost, "/api/v1/ai/review", strings.NewReader(`{
		"resume":{"basics":{"name":"Alex Morgan","headline":"Backend Engineer"},"summary":"Built reliable APIs for finance teams."},
		"target_role":"Backend Engineer"
	}`))
	review.Header.Set("Content-Type", "application/json")
	review.AddCookie(configureResponse.Result().Cookies()[0])
	reviewResponse := httptest.NewRecorder()
	router.ServeHTTP(reviewResponse, review)
	if reviewResponse.Code != http.StatusOK {
		t.Fatalf("review status = %d, body = %s", reviewResponse.Code, reviewResponse.Body.String())
	}

	var body struct {
		Data struct {
			Quality map[string]any `json:"quality"`
			AI      map[string]any `json:"ai"`
		} `json:"data"`
	}
	if err := json.Unmarshal(reviewResponse.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Data.Quality["rubric_version"] != "forma-quality/1.0.0" {
		t.Fatalf("quality envelope = %#v", body.Data.Quality)
	}
	if _, modelScore := body.Data.AI["score"]; modelScore {
		t.Fatalf("AI output must not own a score: %#v", body.Data.AI)
	}
	if body.Data.AI["summary"] != "Strong foundation." {
		t.Fatalf("AI envelope = %#v", body.Data.AI)
	}
}

func TestHealthAndCORS(t *testing.T) {
	router, sessions := testRouter(t, &fakeResumeStore{})
	defer sessions.Close()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("status = %d, origin = %q, body = %s", response.Code, response.Header().Get("Access-Control-Allow-Origin"), response.Body.String())
	}
}
