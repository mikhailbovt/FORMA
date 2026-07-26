package quality

import (
	"encoding/json"
	"math"
	"strings"
)

// ApplySemantic is the safe default for a review without target context.
func ApplySemantic(base Evaluation, assessments []SemanticAssessment, sanitizedSource string) Evaluation {
	return ApplySemanticWithContext(base, assessments, sanitizedSource, SemanticContext{})
}

// ApplySemanticWithContext accepts only the fixed semantic rubric. It never
// trusts model supplied weights, rule IDs, evidence, or low-confidence verdicts.
func ApplySemanticWithContext(base Evaluation, assessments []SemanticAssessment, sanitizedSource string, context SemanticContext) Evaluation {
	result := base
	result.Findings = append([]Finding(nil), base.Findings...)
	result.Semantic.Criteria = append([]SemanticCriterion(nil), base.Semantic.Criteria...)

	byRule := make(map[string][]SemanticAssessment, len(semanticDefinitions))
	ignored := 0
	for _, assessment := range assessments {
		assessment.RuleID = strings.TrimSpace(assessment.RuleID)
		if _, ok := semanticDefinitionFor(assessment.RuleID); !ok {
			ignored++
			continue
		}
		byRule[assessment.RuleID] = append(byRule[assessment.RuleID], assessment)
	}

	semanticFindings := make([]Finding, 0, len(semanticDefinitions))
	criteria := make([]SemanticCriterion, 0, len(semanticDefinitions))
	assessedPoints, earnedPoints := 0, 0
	for _, definition := range semanticDefinitions {
		criterion := SemanticCriterion{
			RuleID: definition.ruleID, Label: definition.label, MaximumPoints: definition.weight, Status: StatusUnassessed,
		}
		finding := Finding{
			RuleID: definition.ruleID, Category: "semantic", Status: StatusUnassessed, Severity: "info",
			Message: "No valid semantic assessment was accepted.", PossiblePoints: definition.weight,
		}
		candidates := byRule[definition.ruleID]
		if definition.ruleID == "semantic.target_relevance" && strings.TrimSpace(context.TargetRole) == "" && strings.TrimSpace(context.JobDescription) == "" {
			criterion.Status = StatusNotApplicable
			finding.Status = StatusNotApplicable
			finding.Message = "Target relevance is not applicable because no target role or job description was supplied."
			if len(candidates) == 1 && strings.ToLower(strings.TrimSpace(candidates[0].Verdict)) != "not_applicable" {
				ignored++
			}
			semanticFindings = append(semanticFindings, finding)
			criteria = append(criteria, criterion)
			continue
		}
		if len(candidates) != 1 {
			if len(candidates) > 1 {
				ignored += len(candidates)
				finding.Message = "Duplicate assessments were rejected for this fixed semantic rule."
			}
			semanticFindings = append(semanticFindings, finding)
			criteria = append(criteria, criterion)
			continue
		}

		assessment := candidates[0]
		assessment.Verdict = strings.ToLower(strings.TrimSpace(assessment.Verdict))
		assessment.Evidence = strings.TrimSpace(assessment.Evidence)
		assessment.Reason = strings.TrimSpace(assessment.Reason)
		if math.IsNaN(assessment.Confidence) || math.IsInf(assessment.Confidence, 0) || assessment.Confidence < ConfidenceThreshold || assessment.Confidence > 1 || assessment.Reason == "" {
			ignored++
			finding.Message = "The semantic assessment was left unassessed because confidence or rationale did not meet the fixed contract."
			semanticFindings = append(semanticFindings, finding)
			criteria = append(criteria, criterion)
			continue
		}
		if assessment.Verdict == "not_applicable" {
			criterion.Status = StatusNotApplicable
			finding.Status = StatusNotApplicable
			finding.Message = assessment.Reason
			semanticFindings = append(semanticFindings, finding)
			criteria = append(criteria, criterion)
			continue
		}
		if assessment.Verdict != "pass" && assessment.Verdict != "partial" && assessment.Verdict != "fail" {
			ignored++
			finding.Message = "The semantic assessment used an unsupported verdict."
			semanticFindings = append(semanticFindings, finding)
			criteria = append(criteria, criterion)
			continue
		}
		if assessment.Evidence == "" || !evidenceExists(sanitizedSource, assessment.Evidence) {
			ignored++
			finding.Message = "The semantic assessment was rejected because its exact evidence quote was not found in the sanitized resume."
			semanticFindings = append(semanticFindings, finding)
			criteria = append(criteria, criterion)
			continue
		}

		assessedPoints += definition.weight
		finding.Message = assessment.Reason
		finding.Evidence = []Evidence{{Path: "$", Actual: assessment.Evidence, Expected: "exact quote from sanitized resume"}}
		switch assessment.Verdict {
		case "pass":
			criterion.Status = StatusPass
			finding.Status = StatusPass
			finding.Severity = "low"
			finding.EarnedPoints = definition.weight
		case "partial":
			criterion.Status = StatusWarn
			finding.Status = StatusWarn
			finding.Severity = "medium"
			finding.EarnedPoints = definition.weight / 2
		case "fail":
			criterion.Status = StatusFail
			finding.Status = StatusFail
			finding.Severity = "high"
		}
		earnedPoints += finding.EarnedPoints
		semanticFindings = append(semanticFindings, finding)
		criteria = append(criteria, criterion)
	}

	deterministicFindings := make([]Finding, 0, len(result.Findings))
	for _, finding := range result.Findings {
		if finding.Category != "semantic" {
			deterministicFindings = append(deterministicFindings, finding)
		}
	}
	result.Findings = append(deterministicFindings, semanticFindings...)
	result.Semantic = SemanticSummary{
		MaximumPoints: SemanticMaximum, AssessedPoints: assessedPoints, EarnedPoints: earnedPoints,
		UnassessedPoints: SemanticMaximum - assessedPoints, IgnoredCount: ignored, Criteria: criteria,
	}
	recalculate(&result)
	return result
}

func evidenceExists(source, quote string) bool {
	if strings.TrimSpace(source) == "" || strings.TrimSpace(quote) == "" {
		return false
	}
	var value any
	if err := json.Unmarshal([]byte(source), &value); err != nil {
		return false
	}
	values := make([]string, 0)
	collectStrings(value, &values)
	for _, value := range values {
		if strings.Contains(value, quote) {
			return true
		}
	}
	return false
}
