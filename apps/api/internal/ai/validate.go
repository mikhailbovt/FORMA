package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/quality"
)

var ErrInvalidOutput = errors.New("provider returned invalid structured output")

func decodeReview(raw []byte, source string) (ReviewResult, error) {
	var result ReviewResult
	if err := decodeStrict(raw, &result); err != nil {
		return ReviewResult{}, fmt.Errorf("%w: %v", ErrInvalidOutput, err)
	}
	if strings.TrimSpace(result.Summary) == "" || len([]rune(result.Summary)) > 2_000 {
		return ReviewResult{}, fmt.Errorf("%w: summary is empty or too long", ErrInvalidOutput)
	}
	semanticIDs := quality.SemanticRuleIDs()
	if len(result.Assessments) != len(semanticIDs) {
		return ReviewResult{}, fmt.Errorf("%w: semantic assessments must contain every fixed rule", ErrInvalidOutput)
	}
	allowedSemanticIDs := make(map[string]struct{}, len(semanticIDs))
	for _, id := range semanticIDs {
		allowedSemanticIDs[id] = struct{}{}
	}
	seenSemanticIDs := make(map[string]struct{}, len(semanticIDs))
	for index, assessment := range result.Assessments {
		if _, ok := allowedSemanticIDs[assessment.RuleID]; !ok {
			return ReviewResult{}, fmt.Errorf("%w: semantic assessment %d has an unsupported rule ID", ErrInvalidOutput, index)
		}
		if _, duplicate := seenSemanticIDs[assessment.RuleID]; duplicate {
			return ReviewResult{}, fmt.Errorf("%w: semantic assessment %d duplicates a rule ID", ErrInvalidOutput, index)
		}
		seenSemanticIDs[assessment.RuleID] = struct{}{}
		if !slices.Contains([]string{"pass", "partial", "fail", "not_applicable"}, assessment.Verdict) {
			return ReviewResult{}, fmt.Errorf("%w: semantic assessment %d has an unsupported verdict", ErrInvalidOutput, index)
		}
		if assessment.Confidence < 0 || assessment.Confidence > 1 {
			return ReviewResult{}, fmt.Errorf("%w: semantic assessment %d has invalid confidence", ErrInvalidOutput, index)
		}
		if strings.TrimSpace(assessment.Reason) == "" || len([]rune(assessment.Reason)) > 2_000 {
			return ReviewResult{}, fmt.Errorf("%w: semantic assessment %d has an invalid reason", ErrInvalidOutput, index)
		}
		if assessment.Verdict != "not_applicable" && strings.TrimSpace(assessment.Evidence) == "" {
			return ReviewResult{}, fmt.Errorf("%w: semantic assessment %d requires exact evidence", ErrInvalidOutput, index)
		}
		if len([]rune(assessment.Evidence)) > 2_000 {
			return ReviewResult{}, fmt.Errorf("%w: semantic assessment %d evidence is too long", ErrInvalidOutput, index)
		}
	}
	if len(result.Suggestions) > 25 {
		return ReviewResult{}, fmt.Errorf("%w: too many suggestions", ErrInvalidOutput)
	}
	for index, suggestion := range result.Suggestions {
		if suggestion.ID == "" || suggestion.Section == "" || suggestion.Title == "" || suggestion.Reason == "" {
			return ReviewResult{}, fmt.Errorf("%w: suggestion %d is incomplete", ErrInvalidOutput, index)
		}
		if !slices.Contains([]string{"basics", "summary", "experience", "projects", "portfolio", "education", "skills", "certifications", "languages"}, strings.ToLower(strings.TrimSpace(suggestion.Section))) {
			return ReviewResult{}, fmt.Errorf("%w: suggestion %d targets an unsupported section", ErrInvalidOutput, index)
		}
		if strings.TrimSpace(suggestion.Original) == "" || strings.TrimSpace(suggestion.Replacement) == "" {
			return ReviewResult{}, fmt.Errorf("%w: suggestion %d must contain an exact original and replacement", ErrInvalidOutput, index)
		}
		if suggestion.Severity != "low" && suggestion.Severity != "medium" && suggestion.Severity != "high" {
			return ReviewResult{}, fmt.Errorf("%w: suggestion %d has invalid severity", ErrInvalidOutput, index)
		}
		if len([]rune(suggestion.Replacement)) > 5_000 {
			return ReviewResult{}, fmt.Errorf("%w: suggestion %d replacement is too long", ErrInvalidOutput, index)
		}
		if err := validateNoNewNumbers(source, suggestion.Replacement); err != nil {
			return ReviewResult{}, fmt.Errorf("%w: suggestion %d %v", ErrInvalidOutput, index, err)
		}
	}
	return result, nil
}

func decodeRewrite(raw []byte, source string) (RewriteResult, error) {
	var result RewriteResult
	if err := decodeStrict(raw, &result); err != nil {
		return RewriteResult{}, fmt.Errorf("%w: %v", ErrInvalidOutput, err)
	}
	if strings.TrimSpace(result.RewrittenText) == "" || len([]rune(result.RewrittenText)) > 20_000 {
		return RewriteResult{}, fmt.Errorf("%w: rewritten_text is empty or too long", ErrInvalidOutput)
	}
	if err := validateNoNewNumbers(source, result.RewrittenText); err != nil {
		return RewriteResult{}, fmt.Errorf("%w: %v", ErrInvalidOutput, err)
	}
	return result, nil
}

func decodeStrict(raw []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func validateNoNewNumbers(source, output string) error {
	allowed := make(map[string]struct{})
	for _, match := range metricPattern.FindAllString(strings.ToLower(source), -1) {
		allowed[normalizeNumber(match)] = struct{}{}
	}
	for _, match := range metricPattern.FindAllString(strings.ToLower(output), -1) {
		if _, ok := allowed[normalizeNumber(match)]; !ok {
			return fmt.Errorf("introduces unsupported numeric claim %q", match)
		}
	}
	return nil
}

func normalizeNumber(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(strings.ToLower(value)), ",", "")
}
