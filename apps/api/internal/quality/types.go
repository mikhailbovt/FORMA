package quality

const (
	RubricVersion        = "forma-quality/1.0.0"
	DeterministicMaximum = 60
	SemanticMaximum      = 40
	ConfidenceThreshold  = 0.75
)

type Status string

const (
	StatusPass          Status = "pass"
	StatusWarn          Status = "warn"
	StatusFail          Status = "fail"
	StatusNotApplicable Status = "not_applicable"
	StatusUnassessed    Status = "unassessed"
)

type Evaluation struct {
	RubricVersion string          `json:"rubric_version"`
	SourceDigest  string          `json:"source_digest"`
	Language      string          `json:"language"`
	Quality       Scorecard       `json:"quality"`
	ATSHygiene    ATSAssessment   `json:"ats_hygiene"`
	Semantic      SemanticSummary `json:"semantic"`
	Findings      []Finding       `json:"findings"`
}

type Scorecard struct {
	Score            int        `json:"score"`
	MaximumScore     int        `json:"maximum_score"`
	AssessedPoints   int        `json:"assessed_points"`
	UnassessedPoints int        `json:"unassessed_points"`
	NormalizedScore  int        `json:"normalized_score"`
	Ready            bool       `json:"ready"`
	Blockers         []string   `json:"blockers"`
	Categories       []Category `json:"categories"`
}

type Category struct {
	ID               string `json:"id"`
	Label            string `json:"label"`
	MaximumPoints    int    `json:"maximum_points"`
	AssessedPoints   int    `json:"assessed_points"`
	EarnedPoints     int    `json:"earned_points"`
	UnassessedPoints int    `json:"unassessed_points"`
	Status           Status `json:"status"`
}

type Finding struct {
	RuleID         string     `json:"rule_id"`
	Category       string     `json:"category"`
	Status         Status     `json:"status"`
	Severity       string     `json:"severity"`
	Message        string     `json:"message"`
	EarnedPoints   int        `json:"earned_points"`
	PossiblePoints int        `json:"possible_points"`
	Evidence       []Evidence `json:"evidence,omitempty"`
}

type Evidence struct {
	Path     string `json:"path"`
	Actual   string `json:"actual,omitempty"`
	Expected string `json:"expected,omitempty"`
}

type ATSAssessment struct {
	Status   Status    `json:"status"`
	Findings []Finding `json:"findings"`
}

type SemanticAssessment struct {
	RuleID     string  `json:"rule_id"`
	Verdict    string  `json:"verdict"`
	Evidence   string  `json:"evidence"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason"`
}

type SemanticContext struct {
	TargetRole     string `json:"target_role,omitempty"`
	JobDescription string `json:"job_description,omitempty"`
}

type SemanticCriterion struct {
	RuleID        string `json:"rule_id"`
	Label         string `json:"label"`
	MaximumPoints int    `json:"maximum_points"`
	Status        Status `json:"status"`
}

type SemanticSummary struct {
	MaximumPoints    int                 `json:"maximum_points"`
	AssessedPoints   int                 `json:"assessed_points"`
	EarnedPoints     int                 `json:"earned_points"`
	UnassessedPoints int                 `json:"unassessed_points"`
	IgnoredCount     int                 `json:"ignored_count"`
	Criteria         []SemanticCriterion `json:"criteria"`
}
