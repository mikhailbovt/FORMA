package importer

import (
	"context"
	"encoding/csv"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/resume"
)

var linkedinFiles = map[string]string{
	"profile": "profile", "positions": "positions", "education": "education", "skills": "skills",
	"languages": "languages", "certifications": "certifications", "projects": "projects",
}

type csvTable struct {
	name    string
	headers map[string]int
	rows    [][]string
}

func parseLinkedInArchive(ctx context.Context, filename string, archive *safeArchive) (draft, error) {
	tables := make(map[string]csvTable)
	invalidUTF8 := false
	for archiveName, file := range archive.files {
		if err := ctx.Err(); err != nil {
			return draft{}, err
		}
		archiveBase := strings.ToLower(path.Base(archiveName))
		extension := strings.ToLower(path.Ext(archiveBase))
		base := strings.TrimSuffix(archiveBase, extension)
		kind, allowed := linkedinFiles[normalizeHeader(base)]
		if !allowed || extension != ".csv" {
			continue
		}
		data, err := archive.read(file, 4<<20)
		if err != nil {
			return draft{}, invalid("linkedin_csv_invalid", "A LinkedIn CSV file could not be read safely.", err)
		}
		if !utf8.Valid(data) {
			invalidUTF8 = true
			data = []byte(strings.ToValidUTF8(string(data), "�"))
		}
		table, err := decodeCSV(path.Base(archiveName), data)
		if err != nil {
			return draft{}, invalid("linkedin_csv_invalid", "A LinkedIn CSV file is malformed.", err)
		}
		if _, exists := tables[kind]; exists {
			return draft{}, invalid("duplicate_linkedin_file", "The archive contains more than one copy of a resume-related LinkedIn CSV file.", nil)
		}
		tables[kind] = table
	}
	if len(tables) == 0 {
		return draft{}, invalid("linkedin_files_missing", "No allowlisted LinkedIn profile CSV files were found. Request Profile, Positions, Education, Skills, Languages, Certifications, and Projects in LinkedIn's data export.", nil)
	}

	result := draft{parserID: "linkedin_archive"}
	if invalidUTF8 {
		result.warn("invalid_utf8_replaced", "Invalid text bytes were replaced while reading the LinkedIn export.", "")
	}
	parseLinkedInProfile(&result, tables["profile"])
	parseLinkedInPositions(&result, tables["positions"])
	parseLinkedInEducation(&result, tables["education"])
	parseLinkedInSkills(&result, tables["skills"])
	parseLinkedInLanguages(&result, tables["languages"])
	parseLinkedInCertifications(&result, tables["certifications"])
	parseLinkedInProjects(&result, tables["projects"])
	result.candidate.Title = importedTitle(filename, result.candidate.Document.Basics.Name)
	result.warn("linkedin_archive_allowlist", "Only resume-related LinkedIn CSV files were read; connections, messages, contacts, and other account data were ignored.", "")
	return result, nil
}

func decodeCSV(name string, data []byte) (csvTable, error) {
	data = []byte(strings.TrimPrefix(string(data), "\ufeff"))
	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = false
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return csvTable{}, err
	}
	if len(records) == 0 {
		return csvTable{name: name, headers: map[string]int{}}, nil
	}
	headers := make(map[string]int, len(records[0]))
	for index, header := range records[0] {
		headers[normalizeHeader(header)] = index
	}
	return csvTable{name: name, headers: headers, rows: records[1:]}, nil
}

var nonHeader = regexp.MustCompile(`[^a-z0-9]+`)

func normalizeHeader(value string) string {
	return strings.Trim(nonHeader.ReplaceAllString(strings.ToLower(strings.TrimSpace(value)), "_"), "_")
}

func (table csvTable) value(row []string, aliases ...string) string {
	for _, alias := range aliases {
		if index, ok := table.headers[normalizeHeader(alias)]; ok && index < len(row) {
			if value := strings.TrimSpace(row[index]); value != "" {
				return value
			}
		}
	}
	return ""
}

func (table csvTable) locator(row int) string { return table.name + ":row " + strconv.Itoa(row+2) }

func parseLinkedInProfile(result *draft, table csvTable) {
	if len(table.rows) == 0 {
		return
	}
	row := table.rows[0]
	basics := &result.candidate.Document.Basics
	basics.Name = joinNonEmpty(" ", table.value(row, "First Name", "FirstName"), table.value(row, "Last Name", "LastName"))
	basics.Headline = table.value(row, "Headline", "Title")
	basics.Location = firstNonEmpty(table.value(row, "Geo Location", "Location"), joinNonEmpty(" ", table.value(row, "City"), table.value(row, "Zip Code")))
	result.candidate.Document.Summary = table.value(row, "Summary", "About")
	websites := table.value(row, "Websites", "Website")
	if websites != "" {
		parts := splitList(websites)
		if len(parts) > 0 {
			basics.Website = parts[0]
			for _, item := range parts {
				basics.Links = append(basics.Links, resume.Link{Label: "Website", URL: item})
			}
		}
	}
	collectMappings(result, "document.basics", *basics, "linkedin_archive", table.locator(0), "exact")
	if result.candidate.Document.Summary != "" {
		result.mapped("document.summary", "linkedin_archive", table.locator(0), "exact")
	}
}

func parseLinkedInPositions(result *draft, table csvTable) {
	for rowIndex, row := range table.rows {
		company := table.value(row, "Company Name", "Company")
		position := table.value(row, "Title", "Position")
		if company == "" && position == "" {
			continue
		}
		end := table.value(row, "Finished On", "End Date", "Ended On")
		index := len(result.candidate.Document.Experience)
		result.candidate.Document.Experience = append(result.candidate.Document.Experience, resume.Experience{
			ID: stableID("exp", table.name, rowIndex), Company: company, Position: position,
			Location: table.value(row, "Location"), StartDate: cleanDate(table.value(row, "Started On", "Start Date")),
			EndDate: cleanDate(end), Current: end == "", Summary: table.value(row, "Description", "Summary"),
		})
		result.mapped(fmt.Sprintf("document.experience[%d]", index), "linkedin_archive", table.locator(rowIndex), "exact")
	}
}

func parseLinkedInEducation(result *draft, table csvTable) {
	for rowIndex, row := range table.rows {
		institution := table.value(row, "School Name", "School", "Institution")
		if institution == "" {
			continue
		}
		index := len(result.candidate.Document.Education)
		highlights := cleanStrings([]string{table.value(row, "Activities", "Activities and Societies"), table.value(row, "Notes", "Description")})
		result.candidate.Document.Education = append(result.candidate.Document.Education, resume.Education{
			ID: stableID("edu", table.name, rowIndex), Institution: institution,
			StudyType: table.value(row, "Degree Name", "Degree"), Area: table.value(row, "Field Of Study", "Field of Study"),
			StartDate: cleanDate(table.value(row, "Start Date", "Started On")), EndDate: cleanDate(table.value(row, "End Date", "Finished On")), Highlights: highlights,
		})
		result.mapped(fmt.Sprintf("document.education[%d]", index), "linkedin_archive", table.locator(rowIndex), "exact")
	}
}

func parseLinkedInSkills(result *draft, table csvTable) {
	var skills []string
	for _, row := range table.rows {
		if value := table.value(row, "Name", "Skill"); value != "" {
			skills = append(skills, value)
		}
	}
	if len(skills) > 0 {
		result.candidate.Document.Skills = append(result.candidate.Document.Skills, resume.SkillGroup{ID: stableID("skill", table.name, 0), Name: "Skills", Keywords: skills})
		result.mapped("document.skills[0]", "linkedin_archive", table.name, "exact")
	}
}

func parseLinkedInLanguages(result *draft, table csvTable) {
	for rowIndex, row := range table.rows {
		name := table.value(row, "Name", "Language")
		if name == "" {
			continue
		}
		index := len(result.candidate.Document.Languages)
		result.candidate.Document.Languages = append(result.candidate.Document.Languages, resume.Language{ID: stableID("lang", table.name, rowIndex), Name: name, Fluency: table.value(row, "Proficiency", "Fluency")})
		result.mapped(fmt.Sprintf("document.languages[%d]", index), "linkedin_archive", table.locator(rowIndex), "exact")
	}
}

func parseLinkedInCertifications(result *draft, table csvTable) {
	for rowIndex, row := range table.rows {
		name := table.value(row, "Name", "Certification Name")
		if name == "" {
			continue
		}
		index := len(result.candidate.Document.Certifications)
		result.candidate.Document.Certifications = append(result.candidate.Document.Certifications, resume.Certification{
			ID: stableID("cert", table.name, rowIndex), Name: name, Issuer: table.value(row, "Authority", "Issuer"),
			Date: cleanDate(table.value(row, "Started On", "Date")), URL: table.value(row, "Url", "URL"),
		})
		result.mapped(fmt.Sprintf("document.certifications[%d]", index), "linkedin_archive", table.locator(rowIndex), "exact")
	}
}

func parseLinkedInProjects(result *draft, table csvTable) {
	for rowIndex, row := range table.rows {
		name := table.value(row, "Title", "Name", "Project Name")
		if name == "" {
			continue
		}
		index := len(result.candidate.Document.Projects)
		result.candidate.Document.Projects = append(result.candidate.Document.Projects, resume.Project{
			ID: stableID("project", table.name, rowIndex), Name: name, Summary: table.value(row, "Description", "Summary"),
			StartDate: cleanDate(table.value(row, "Started On", "Start Date")), EndDate: cleanDate(table.value(row, "Finished On", "End Date")),
			URL: table.value(row, "Url", "URL"),
		})
		result.mapped(fmt.Sprintf("document.projects[%d]", index), "linkedin_archive", table.locator(rowIndex), "exact")
	}
}

func splitList(value string) []string {
	return cleanStrings(strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' || r == '\n' || r == '|' }))
}
