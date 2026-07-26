package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/quality"
)

func TestEvaluateResumeQualityDoesNotRequireAISession(t *testing.T) {
	t.Parallel()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/quality/evaluate", strings.NewReader(`{
		"resume":{
			"version":1,
			"basics":{"name":"Alex Morgan","headline":"Product Engineer","email":"alex@example.com"},
			"summary":"Product engineer building reliable products with clear ownership and maintainable delivery practices across customer-facing teams and internal platforms.",
			"experience":[{"id":"exp-1","company":"Forma","position":"Engineer","start_date":"2022-01","current":true,"highlights":["Built reliable product services that improved support team response."]}],
			"language":"en"
		}
	}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	(&API{}).evaluateResumeQuality(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var envelope struct {
		Data quality.Evaluation `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.RubricVersion != quality.RubricVersion || envelope.Data.Semantic.UnassessedPoints != quality.SemanticMaximum {
		t.Fatalf("evaluation = %#v", envelope.Data)
	}
}

func TestEvaluateResumeQualityRejectsUnknownRequestFields(t *testing.T) {
	t.Parallel()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/quality/evaluate", strings.NewReader(`{"resume":{},"model_score":99}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	(&API{}).evaluateResumeQuality(response, request)
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "invalid_json") {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}
