package ai

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSanitizeJSONRemovesPersonalContactData(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"basics":{"name":"Alice Example","email":"alice@example.com","phone":"+1 212 555 0198","location":"New York","photo_url":"data:image/jpeg;base64,private-photo","headline":"Engineer","links":[{"url":"https://example.com"}]},
		"summary":"Alice Example ships systems. Contact alice@example.com.",
		"experience":[{"company":"Named Corp","location":"Remote","highlights":["Reduced latency by 25%"]}]
	}`)
	clean, err := SanitizeJSON(raw)
	if err != nil {
		t.Fatalf("SanitizeJSON() error = %v", err)
	}
	text := string(clean)
	for _, forbidden := range []string{"Alice Example", "alice@example.com", "+1 212 555 0198", "New York", "https://example.com", "private-photo"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("sanitized JSON still contains %q: %s", forbidden, text)
		}
	}
	for _, retained := range []string{"Engineer", "Named Corp", "Remote", "25%"} {
		if !strings.Contains(text, retained) {
			t.Errorf("sanitized JSON lost useful value %q: %s", retained, text)
		}
	}
}

func TestDecodeRewriteRejectsInventedMetric(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"rewritten_text":"Improved throughput by 9000%","explanation":"Clearer","warnings":[]}`)
	if _, err := decodeRewrite(raw, "Improved throughput"); err == nil || !strings.Contains(err.Error(), "unsupported numeric claim") {
		t.Fatalf("decodeRewrite() error = %v, want unsupported metric", err)
	}

	valid := []byte(`{"rewritten_text":"Improved throughput by 25%","explanation":"Clearer","warnings":[]}`)
	if _, err := decodeRewrite(valid, "Improved throughput by 25%"); err != nil {
		t.Fatalf("decodeRewrite(valid) error = %v", err)
	}
}

func TestNormalizeSessionKeepsEditableModel(t *testing.T) {
	t.Parallel()
	session, err := NormalizeSession(Session{
		Provider: "custom", Model: "my-company/model-2026", BaseURL: "https://models.example.test/v1/",
	})
	if err != nil {
		t.Fatalf("NormalizeSession() error = %v", err)
	}
	if session.Model != "my-company/model-2026" || session.BaseURL != "https://models.example.test/v1" {
		t.Fatalf("NormalizeSession() = %#v", session)
	}
	encoded, err := json.Marshal(Session{Provider: "openai", Model: "model", APIKey: "top-secret", ExpiresAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "top-secret") {
		t.Fatalf("Session JSON leaked API key: %s", encoded)
	}
}

func TestBuildRewritePromptBlocksPersonalSections(t *testing.T) {
	t.Parallel()
	_, _, _, err := BuildRewritePrompt(RewriteRequest{Text: "Alice Example", Section: "header"})
	if err == nil {
		t.Fatal("BuildRewritePrompt() unexpectedly allowed header data")
	}
}

func TestBuildReviewPromptRequiresStableSuggestionTargets(t *testing.T) {
	t.Parallel()
	_, user, _, err := BuildReviewPrompt(ReviewRequest{Resume: json.RawMessage(`{"summary":"Builds reliable APIs"}`)})
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"Forma calculates all numeric scores", "semantic.impact_strength", "exact verbatim quote", "Copy original verbatim", "exactly one editable text field", "Never target contact details"} {
		if !strings.Contains(user, required) {
			t.Fatalf("review prompt is missing %q: %s", required, user)
		}
	}
}

func TestDecodeReviewRejectsSuggestionsWithoutAnApplicableTarget(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
	}{
		{name: "blank original", raw: `{"summary":"Review","assessments":` + validAssessmentsJSON + `,"suggestions":[{"id":"one","section":"summary","severity":"medium","title":"Tighten","reason":"Be direct","original":"","replacement":"Builds APIs"}],"warnings":[]}`},
		{name: "unsupported section", raw: `{"summary":"Review","assessments":` + validAssessmentsJSON + `,"suggestions":[{"id":"one","section":"contacts","severity":"medium","title":"Tighten","reason":"Be direct","original":"Old","replacement":"New"}],"warnings":[]}`},
		{name: "model-owned score", raw: `{"summary":"Review","score":80,"assessments":` + validAssessmentsJSON + `,"suggestions":[],"warnings":[]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := decodeReview([]byte(test.raw), "Builds APIs"); err == nil {
				t.Fatalf("decodeReview() accepted %s", test.name)
			}
		})
	}
}

func TestDecodeReviewRequiresFixedSemanticAssessments(t *testing.T) {
	t.Parallel()
	missing := `{"summary":"Review","assessments":[],"suggestions":[],"warnings":[]}`
	if _, err := decodeReview([]byte(missing), "Builds reliable APIs"); err == nil || !strings.Contains(err.Error(), "every fixed rule") {
		t.Fatalf("decodeReview(missing) error = %v", err)
	}

	duplicate := `{"summary":"Review","assessments":[
		{"rule_id":"semantic.impact_strength","verdict":"pass","evidence":"Builds reliable APIs","confidence":0.9,"reason":"Clear"},
		{"rule_id":"semantic.impact_strength","verdict":"pass","evidence":"Builds reliable APIs","confidence":0.9,"reason":"Clear"},
		{"rule_id":"semantic.target_relevance","verdict":"not_applicable","evidence":"","confidence":0.9,"reason":"No target"},
		{"rule_id":"semantic.voice_coherence","verdict":"pass","evidence":"Builds reliable APIs","confidence":0.9,"reason":"Clear"}
	],"suggestions":[],"warnings":[]}`
	if _, err := decodeReview([]byte(duplicate), "Builds reliable APIs"); err == nil || !strings.Contains(err.Error(), "duplicates") {
		t.Fatalf("decodeReview(duplicate) error = %v", err)
	}
}
