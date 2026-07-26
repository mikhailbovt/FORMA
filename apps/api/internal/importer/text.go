package importer

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/resume"
)

var (
	textEmail = regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`)
	textURL   = regexp.MustCompile(`(?i)\b(?:https?://|www\.)\S+`)
	textPhone = regexp.MustCompile(`(?:\+?\d[\d ()\-.]{7,}\d)`)
)

var sectionNames = map[string]string{
	"summary": "summary", "profile": "summary", "about": "summary", "professional_summary": "summary", "objective": "summary",
	"о_себе": "summary", "профиль": "summary", "цель": "summary",
	"experience": "experience", "work_experience": "experience", "employment": "experience", "employment_history": "experience", "work_history": "experience",
	"опыт": "experience", "опыт_работы": "experience",
	"education": "education", "academic_background": "education",
	"образование": "education",
	"skills":      "skills", "technical_skills": "skills", "core_skills": "skills", "competencies": "skills",
	"навыки": "skills", "ключевые_навыки": "skills",
	"projects": "projects", "selected_projects": "projects",
	"проекты":        "projects",
	"certifications": "certifications", "certificates": "certifications", "licenses_certifications": "certifications",
	"сертификаты": "certifications", "сертификации": "certifications",
	"languages": "languages",
	"языки":     "languages",
}

func mapTextResume(filename string, lines []string, parserID, source string) (draft, error) {
	if len(lines) == 0 {
		return draft{}, invalid("no_text_content", "No readable resume text was found.", nil)
	}
	result := draft{parserID: parserID}
	sections := make(map[string][]string)
	current := "header"
	meaningful := 0
	for _, original := range lines {
		line := strings.TrimSpace(original)
		if line == "" {
			if len(sections[current]) > 0 && sections[current][len(sections[current])-1] != "" {
				sections[current] = append(sections[current], "")
			}
			continue
		}
		meaningful++
		if section, ok := sectionNames[normalizeHeading(line)]; ok {
			current = section
			continue
		}
		sections[current] = append(sections[current], line)
	}
	if meaningful == 0 {
		return draft{}, invalid("no_text_content", "No readable resume text was found.", nil)
	}

	parseTextHeader(&result, sections["header"], source)
	if values := cleanStrings(sections["summary"]); len(values) > 0 {
		result.candidate.Document.Summary = strings.Join(values, " ")
		result.mapped("document.summary", parserID, source+":summary", "heuristic")
	}
	parseTextExperience(&result, sections["experience"], source)
	parseTextEducation(&result, sections["education"], source)
	parseTextProjects(&result, sections["projects"], source)
	if values := splitTextItems(sections["skills"]); len(values) > 0 {
		result.candidate.Document.Skills = []resume.SkillGroup{{ID: stableID("skill", source, 0), Name: "Skills", Keywords: values}}
		result.mapped("document.skills[0]", parserID, source+":skills", "heuristic")
	}
	for index, value := range splitTextItems(sections["languages"]) {
		name, fluency := splitPair(value)
		result.candidate.Document.Languages = append(result.candidate.Document.Languages, resume.Language{ID: stableID("lang", source, index), Name: name, Fluency: fluency})
		result.mapped(fmt.Sprintf("document.languages[%d]", index), parserID, source+":languages", "heuristic")
	}
	for index, value := range cleanStrings(sections["certifications"]) {
		result.candidate.Document.Certifications = append(result.candidate.Document.Certifications, resume.Certification{ID: stableID("cert", source, index), Name: strings.TrimLeft(value, "•*- ")})
		result.mapped(fmt.Sprintf("document.certifications[%d]", index), parserID, source+":certifications", "heuristic")
	}
	result.candidate.Title = importedTitle(filename, result.candidate.Document.Basics.Name)
	result.warn("heuristic_mapping_requires_review", "Text-derived fields are best-effort and must be reviewed before creating a resume.", "")
	if result.candidate.Document.Basics.Name == "" && len(result.candidate.Document.Experience) == 0 && len(result.candidate.Document.Education) == 0 {
		return draft{}, invalid("resume_structure_not_found", "Readable text was found, but its resume structure could not be identified reliably.", nil)
	}
	return result, nil
}

func parseTextHeader(result *draft, lines []string, source string) {
	var labels []string
	for _, line := range cleanStrings(lines) {
		if result.candidate.Document.Basics.Email == "" {
			result.candidate.Document.Basics.Email = textEmail.FindString(line)
		}
		if result.candidate.Document.Basics.Phone == "" {
			result.candidate.Document.Basics.Phone = strings.TrimSpace(textPhone.FindString(line))
		}
		if result.candidate.Document.Basics.Website == "" {
			result.candidate.Document.Basics.Website = strings.TrimRight(textURL.FindString(line), ".,;)")
		}
		stripped := textEmail.ReplaceAllString(line, "")
		stripped = textPhone.ReplaceAllString(stripped, "")
		stripped = textURL.ReplaceAllString(stripped, "")
		stripped = strings.Trim(stripped, " |,;•-")
		if stripped != "" {
			labels = append(labels, stripped)
		}
	}
	if len(labels) > 0 {
		result.candidate.Document.Basics.Name = labels[0]
	}
	if len(labels) > 1 {
		result.candidate.Document.Basics.Headline = labels[1]
	}
	collectMappings(result, "document.basics", result.candidate.Document.Basics, result.parserID, source+":header", "heuristic")
}

func parseTextExperience(result *draft, lines []string, source string) {
	for blockIndex, block := range textBlocks(lines) {
		if len(block) == 0 {
			continue
		}
		position, company := splitRoleCompany(block[0])
		cursor := 1
		if company == "" && len(block) > 1 && !looksLikeDateRange(block[1]) {
			company = block[1]
			cursor++
		}
		item := resume.Experience{ID: stableID("exp", source, blockIndex), Position: position, Company: company}
		if cursor < len(block) && looksLikeDateRange(block[cursor]) {
			item.StartDate, item.EndDate, item.Current = parseDateRange(block[cursor])
			cursor++
		}
		item.Highlights = cleanBullets(block[cursor:])
		if item.Position == "" && item.Company == "" {
			continue
		}
		index := len(result.candidate.Document.Experience)
		result.candidate.Document.Experience = append(result.candidate.Document.Experience, item)
		result.mapped(fmt.Sprintf("document.experience[%d]", index), result.parserID, source+":experience", "heuristic")
	}
}

func parseTextEducation(result *draft, lines []string, source string) {
	for blockIndex, block := range textBlocks(lines) {
		if len(block) == 0 {
			continue
		}
		item := resume.Education{ID: stableID("edu", source, blockIndex), Institution: block[0]}
		cursor := 1
		if cursor < len(block) && !looksLikeDateRange(block[cursor]) {
			item.StudyType = block[cursor]
			cursor++
		}
		if cursor < len(block) && looksLikeDateRange(block[cursor]) {
			item.StartDate, item.EndDate, _ = parseDateRange(block[cursor])
			cursor++
		}
		item.Highlights = cleanBullets(block[cursor:])
		index := len(result.candidate.Document.Education)
		result.candidate.Document.Education = append(result.candidate.Document.Education, item)
		result.mapped(fmt.Sprintf("document.education[%d]", index), result.parserID, source+":education", "heuristic")
	}
}

func parseTextProjects(result *draft, lines []string, source string) {
	for blockIndex, block := range textBlocks(lines) {
		if len(block) == 0 {
			continue
		}
		item := resume.Project{ID: stableID("project", source, blockIndex), Name: block[0]}
		if len(block) > 1 {
			item.Summary = strings.Join(cleanBullets(block[1:]), " ")
		}
		index := len(result.candidate.Document.Projects)
		result.candidate.Document.Projects = append(result.candidate.Document.Projects, item)
		result.mapped(fmt.Sprintf("document.projects[%d]", index), result.parserID, source+":projects", "heuristic")
	}
}

func normalizeHeading(value string) string {
	value = strings.TrimSpace(strings.TrimSuffix(value, ":"))
	var builder strings.Builder
	underscore := false
	for _, char := range strings.ToLower(value) {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			builder.WriteRune(char)
			underscore = false
		} else if !underscore {
			builder.WriteByte('_')
			underscore = true
		}
	}
	return strings.Trim(builder.String(), "_")
}

func textBlocks(lines []string) [][]string {
	var blocks [][]string
	var current []string
	for _, line := range lines {
		if line = strings.TrimSpace(line); line == "" {
			if len(current) > 0 {
				blocks = append(blocks, current)
				current = nil
			}
			continue
		}
		current = append(current, line)
	}
	if len(current) > 0 {
		blocks = append(blocks, current)
	}
	return blocks
}

func splitTextItems(lines []string) []string {
	var result []string
	for _, line := range cleanStrings(lines) {
		line = strings.TrimLeft(line, "•*- ")
		result = append(result, splitList(line)...)
	}
	return result
}

func cleanBullets(lines []string) []string {
	values := cleanStrings(lines)
	for index := range values {
		values[index] = strings.TrimSpace(strings.TrimLeft(values[index], "•*-"))
	}
	return cleanStrings(values)
}

func splitRoleCompany(value string) (string, string) {
	for _, separator := range []string{" at ", " | ", " — ", " – ", " - "} {
		if parts := strings.SplitN(value, separator, 2); len(parts) == 2 {
			return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		}
	}
	return strings.TrimSpace(value), ""
}

func splitPair(value string) (string, string) {
	for _, separator := range []string{" — ", " – ", " - ", ":", " | "} {
		if parts := strings.SplitN(value, separator, 2); len(parts) == 2 {
			return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		}
	}
	return strings.TrimSpace(value), ""
}

func looksLikeDateRange(value string) bool {
	lower := strings.ToLower(value)
	return regexp.MustCompile(`\b(?:19|20)\d{2}\b`).MatchString(value) && (strings.Contains(lower, "present") || strings.Contains(value, "-") || strings.Contains(value, "–") || strings.Contains(value, "—") || strings.Contains(lower, " to "))
}

func parseDateRange(value string) (string, string, bool) {
	current := strings.Contains(strings.ToLower(value), "present") || strings.Contains(strings.ToLower(value), "current")
	parts := regexp.MustCompile(`(?i)\s+(?:to|[-–—])\s+`).Split(strings.TrimSpace(value), 2)
	if len(parts) == 1 {
		return strings.TrimSpace(parts[0]), "", current
	}
	end := strings.TrimSpace(parts[1])
	if current {
		end = ""
	}
	return strings.TrimSpace(parts[0]), end, current
}
