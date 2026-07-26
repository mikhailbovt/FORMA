package quality

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/resume"
)

func TestGoldenEvaluations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		file            string
		language        string
		score           int
		assessed        int
		unassessed      int
		normalized      int
		ready           bool
		blockers        []string
		semanticPending int
	}{
		{file: "en_strong.json", language: "en", score: 60, assessed: 60, unassessed: 40, normalized: 100, ready: true, blockers: []string{}, semanticPending: 40},
		{file: "ru_strong.json", language: "ru", score: 58, assessed: 58, unassessed: 42, normalized: 100, ready: true, blockers: []string{}, semanticPending: 40},
		{file: "unknown_language.json", language: "de", score: 47, assessed: 48, unassessed: 52, normalized: 98, ready: true, blockers: []string{}, semanticPending: 40},
		{file: "empty_invalid.json", language: "en", score: 0, assessed: 18, unassessed: 82, normalized: 0, ready: false,
			blockers: []string{"essentials.name.present", "essentials.contact.valid", "essentials.content.substantive", "structure.placeholders.absent"}, semanticPending: 40},
	}
	for _, test := range tests {
		test := test
		t.Run(test.file, func(t *testing.T) {
			t.Parallel()
			document := loadFixture(t, test.file)
			got, err := Evaluate(document)
			if err != nil {
				t.Fatal(err)
			}
			if got.RubricVersion != RubricVersion || !strings.HasPrefix(got.SourceDigest, "sha256:") || len(got.SourceDigest) != 71 {
				t.Fatalf("metadata = version %q digest %q", got.RubricVersion, got.SourceDigest)
			}
			if got.Language != test.language || got.Quality.Score != test.score || got.Quality.AssessedPoints != test.assessed ||
				got.Quality.UnassessedPoints != test.unassessed || got.Quality.NormalizedScore != test.normalized || got.Quality.Ready != test.ready {
				t.Fatalf("evaluation = language=%s quality=%#v", got.Language, got.Quality)
			}
			if !reflect.DeepEqual(got.Quality.Blockers, test.blockers) {
				t.Fatalf("blockers = %#v, want %#v", got.Quality.Blockers, test.blockers)
			}
			if got.Semantic.MaximumPoints != SemanticMaximum || got.Semantic.AssessedPoints != 0 || got.Semantic.UnassessedPoints != test.semanticPending {
				t.Fatalf("semantic budget = %#v", got.Semantic)
			}
		})
	}
}

func TestEvaluationIsStableAndRuleOrderIsVersioned(t *testing.T) {
	t.Parallel()
	document := loadFixture(t, "en_strong.json")
	first, err := Evaluate(document)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Evaluate(document)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("same input produced different evaluations:\nfirst=%#v\nsecond=%#v", first, second)
	}
	wantRules := []string{
		"essentials.name.present", "essentials.contact.valid", "essentials.headline.focused", "essentials.content.substantive",
		"structure.entries.complete", "structure.evidence.coverage", "structure.placeholders.absent",
		"evidence.lines.substantive", "evidence.action_led", "evidence.result_signals", "evidence.duplicates.absent",
		"clarity.summary.concise", "clarity.evidence.concise", "clarity.first_person.limited", "clarity.filler.limited", "clarity.punctuation.consistent",
		"consistency.dates.parseable", "consistency.periods.valid", "consistency.reverse_chronological",
		"semantic.impact_strength", "semantic.clarity_specificity", "semantic.target_relevance", "semantic.voice_coherence",
	}
	gotRules := make([]string, 0, len(first.Findings))
	for _, finding := range first.Findings {
		gotRules = append(gotRules, finding.RuleID)
	}
	if !reflect.DeepEqual(gotRules, wantRules) {
		t.Fatalf("rule order = %#v, want %#v", gotRules, wantRules)
	}
}

func TestUnknownLanguageRulesAreNotApplicable(t *testing.T) {
	t.Parallel()
	evaluation, err := Evaluate(loadFixture(t, "unknown_language.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, ruleID := range []string{"evidence.action_led", "evidence.result_signals", "clarity.first_person.limited", "clarity.filler.limited"} {
		if finding := findingByID(t, evaluation, ruleID); finding.Status != StatusNotApplicable || finding.EarnedPoints != 0 {
			t.Fatalf("%s = %#v, want not_applicable with no points", ruleID, finding)
		}
	}
}

func TestPhotoPageSizeEducationAndNumbersHaveNoUniversalPenalty(t *testing.T) {
	t.Parallel()
	base := loadFixture(t, "en_strong.json")
	baseline, err := Evaluate(base)
	if err != nil {
		t.Fatal(err)
	}
	variant := base
	variant.Basics.PhotoURL = "data:image/jpeg;base64,cGhvdG8="
	variant.PageSize = "LETTER"
	variant.Education = []resume.Education{{ID: "edu-1", Institution: "Example University", StudyType: "BSc", Area: "Computer Science"}}
	variant.Experience[0].Highlights[0] = strings.TrimSuffix(variant.Experience[0].Highlights[0], ".") + " for 35 teams."
	changed, err := Evaluate(variant)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Quality.Score != baseline.Quality.Score || changed.Quality.AssessedPoints != baseline.Quality.AssessedPoints || changed.Quality.Ready != baseline.Quality.Ready {
		t.Fatalf("neutral fields changed quality: baseline=%#v variant=%#v", baseline.Quality, changed.Quality)
	}
	if changed.SourceDigest == baseline.SourceDigest {
		t.Fatal("source digest did not reflect the changed source")
	}
}

func TestApplySemanticUsesOnlyFixedWeightsAndExactEvidence(t *testing.T) {
	t.Parallel()
	document := loadFixture(t, "en_strong.json")
	base, err := Evaluate(document)
	if err != nil {
		t.Fatal(err)
	}
	sourceBytes, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	evidence := document.Experience[0].Highlights[0]
	assessments := passingAssessments(evidence)
	context := SemanticContext{TargetRole: "Senior Product Engineer"}
	full := ApplySemanticWithContext(base, assessments, string(sourceBytes), context)
	if full.Quality.Score != 100 || full.Quality.AssessedPoints != 100 || full.Quality.UnassessedPoints != 0 || full.Semantic.EarnedPoints != 40 {
		t.Fatalf("full semantic result = quality=%#v semantic=%#v", full.Quality, full.Semantic)
	}

	withUnknown := append(append([]SemanticAssessment(nil), assessments...), SemanticAssessment{
		RuleID: "semantic.model_chosen_bonus", Verdict: "pass", Evidence: evidence, Confidence: 1, Reason: "Invented bonus",
	})
	unknownResult := ApplySemanticWithContext(base, withUnknown, string(sourceBytes), context)
	if unknownResult.Quality.Score != 100 || unknownResult.Semantic.IgnoredCount != 1 {
		t.Fatalf("unknown rule affected result: quality=%#v semantic=%#v", unknownResult.Quality, unknownResult.Semantic)
	}

	badEvidence := append([]SemanticAssessment(nil), assessments...)
	badEvidence[0].Evidence = "A fabricated quote that is absent"
	badEvidenceResult := ApplySemanticWithContext(base, badEvidence, string(sourceBytes), context)
	if badEvidenceResult.Quality.Score != 88 || badEvidenceResult.Quality.AssessedPoints != 88 || badEvidenceResult.Semantic.IgnoredCount != 1 {
		t.Fatalf("bad evidence affected result: quality=%#v semantic=%#v", badEvidenceResult.Quality, badEvidenceResult.Semantic)
	}

	lowConfidence := append([]SemanticAssessment(nil), assessments...)
	lowConfidence[1].Confidence = ConfidenceThreshold - 0.01
	lowConfidenceResult := ApplySemanticWithContext(base, lowConfidence, string(sourceBytes), context)
	if lowConfidenceResult.Quality.Score != 90 || lowConfidenceResult.Quality.AssessedPoints != 90 || lowConfidenceResult.Semantic.IgnoredCount != 1 {
		t.Fatalf("low-confidence assessment affected result: quality=%#v semantic=%#v", lowConfidenceResult.Quality, lowConfidenceResult.Semantic)
	}
}

func TestApplySemanticMapsVerdictsWithoutModelWeights(t *testing.T) {
	t.Parallel()
	document := loadFixture(t, "en_strong.json")
	base, err := Evaluate(document)
	if err != nil {
		t.Fatal(err)
	}
	sourceBytes, _ := json.Marshal(document)
	evidence := document.Experience[0].Highlights[0]
	assessments := passingAssessments(evidence)
	assessments[0].Verdict = "partial" // 6 of the fixed 12 points.
	assessments[2].Verdict = "fail"    // 0 of the fixed 10 points.
	assessments[3].Verdict = "not_applicable"
	assessments[3].Evidence = ""
	result := ApplySemanticWithContext(base, assessments, string(sourceBytes), SemanticContext{JobDescription: "Seeking a senior product engineer."})
	if result.Semantic.AssessedPoints != 32 || result.Semantic.EarnedPoints != 16 || result.Semantic.UnassessedPoints != 8 {
		t.Fatalf("semantic mapping = %#v", result.Semantic)
	}
	if result.Quality.Score != 76 || result.Quality.AssessedPoints != 92 || result.Quality.UnassessedPoints != 8 {
		t.Fatalf("combined quality = %#v", result.Quality)
	}
}

func TestTargetRelevanceCannotScoreWithoutTargetContext(t *testing.T) {
	t.Parallel()
	document := loadFixture(t, "en_strong.json")
	base, err := Evaluate(document)
	if err != nil {
		t.Fatal(err)
	}
	sourceBytes, _ := json.Marshal(document)
	evidence := document.Experience[0].Highlights[0]
	result := ApplySemantic(base, passingAssessments(evidence), string(sourceBytes))
	if result.Quality.Score != 90 || result.Quality.AssessedPoints != 90 || result.Quality.UnassessedPoints != 10 {
		t.Fatalf("target relevance earned points without target context: %#v", result.Quality)
	}
	target := findingByID(t, result, "semantic.target_relevance")
	if target.Status != StatusNotApplicable || target.EarnedPoints != 0 || result.Semantic.IgnoredCount != 1 {
		t.Fatalf("target relevance = %#v semantic=%#v", target, result.Semantic)
	}
}

func TestInvalidDatesAndPlaceholdersAreReadyBlockers(t *testing.T) {
	t.Parallel()
	document := loadFixture(t, "en_strong.json")
	document.Experience[0].StartDate = "2022-99"
	document.Skills[0].Name = "Skill group"
	evaluation, err := Evaluate(document)
	if err != nil {
		t.Fatal(err)
	}
	if evaluation.Quality.Ready {
		t.Fatal("evaluation remained ready with an invalid date and a template placeholder")
	}
	for _, ruleID := range []string{"structure.placeholders.absent", "consistency.dates.parseable"} {
		if !containsString(evaluation.Quality.Blockers, ruleID) {
			t.Fatalf("blockers = %#v, missing %q", evaluation.Quality.Blockers, ruleID)
		}
	}
}

func loadFixture(t *testing.T, name string) resume.Document {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	var document resume.Document
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func findingByID(t *testing.T, evaluation Evaluation, ruleID string) Finding {
	t.Helper()
	for _, finding := range evaluation.Findings {
		if finding.RuleID == ruleID {
			return finding
		}
	}
	t.Fatalf("finding %q not found", ruleID)
	return Finding{}
}

func passingAssessments(evidence string) []SemanticAssessment {
	return []SemanticAssessment{
		{RuleID: "semantic.impact_strength", Verdict: "pass", Evidence: evidence, Confidence: 0.95, Reason: "Impact and ownership are clear."},
		{RuleID: "semantic.clarity_specificity", Verdict: "pass", Evidence: evidence, Confidence: 0.95, Reason: "The claim is clear and specific."},
		{RuleID: "semantic.target_relevance", Verdict: "pass", Evidence: evidence, Confidence: 0.95, Reason: "The evidence is relevant to the target."},
		{RuleID: "semantic.voice_coherence", Verdict: "pass", Evidence: evidence, Confidence: 0.95, Reason: "The voice is coherent."},
	}
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
