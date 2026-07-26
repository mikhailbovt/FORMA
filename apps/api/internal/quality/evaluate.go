package quality

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/resume"
)

var phoneDigits = regexp.MustCompile(`\d`)

func Evaluate(document resume.Document) (Evaluation, error) {
	digest, err := sourceDigest(document)
	if err != nil {
		return Evaluation{}, err
	}
	language := normalizeLanguage(document.Language)
	findings := evaluateDeterministic(document, language)
	for _, definition := range semanticDefinitions {
		findings = append(findings, Finding{
			RuleID: definition.ruleID, Category: "semantic", Status: StatusUnassessed, Severity: "info",
			Message:        "Requires contextual semantic assessment; no model-provided points have been accepted.",
			PossiblePoints: definition.weight,
		})
	}

	evaluation := Evaluation{
		RubricVersion: RubricVersion,
		SourceDigest:  digest,
		Language:      language,
		ATSHygiene:    evaluateATS(document),
		Semantic:      initialSemanticSummary(),
		Findings:      findings,
	}
	recalculate(&evaluation)
	return evaluation, nil
}

func sourceDigest(document resume.Document) (string, error) {
	encoded, err := json.Marshal(document)
	if err != nil {
		return "", fmt.Errorf("encode quality source: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func evaluateDeterministic(document resume.Document, language string) []Finding {
	lines := evidenceLines(document)
	languageRule, languageSupported := languageCatalog[language]
	return []Finding{
		evaluateName(document),
		evaluateContact(document),
		evaluateHeadline(document),
		evaluateSubstantiveContent(document),
		evaluateEntriesComplete(document),
		evaluateEvidenceCoverage(document),
		evaluatePlaceholders(document),
		evaluateSubstantiveLines(lines),
		evaluateActionLed(lines, languageRule, languageSupported),
		evaluateResultSignals(lines, languageRule, languageSupported),
		evaluateDuplicates(lines),
		evaluateSummaryLength(document.Summary),
		evaluateLineLength(lines),
		evaluateFirstPerson(lines, languageRule, languageSupported),
		evaluateFiller(lines, languageRule, languageSupported),
		evaluatePunctuation(lines),
		evaluateDateSyntax(document),
		evaluatePeriods(document),
		evaluateReverseChronology(document),
	}
}

func initialSemanticSummary() SemanticSummary {
	criteria := make([]SemanticCriterion, 0, len(semanticDefinitions))
	for _, definition := range semanticDefinitions {
		criteria = append(criteria, SemanticCriterion{
			RuleID: definition.ruleID, Label: definition.label, MaximumPoints: definition.weight, Status: StatusUnassessed,
		})
	}
	return SemanticSummary{
		MaximumPoints: SemanticMaximum, UnassessedPoints: SemanticMaximum, Criteria: criteria,
	}
}

func recalculate(evaluation *Evaluation) {
	categories := make([]Category, 0, len(deterministicCategories)+1)
	for _, definition := range deterministicCategories {
		categories = append(categories, summarizeCategory(definition, evaluation.Findings))
	}
	categories = append(categories, summarizeCategory(categoryDefinition{id: "semantic", label: "Contextual semantic assessment", weight: SemanticMaximum}, evaluation.Findings))

	score, assessed := 0, 0
	for _, category := range categories {
		score += category.EarnedPoints
		assessed += category.AssessedPoints
	}
	blockers := make([]string, 0)
	for _, finding := range evaluation.Findings {
		if finding.Severity == "blocker" && finding.Status == StatusFail {
			blockers = append(blockers, finding.RuleID)
		}
	}
	evaluation.Quality = Scorecard{
		Score:            score,
		MaximumScore:     DeterministicMaximum + SemanticMaximum,
		AssessedPoints:   assessed,
		UnassessedPoints: DeterministicMaximum + SemanticMaximum - assessed,
		NormalizedScore:  normalizedScore(score, assessed),
		Ready:            len(blockers) == 0,
		Blockers:         blockers,
		Categories:       categories,
	}
}

func summarizeCategory(definition categoryDefinition, findings []Finding) Category {
	category := Category{ID: definition.id, Label: definition.label, MaximumPoints: definition.weight}
	status := StatusPass
	hasAssessed := false
	for _, finding := range findings {
		if finding.Category != definition.id {
			continue
		}
		if finding.Status != StatusNotApplicable && finding.Status != StatusUnassessed {
			category.AssessedPoints += finding.PossiblePoints
			category.EarnedPoints += finding.EarnedPoints
			hasAssessed = true
		}
		switch finding.Status {
		case StatusFail:
			status = StatusFail
		case StatusWarn:
			if status != StatusFail {
				status = StatusWarn
			}
		}
	}
	category.UnassessedPoints = category.MaximumPoints - category.AssessedPoints
	if !hasAssessed {
		status = StatusUnassessed
	}
	category.Status = status
	return category
}

func normalizedScore(score, assessed int) int {
	if assessed == 0 {
		return 0
	}
	return (score*100 + assessed/2) / assessed
}

func scoredFinding(ruleID, category string, status Status, severity, message string, points int, evidence ...Evidence) Finding {
	earned := 0
	switch status {
	case StatusPass:
		earned = points
	case StatusWarn:
		earned = (points + 1) / 2
	}
	return Finding{
		RuleID: ruleID, Category: category, Status: status, Severity: severity, Message: message,
		EarnedPoints: earned, PossiblePoints: points, Evidence: evidence,
	}
}

func notApplicable(ruleID, category, message string, points int) Finding {
	return scoredFinding(ruleID, category, StatusNotApplicable, "info", message, points)
}

func evaluateName(document resume.Document) Finding {
	name := strings.TrimSpace(document.Basics.Name)
	if name == "" || normalizedText(name) == "your name" {
		return scoredFinding("essentials.name.present", "essentials", StatusFail, "blocker", "Add the resume owner's name.", 3,
			Evidence{Path: "$.basics.name", Actual: "missing", Expected: "non-placeholder name"})
	}
	return scoredFinding("essentials.name.present", "essentials", StatusPass, "blocker", "A name is present.", 3)
}

func evaluateContact(document resume.Document) Finding {
	valid := validContactCount(document.Basics)
	if valid == 0 {
		return scoredFinding("essentials.contact.valid", "essentials", StatusFail, "blocker", "Add at least one valid email, phone number, or website.", 4,
			Evidence{Path: "$.basics", Actual: "0 valid contact methods", Expected: ">= 1"})
	}
	return scoredFinding("essentials.contact.valid", "essentials", StatusPass, "blocker", "At least one machine-readable contact method is present.", 4,
		Evidence{Path: "$.basics", Actual: fmt.Sprintf("%d valid contact method(s)", valid), Expected: ">= 1"})
}

func evaluateHeadline(document resume.Document) Finding {
	if len(words(document.Basics.Headline)) < 2 {
		return scoredFinding("essentials.headline.focused", "essentials", StatusFail, "high", "Add a focused professional headline with at least two words.", 3,
			Evidence{Path: "$.basics.headline", Actual: fmt.Sprintf("%d word(s)", len(words(document.Basics.Headline))), Expected: ">= 2"})
	}
	return scoredFinding("essentials.headline.focused", "essentials", StatusPass, "high", "The headline communicates a professional focus.", 3)
}

func evaluateSubstantiveContent(document resume.Document) Finding {
	if !hasSubstantiveContent(document) {
		return scoredFinding("essentials.content.substantive", "essentials", StatusFail, "blocker", "Add substantive experience, project, portfolio, or education content.", 5,
			Evidence{Path: "$", Actual: "no substantive body section", Expected: ">= 1"})
	}
	return scoredFinding("essentials.content.substantive", "essentials", StatusPass, "blocker", "The resume contains a substantive body section.", 5)
}

func evaluateEntriesComplete(document resume.Document) Finding {
	total, complete := entryCompleteness(document)
	if total == 0 {
		return notApplicable("structure.entries.complete", "structure", "No structured entries are present to assess.", 4)
	}
	ratio := float64(complete) / float64(total)
	status := StatusFail
	if ratio == 1 {
		status = StatusPass
	} else if ratio >= 0.7 {
		status = StatusWarn
	}
	return scoredFinding("structure.entries.complete", "structure", status, "medium", "Structured entries should include their identifying fields.", 4,
		Evidence{Path: "$", Actual: fmt.Sprintf("%d/%d complete", complete, total), Expected: "all entries complete"})
}

func evaluateEvidenceCoverage(document resume.Document) Finding {
	total, covered := evidenceCoverage(document)
	if total == 0 {
		return notApplicable("structure.evidence.coverage", "structure", "No experience, project, or portfolio entries are present to assess.", 3)
	}
	ratio := float64(covered) / float64(total)
	status := StatusFail
	if ratio == 1 {
		status = StatusPass
	} else if ratio >= 0.6 {
		status = StatusWarn
	}
	return scoredFinding("structure.evidence.coverage", "structure", status, "medium", "Experience, project, and portfolio entries should contain evidence-bearing text.", 3,
		Evidence{Path: "$.experience|$.projects|$.portfolio", Actual: fmt.Sprintf("%d/%d entries covered", covered, total), Expected: "all entries covered"})
}

func evaluatePlaceholders(document resume.Document) Finding {
	count := placeholderCount(document)
	if count > 0 {
		return scoredFinding("structure.placeholders.absent", "structure", StatusFail, "blocker", "Replace template placeholder text before export.", 3,
			Evidence{Path: "$", Actual: fmt.Sprintf("%d placeholder value(s)", count), Expected: "0"})
	}
	return scoredFinding("structure.placeholders.absent", "structure", StatusPass, "blocker", "No known template placeholders remain.", 3)
}

func evaluateSubstantiveLines(lines []string) Finding {
	if len(lines) == 0 {
		return notApplicable("evidence.lines.substantive", "evidence", "No evidence lines are present to assess.", 6)
	}
	count := 0
	for _, line := range lines {
		if len(words(line)) >= 5 && len([]rune(strings.TrimSpace(line))) >= 24 {
			count++
		}
	}
	return ratioFinding("evidence.lines.substantive", "evidence", "Evidence lines should contain enough context to be useful.", 6, count, len(lines), 0.8, 0.5)
}

func evaluateActionLed(lines []string, rules languageRules, supported bool) Finding {
	if !supported {
		return notApplicable("evidence.action_led", "evidence", "Action-led wording is not assessed for this document language.", 4)
	}
	if len(lines) == 0 {
		return notApplicable("evidence.action_led", "evidence", "No evidence lines are present to assess.", 4)
	}
	count := 0
	for _, line := range lines {
		if startsWithRoot(line, rules.actionRoots) {
			count++
		}
	}
	return ratioFinding("evidence.action_led", "evidence", "Lead evidence lines with a concrete action where appropriate.", 4, count, len(lines), 0.6, 0.3)
}

func evaluateResultSignals(lines []string, rules languageRules, supported bool) Finding {
	if !supported {
		return notApplicable("evidence.result_signals", "evidence", "Result wording is not assessed for this document language.", 3)
	}
	if len(lines) == 0 {
		return notApplicable("evidence.result_signals", "evidence", "No evidence lines are present to assess.", 3)
	}
	count := 0
	for _, line := range lines {
		if hasRoot(line, rules.resultRoots) {
			count++
		}
	}
	return ratioFinding("evidence.result_signals", "evidence", "Show outcomes where the source facts support them; numbers are optional.", 3, count, len(lines), 0.4, 0.01)
}

func evaluateDuplicates(lines []string) Finding {
	duplicates := duplicateLineCount(lines)
	if len(lines) < 2 {
		return notApplicable("evidence.duplicates.absent", "evidence", "At least two evidence lines are needed to assess duplication.", 2)
	}
	if duplicates > 0 {
		return scoredFinding("evidence.duplicates.absent", "evidence", StatusFail, "medium", "Remove exact duplicate evidence lines.", 2,
			Evidence{Path: "$.experience|$.projects|$.portfolio", Actual: fmt.Sprintf("%d duplicate(s)", duplicates), Expected: "0"})
	}
	return scoredFinding("evidence.duplicates.absent", "evidence", StatusPass, "medium", "Evidence lines are not exact duplicates.", 2)
}

func evaluateSummaryLength(summary string) Finding {
	count := len(words(summary))
	if count == 0 {
		return notApplicable("clarity.summary.concise", "clarity", "A summary is optional and is not present.", 3)
	}
	status := StatusFail
	if count >= 20 && count <= 100 {
		status = StatusPass
	} else if count >= 10 && count <= 160 {
		status = StatusWarn
	}
	return scoredFinding("clarity.summary.concise", "clarity", status, "medium", "Keep the optional summary concise enough to scan.", 3,
		Evidence{Path: "$.summary", Actual: fmt.Sprintf("%d words", count), Expected: "20-100 words"})
}

func evaluateLineLength(lines []string) Finding {
	if len(lines) == 0 {
		return notApplicable("clarity.evidence.concise", "clarity", "No evidence lines are present to assess.", 4)
	}
	count := 0
	for _, line := range lines {
		wordCount := len(words(line))
		if wordCount >= 5 && wordCount <= 40 {
			count++
		}
	}
	return ratioFinding("clarity.evidence.concise", "clarity", "Keep evidence lines concise and scannable.", 4, count, len(lines), 0.8, 0.5)
}

func evaluateFirstPerson(lines []string, rules languageRules, supported bool) Finding {
	if !supported {
		return notApplicable("clarity.first_person.limited", "clarity", "First-person wording is not assessed for this document language.", 2)
	}
	if len(lines) == 0 {
		return notApplicable("clarity.first_person.limited", "clarity", "No evidence lines are present to assess.", 2)
	}
	count := 0
	for _, line := range lines {
		for _, word := range words(line) {
			if _, ok := rules.firstPerson[word]; ok {
				count++
				break
			}
		}
	}
	if count == 0 {
		return scoredFinding("clarity.first_person.limited", "clarity", StatusPass, "low", "Evidence lines avoid first-person narration.", 2)
	}
	status := StatusFail
	if float64(count)/float64(len(lines)) <= 0.2 {
		status = StatusWarn
	}
	return scoredFinding("clarity.first_person.limited", "clarity", status, "low", "Use concise resume fragments instead of first-person narration.", 2,
		Evidence{Path: "$.experience|$.projects|$.portfolio", Actual: fmt.Sprintf("%d/%d lines", count, len(lines)), Expected: "0"})
}

func evaluateFiller(lines []string, rules languageRules, supported bool) Finding {
	if !supported {
		return notApplicable("clarity.filler.limited", "clarity", "Filler wording is not assessed for this document language.", 1)
	}
	if len(lines) == 0 {
		return notApplicable("clarity.filler.limited", "clarity", "No evidence lines are present to assess.", 1)
	}
	count := 0
	for _, line := range lines {
		lower := strings.ToLower(line)
		for _, phrase := range rules.fillerPhrases {
			if strings.Contains(lower, phrase) {
				count++
				break
			}
		}
	}
	if count == 0 {
		return scoredFinding("clarity.filler.limited", "clarity", StatusPass, "low", "No configured filler phrases were found.", 1)
	}
	status := StatusFail
	if float64(count)/float64(len(lines)) <= 0.1 {
		status = StatusWarn
	}
	return scoredFinding("clarity.filler.limited", "clarity", status, "low", "Replace generic filler with concrete evidence.", 1,
		Evidence{Path: "$.experience|$.projects|$.portfolio", Actual: fmt.Sprintf("%d/%d lines", count, len(lines)), Expected: "0"})
}

func evaluatePunctuation(lines []string) Finding {
	if len(lines) < 3 {
		return notApplicable("clarity.punctuation.consistent", "clarity", "At least three evidence lines are needed to assess punctuation consistency.", 2)
	}
	ended := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasSuffix(trimmed, ".") || strings.HasSuffix(trimmed, ";") {
			ended++
		}
	}
	ratio := float64(max(ended, len(lines)-ended)) / float64(len(lines))
	status := StatusWarn
	if ratio >= 0.8 {
		status = StatusPass
	}
	return scoredFinding("clarity.punctuation.consistent", "clarity", status, "low", "Use a consistent ending style across evidence lines.", 2,
		Evidence{Path: "$.experience|$.projects|$.portfolio", Actual: fmt.Sprintf("%d/%d share the majority style", max(ended, len(lines)-ended), len(lines)), Expected: ">= 80%"})
}

func evaluateDateSyntax(document resume.Document) Finding {
	dates := allDates(document)
	if len(dates) == 0 {
		return notApplicable("consistency.dates.parseable", "consistency", "No dates are present to assess.", 3)
	}
	invalid := 0
	for _, value := range dates {
		if _, ok := parseDate(value); !ok {
			invalid++
		}
	}
	if invalid > 0 {
		return scoredFinding("consistency.dates.parseable", "consistency", StatusFail, "blocker", "Use machine-readable YYYY, YYYY-MM, or YYYY-MM-DD dates.", 3,
			Evidence{Path: "$", Actual: fmt.Sprintf("%d invalid date(s)", invalid), Expected: "0"})
	}
	return scoredFinding("consistency.dates.parseable", "consistency", StatusPass, "blocker", "All supplied dates use a machine-readable format.", 3)
}

func evaluatePeriods(document resume.Document) Finding {
	total, invalid, incomplete := periodConsistency(document)
	if total == 0 {
		return notApplicable("consistency.periods.valid", "consistency", "No dated periods are present to assess.", 3)
	}
	if invalid > 0 {
		return scoredFinding("consistency.periods.valid", "consistency", StatusFail, "blocker", "Fix periods where the start follows the end or a current role also has an end date.", 3,
			Evidence{Path: "$", Actual: fmt.Sprintf("%d invalid period(s)", invalid), Expected: "0"})
	}
	if incomplete > 0 {
		return scoredFinding("consistency.periods.valid", "consistency", StatusWarn, "medium", "Complete unfinished date ranges or mark current roles explicitly.", 3,
			Evidence{Path: "$", Actual: fmt.Sprintf("%d incomplete period(s)", incomplete), Expected: "0"})
	}
	return scoredFinding("consistency.periods.valid", "consistency", StatusPass, "blocker", "Supplied date periods are internally consistent.", 3)
}

func evaluateReverseChronology(document resume.Document) Finding {
	groups := [][]string{startDatesExperience(document.Experience), startDatesProjects(document.Projects), startDatesEducation(document.Education)}
	assessed, unsorted := 0, 0
	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		assessed++
		if !datesDescending(group) {
			unsorted++
		}
	}
	if assessed == 0 {
		return notApplicable("consistency.reverse_chronological", "consistency", "At least two dated entries in one section are needed to assess ordering.", 2)
	}
	if unsorted > 0 {
		return scoredFinding("consistency.reverse_chronological", "consistency", StatusFail, "medium", "Order dated entries from most recent to oldest within each section.", 2,
			Evidence{Path: "$.experience|$.projects|$.education", Actual: fmt.Sprintf("%d unsorted section(s)", unsorted), Expected: "0"})
	}
	return scoredFinding("consistency.reverse_chronological", "consistency", StatusPass, "medium", "Dated entries are in reverse chronological order.", 2)
}

func ratioFinding(ruleID, category, message string, points, count, total int, passAt, warnAt float64) Finding {
	ratio := float64(count) / float64(total)
	status := StatusFail
	if ratio >= passAt {
		status = StatusPass
	} else if ratio >= warnAt {
		status = StatusWarn
	}
	return scoredFinding(ruleID, category, status, "medium", message, points,
		Evidence{Path: "$.experience|$.projects|$.portfolio", Actual: fmt.Sprintf("%d/%d (%.0f%%)", count, total, ratio*100), Expected: fmt.Sprintf(">= %.0f%%", passAt*100)})
}

func validContactCount(basics resume.Basics) int {
	valid := 0
	if address, err := mail.ParseAddress(strings.TrimSpace(basics.Email)); err == nil && address.Address == strings.TrimSpace(basics.Email) {
		valid++
	}
	digits := len(phoneDigits.FindAllString(basics.Phone, -1))
	if digits >= 7 && digits <= 15 {
		valid++
	}
	if validWebsite(basics.Website) {
		valid++
	}
	return valid
}

func validWebsite(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, " \t\r\n") {
		return false
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && strings.Contains(parsed.Host, ".")
}

func hasSubstantiveContent(document resume.Document) bool {
	for _, item := range document.Experience {
		if strings.TrimSpace(item.Company+item.Position+item.Summary+strings.Join(item.Highlights, "")) != "" {
			return true
		}
	}
	for _, item := range document.Projects {
		if strings.TrimSpace(item.Name+item.Summary+strings.Join(item.Highlights, "")) != "" {
			return true
		}
	}
	for _, item := range document.Portfolio {
		if strings.TrimSpace(item.Name+item.Description) != "" {
			return true
		}
	}
	for _, item := range document.Education {
		if strings.TrimSpace(item.Institution+item.StudyType+item.Area) != "" {
			return true
		}
	}
	return false
}

func entryCompleteness(document resume.Document) (int, int) {
	total, complete := 0, 0
	for _, item := range document.Experience {
		total++
		if strings.TrimSpace(item.Company) != "" && strings.TrimSpace(item.Position) != "" {
			complete++
		}
	}
	for _, item := range document.Projects {
		total++
		if strings.TrimSpace(item.Name) != "" {
			complete++
		}
	}
	for _, item := range document.Education {
		total++
		if strings.TrimSpace(item.Institution) != "" && strings.TrimSpace(item.StudyType+item.Area) != "" {
			complete++
		}
	}
	for _, item := range document.Portfolio {
		total++
		if strings.TrimSpace(item.Name) != "" {
			complete++
		}
	}
	for _, item := range document.Certifications {
		total++
		if strings.TrimSpace(item.Name) != "" {
			complete++
		}
	}
	for _, item := range document.Languages {
		total++
		if strings.TrimSpace(item.Name) != "" {
			complete++
		}
	}
	for _, item := range document.Skills {
		total++
		if strings.TrimSpace(item.Name) != "" && len(nonEmpty(item.Keywords)) > 0 {
			complete++
		}
	}
	return total, complete
}

func evidenceCoverage(document resume.Document) (int, int) {
	total, covered := 0, 0
	for _, item := range document.Experience {
		total++
		if strings.TrimSpace(item.Summary) != "" || len(nonEmpty(item.Highlights)) > 0 {
			covered++
		}
	}
	for _, item := range document.Projects {
		total++
		if strings.TrimSpace(item.Summary) != "" || len(nonEmpty(item.Highlights)) > 0 {
			covered++
		}
	}
	for _, item := range document.Portfolio {
		total++
		if strings.TrimSpace(item.Description) != "" {
			covered++
		}
	}
	return total, covered
}

func evidenceLines(document resume.Document) []string {
	result := make([]string, 0)
	for _, item := range document.Experience {
		lines := nonEmpty(item.Highlights)
		if len(lines) == 0 && strings.TrimSpace(item.Summary) != "" {
			lines = []string{strings.TrimSpace(item.Summary)}
		}
		result = append(result, lines...)
	}
	for _, item := range document.Projects {
		lines := nonEmpty(item.Highlights)
		if len(lines) == 0 && strings.TrimSpace(item.Summary) != "" {
			lines = []string{strings.TrimSpace(item.Summary)}
		}
		result = append(result, lines...)
	}
	for _, item := range document.Portfolio {
		if strings.TrimSpace(item.Description) != "" {
			result = append(result, strings.TrimSpace(item.Description))
		}
	}
	return result
}

func nonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

var placeholders = map[string]struct{}{
	"your name": {}, "new company": {}, "role": {}, "describe a verified outcome": {}, "new project": {},
	"portfolio item": {}, "describe the work and your contribution": {}, "institution": {}, "degree": {},
	"field of study": {}, "skill group": {}, "skill": {}, "certification": {}, "issuer": {}, "language": {},
	"ваше имя": {}, "новая компания": {}, "должность": {}, "опишите подтвержденный результат": {},
}

func placeholderCount(document resume.Document) int {
	count := 0
	for _, value := range documentStrings(document) {
		if _, ok := placeholders[normalizedText(value)]; ok {
			count++
		}
	}
	return count
}

func normalizedText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.Trim(value, ".,;:!?\"'`“”‘’")
	return strings.Join(strings.Fields(value), " ")
}

func duplicateLineCount(lines []string) int {
	seen := make(map[string]struct{}, len(lines))
	duplicates := 0
	for _, line := range lines {
		normalized := normalizedText(line)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			duplicates++
		} else {
			seen[normalized] = struct{}{}
		}
	}
	return duplicates
}

func allDates(document resume.Document) []string {
	values := make([]string, 0)
	appendDate := func(value string) {
		if value = strings.TrimSpace(value); value != "" {
			values = append(values, value)
		}
	}
	for _, item := range document.Experience {
		appendDate(item.StartDate)
		appendDate(item.EndDate)
	}
	for _, item := range document.Projects {
		appendDate(item.StartDate)
		appendDate(item.EndDate)
	}
	for _, item := range document.Education {
		appendDate(item.StartDate)
		appendDate(item.EndDate)
	}
	for _, item := range document.Certifications {
		appendDate(item.Date)
	}
	return values
}

func parseDate(value string) (int, bool) {
	parts := strings.Split(strings.TrimSpace(value), "-")
	if len(parts) < 1 || len(parts) > 3 || len(parts[0]) != 4 {
		return 0, false
	}
	year, err := strconv.Atoi(parts[0])
	if err != nil || year < 1900 || year > 2200 {
		return 0, false
	}
	month, day := 1, 1
	if len(parts) >= 2 {
		if len(parts[1]) != 2 {
			return 0, false
		}
		month, err = strconv.Atoi(parts[1])
		if err != nil || month < 1 || month > 12 {
			return 0, false
		}
	}
	if len(parts) == 3 {
		if len(parts[2]) != 2 {
			return 0, false
		}
		day, err = strconv.Atoi(parts[2])
		if err != nil || day < 1 || day > 31 {
			return 0, false
		}
	}
	return year*10_000 + month*100 + day, true
}

func periodConsistency(document resume.Document) (int, int, int) {
	total, invalid, incomplete := 0, 0, 0
	check := func(start, end string, current bool) {
		start, end = strings.TrimSpace(start), strings.TrimSpace(end)
		if start == "" && end == "" && !current {
			return
		}
		total++
		if current && end != "" {
			invalid++
			return
		}
		if start == "" && end != "" {
			invalid++
			return
		}
		if start != "" && end == "" && !current {
			incomplete++
			return
		}
		startValue, startOK := parseDate(start)
		endValue, endOK := parseDate(end)
		if start != "" && end != "" && startOK && endOK && startValue > endValue {
			invalid++
		}
	}
	for _, item := range document.Experience {
		check(item.StartDate, item.EndDate, item.Current)
	}
	for _, item := range document.Projects {
		check(item.StartDate, item.EndDate, false)
	}
	for _, item := range document.Education {
		check(item.StartDate, item.EndDate, false)
	}
	return total, invalid, incomplete
}

func startDatesExperience(items []resume.Experience) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.StartDate) != "" {
			values = append(values, item.StartDate)
		}
	}
	return values
}

func startDatesProjects(items []resume.Project) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.StartDate) != "" {
			values = append(values, item.StartDate)
		}
	}
	return values
}

func startDatesEducation(items []resume.Education) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.StartDate) != "" {
			values = append(values, item.StartDate)
		}
	}
	return values
}

func datesDescending(values []string) bool {
	parsed := make([]int, 0, len(values))
	for _, value := range values {
		if date, ok := parseDate(value); ok {
			parsed = append(parsed, date)
		}
	}
	if len(parsed) < 2 {
		return true
	}
	return sort.SliceIsSorted(parsed, func(left, right int) bool { return parsed[left] > parsed[right] })
}

func documentStrings(document resume.Document) []string {
	encoded, err := json.Marshal(document)
	if err != nil {
		return nil
	}
	var value any
	if json.Unmarshal(encoded, &value) != nil {
		return nil
	}
	result := make([]string, 0)
	collectStrings(value, &result)
	return result
}

func collectStrings(value any, result *[]string) {
	switch typed := value.(type) {
	case string:
		*result = append(*result, typed)
	case []any:
		for _, child := range typed {
			collectStrings(child, result)
		}
	case map[string]any:
		for key, child := range typed {
			if key == "photo_url" {
				continue
			}
			collectStrings(child, result)
		}
	}
}
