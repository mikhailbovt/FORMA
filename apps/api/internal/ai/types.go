package ai

import (
	"encoding/json"

	"github.com/mikhailbovt/FORMA/apps/api/internal/quality"
)

type ReviewRequest struct {
	Resume         json.RawMessage `json:"resume"`
	TargetRole     string          `json:"target_role,omitempty"`
	JobDescription string          `json:"job_description,omitempty"`
	Focus          string          `json:"focus,omitempty"`
}

type ReviewResult struct {
	Summary     string                       `json:"summary"`
	Assessments []quality.SemanticAssessment `json:"assessments"`
	Suggestions []Suggestion                 `json:"suggestions"`
	Warnings    []string                     `json:"warnings"`
}

type Assessment = quality.SemanticAssessment

type Suggestion struct {
	ID          string `json:"id"`
	Section     string `json:"section"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Reason      string `json:"reason"`
	Original    string `json:"original"`
	Replacement string `json:"replacement"`
}

type RewriteRequest struct {
	Text        string          `json:"text"`
	Section     string          `json:"section,omitempty"`
	Instruction string          `json:"instruction,omitempty"`
	Context     json.RawMessage `json:"context,omitempty"`
	TargetRole  string          `json:"target_role,omitempty"`
}

type RewriteResult struct {
	RewrittenText string   `json:"rewritten_text"`
	Explanation   string   `json:"explanation"`
	Warnings      []string `json:"warnings"`
}
