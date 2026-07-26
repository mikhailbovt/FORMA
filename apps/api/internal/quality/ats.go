package quality

import (
	"fmt"
	"unicode"

	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/resume"
)

func evaluateATS(document resume.Document) ATSAssessment {
	findings := []Finding{
		atsStructuredSections(),
		atsContact(document),
		atsDates(document),
		atsControlCharacters(document),
		atsDuplicateEvidence(document),
	}
	status := StatusPass
	for _, finding := range findings {
		switch finding.Status {
		case StatusFail:
			status = StatusFail
		case StatusWarn:
			if status != StatusFail {
				status = StatusWarn
			}
		}
	}
	return ATSAssessment{Status: status, Findings: findings}
}

func atsStructuredSections() Finding {
	return Finding{
		RuleID: "ats.structured_text_fields", Category: "ats", Status: StatusPass, Severity: "info",
		Message: "Resume content is stored in typed text fields. This is content hygiene, not a guarantee about a generated PDF or a specific ATS.",
	}
}

func atsContact(document resume.Document) Finding {
	valid := validContactCount(document.Basics)
	if valid == 0 {
		return Finding{
			RuleID: "ats.contact.machine_readable", Category: "ats", Status: StatusFail, Severity: "high",
			Message:  "No machine-readable email, phone number, or website was found.",
			Evidence: []Evidence{{Path: "$.basics", Actual: "0 valid contact methods", Expected: ">= 1"}},
		}
	}
	return Finding{
		RuleID: "ats.contact.machine_readable", Category: "ats", Status: StatusPass, Severity: "high",
		Message:  "At least one machine-readable contact method is present.",
		Evidence: []Evidence{{Path: "$.basics", Actual: fmt.Sprintf("%d valid contact method(s)", valid), Expected: ">= 1"}},
	}
}

func atsDates(document resume.Document) Finding {
	dates := allDates(document)
	if len(dates) == 0 {
		return Finding{RuleID: "ats.dates.machine_readable", Category: "ats", Status: StatusNotApplicable, Severity: "info", Message: "No dates are present to assess."}
	}
	invalid := 0
	for _, value := range dates {
		if _, ok := parseDate(value); !ok {
			invalid++
		}
	}
	if invalid > 0 {
		return Finding{
			RuleID: "ats.dates.machine_readable", Category: "ats", Status: StatusFail, Severity: "high",
			Message:  "Use YYYY, YYYY-MM, or YYYY-MM-DD dates for reliable parsing.",
			Evidence: []Evidence{{Path: "$", Actual: fmt.Sprintf("%d invalid date(s)", invalid), Expected: "0"}},
		}
	}
	return Finding{RuleID: "ats.dates.machine_readable", Category: "ats", Status: StatusPass, Severity: "high", Message: "All supplied dates use a machine-readable format."}
}

func atsControlCharacters(document resume.Document) Finding {
	count := 0
	for _, value := range documentStrings(document) {
		for _, r := range value {
			if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
				count++
			}
		}
	}
	if count > 0 {
		return Finding{
			RuleID: "ats.control_characters.absent", Category: "ats", Status: StatusFail, Severity: "medium",
			Message:  "Remove invisible control characters that may disrupt text extraction.",
			Evidence: []Evidence{{Path: "$", Actual: fmt.Sprintf("%d unsafe control character(s)", count), Expected: "0"}},
		}
	}
	return Finding{RuleID: "ats.control_characters.absent", Category: "ats", Status: StatusPass, Severity: "medium", Message: "No unsafe control characters were found."}
}

func atsDuplicateEvidence(document resume.Document) Finding {
	lines := evidenceLines(document)
	if len(lines) < 2 {
		return Finding{RuleID: "ats.duplicate_evidence.absent", Category: "ats", Status: StatusNotApplicable, Severity: "info", Message: "At least two evidence lines are needed to assess duplication."}
	}
	duplicates := duplicateLineCount(lines)
	if duplicates > 0 {
		return Finding{
			RuleID: "ats.duplicate_evidence.absent", Category: "ats", Status: StatusWarn, Severity: "low",
			Message:  "Repeated evidence can look like keyword stuffing even when it is accidental.",
			Evidence: []Evidence{{Path: "$.experience|$.projects|$.portfolio", Actual: fmt.Sprintf("%d duplicate(s)", duplicates), Expected: "0"}},
		}
	}
	return Finding{RuleID: "ats.duplicate_evidence.absent", Category: "ats", Status: StatusPass, Severity: "low", Message: "No exact duplicate evidence lines were found."}
}
