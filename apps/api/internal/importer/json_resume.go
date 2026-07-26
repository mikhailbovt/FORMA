package importer

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mikhailbovt/FORMA/apps/api/internal/resume"
)

type jsonResumeDocument struct {
	Basics struct {
		Name     string `json:"name"`
		Label    string `json:"label"`
		Image    string `json:"image"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		URL      string `json:"url"`
		Summary  string `json:"summary"`
		Location struct {
			Address     string `json:"address"`
			PostalCode  string `json:"postalCode"`
			City        string `json:"city"`
			CountryCode string `json:"countryCode"`
			Region      string `json:"region"`
		} `json:"location"`
		Profiles []struct {
			Network  string `json:"network"`
			Username string `json:"username"`
			URL      string `json:"url"`
		} `json:"profiles"`
	} `json:"basics"`
	Work []struct {
		Name       string   `json:"name"`
		Position   string   `json:"position"`
		URL        string   `json:"url"`
		StartDate  string   `json:"startDate"`
		EndDate    string   `json:"endDate"`
		Summary    string   `json:"summary"`
		Highlights []string `json:"highlights"`
		Location   string   `json:"location"`
	} `json:"work"`
	Education []struct {
		Institution string   `json:"institution"`
		URL         string   `json:"url"`
		Area        string   `json:"area"`
		StudyType   string   `json:"studyType"`
		StartDate   string   `json:"startDate"`
		EndDate     string   `json:"endDate"`
		Score       string   `json:"score"`
		Courses     []string `json:"courses"`
	} `json:"education"`
	Skills []struct {
		Name     string   `json:"name"`
		Level    string   `json:"level"`
		Keywords []string `json:"keywords"`
	} `json:"skills"`
	Languages []struct {
		Language string `json:"language"`
		Fluency  string `json:"fluency"`
	} `json:"languages"`
	Certificates []struct {
		Name   string `json:"name"`
		Date   string `json:"date"`
		Issuer string `json:"issuer"`
		URL    string `json:"url"`
	} `json:"certificates"`
	Projects []struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Highlights  []string `json:"highlights"`
		Keywords    []string `json:"keywords"`
		StartDate   string   `json:"startDate"`
		EndDate     string   `json:"endDate"`
		URL         string   `json:"url"`
		Roles       []string `json:"roles"`
	} `json:"projects"`
	Awards []struct {
		Title   string `json:"title"`
		Date    string `json:"date"`
		Awarder string `json:"awarder"`
		Summary string `json:"summary"`
	} `json:"awards"`
	Publications []struct {
		Name        string `json:"name"`
		Publisher   string `json:"publisher"`
		ReleaseDate string `json:"releaseDate"`
		URL         string `json:"url"`
		Summary     string `json:"summary"`
	} `json:"publications"`
	References []json.RawMessage `json:"references"`
}

func parseJSONResume(filename string, root map[string]any) (draft, error) {
	raw, err := json.Marshal(root)
	if err != nil {
		return draft{}, invalid("invalid_json_resume", "The JSON Resume document could not be decoded.", err)
	}
	var source jsonResumeDocument
	if err := json.Unmarshal(raw, &source); err != nil {
		return draft{}, invalid("invalid_json_resume", "The JSON Resume document could not be decoded.", err)
	}
	if strings.TrimSpace(source.Basics.Name) == "" && len(source.Work) == 0 && len(source.Education) == 0 && len(source.Projects) == 0 {
		return draft{}, invalid("invalid_json_resume", "The JSON Resume document contains no identifiable resume content.", nil)
	}

	document := resume.Document{
		Basics: resume.Basics{Name: source.Basics.Name, Headline: source.Basics.Label, Email: source.Basics.Email,
			Phone: source.Basics.Phone, Website: source.Basics.URL,
			Location: joinNonEmpty(", ", source.Basics.Location.Address, source.Basics.Location.PostalCode, source.Basics.Location.City, source.Basics.Location.Region, source.Basics.Location.CountryCode)},
		Summary: source.Basics.Summary,
	}
	result := draft{parserID: "json_resume"}
	if source.Basics.URL != "" {
		document.Basics.Links = append(document.Basics.Links, resume.Link{Label: "Website", URL: source.Basics.URL})
	}
	for _, profile := range source.Basics.Profiles {
		if strings.TrimSpace(profile.URL) != "" {
			document.Basics.Links = append(document.Basics.Links, resume.Link{Label: firstNonEmpty(profile.Network, profile.Username, "Profile"), URL: profile.URL})
		}
	}
	if strings.TrimSpace(source.Basics.Image) != "" {
		result.warn("external_image_omitted", "Remote profile images are not fetched during import.", "document.basics.photo_url")
	}
	for index, item := range source.Work {
		document.Experience = append(document.Experience, resume.Experience{
			ID: stableID("exp", "json_resume", index), Company: item.Name, Position: item.Position, Location: item.Location,
			StartDate: cleanDate(item.StartDate), EndDate: cleanDate(item.EndDate), Current: strings.TrimSpace(item.EndDate) == "",
			Summary: item.Summary, Highlights: cleanStrings(item.Highlights),
		})
		result.mapped(fmt.Sprintf("document.experience[%d]", index), "json_resume", fmt.Sprintf("$.work[%d]", index), "exact")
	}
	for index, item := range source.Education {
		document.Education = append(document.Education, resume.Education{
			ID: stableID("edu", "json_resume", index), Institution: item.Institution, StudyType: item.StudyType, Area: item.Area,
			StartDate: cleanDate(item.StartDate), EndDate: cleanDate(item.EndDate), Score: item.Score, Highlights: cleanStrings(item.Courses),
		})
		result.mapped(fmt.Sprintf("document.education[%d]", index), "json_resume", fmt.Sprintf("$.education[%d]", index), "exact")
	}
	for index, item := range source.Skills {
		document.Skills = append(document.Skills, resume.SkillGroup{ID: stableID("skill", "json_resume", index), Name: item.Name, Level: item.Level, Keywords: cleanStrings(item.Keywords)})
		result.mapped(fmt.Sprintf("document.skills[%d]", index), "json_resume", fmt.Sprintf("$.skills[%d]", index), "exact")
	}
	for index, item := range source.Languages {
		document.Languages = append(document.Languages, resume.Language{ID: stableID("lang", "json_resume", index), Name: item.Language, Fluency: item.Fluency})
		result.mapped(fmt.Sprintf("document.languages[%d]", index), "json_resume", fmt.Sprintf("$.languages[%d]", index), "exact")
	}
	for index, item := range source.Certificates {
		document.Certifications = append(document.Certifications, resume.Certification{ID: stableID("cert", "json_resume", index), Name: item.Name, Issuer: item.Issuer, Date: cleanDate(item.Date), URL: item.URL})
		result.mapped(fmt.Sprintf("document.certifications[%d]", index), "json_resume", fmt.Sprintf("$.certificates[%d]", index), "exact")
	}
	for index, item := range source.Projects {
		document.Projects = append(document.Projects, resume.Project{ID: stableID("project", "json_resume", index), Name: item.Name,
			Role: strings.Join(cleanStrings(item.Roles), ", "), URL: item.URL, StartDate: cleanDate(item.StartDate), EndDate: cleanDate(item.EndDate),
			Summary: item.Description, Highlights: cleanStrings(item.Highlights), Keywords: cleanStrings(item.Keywords)})
		result.mapped(fmt.Sprintf("document.projects[%d]", index), "json_resume", fmt.Sprintf("$.projects[%d]", index), "exact")
	}
	appendJSONResumeCustomSections(&document, source)
	if len(source.References) > 0 {
		result.warn("references_omitted", "References may contain third-party personal data and were not imported.", "document.custom_sections")
	}
	result.candidate = resume.Input{Title: importedTitle(filename, document.Basics.Name), Document: document}
	collectMappings(&result, "document.basics", document.Basics, "json_resume", "$.basics", "exact")
	if document.Summary != "" {
		result.mapped("document.summary", "json_resume", "$.basics.summary", "exact")
	}
	return result, nil
}

func appendJSONResumeCustomSections(document *resume.Document, source jsonResumeDocument) {
	if len(source.Awards) > 0 {
		section := resume.CustomSection{ID: "json-resume-awards", Title: "Awards"}
		for index, item := range source.Awards {
			section.Items = append(section.Items, resume.CustomItem{ID: stableID("award", "json_resume", index), Title: item.Title, Subtitle: item.Awarder, Date: cleanDate(item.Date), Summary: item.Summary})
		}
		document.CustomSections = append(document.CustomSections, section)
	}
	if len(source.Publications) > 0 {
		section := resume.CustomSection{ID: "json-resume-publications", Title: "Publications"}
		for index, item := range source.Publications {
			section.Items = append(section.Items, resume.CustomItem{ID: stableID("publication", "json_resume", index), Title: item.Name, Subtitle: item.Publisher, Date: cleanDate(item.ReleaseDate), URL: item.URL, Summary: item.Summary})
		}
		document.CustomSections = append(document.CustomSections, section)
	}
}

func joinNonEmpty(separator string, values ...string) string {
	var clean []string
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			clean = append(clean, value)
		}
	}
	return strings.Join(clean, separator)
}

func cleanStrings(values []string) []string {
	clean := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			clean = append(clean, value)
		}
	}
	return clean
}

func cleanDate(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 10 && value[4] == '-' && value[7] == '-' {
		return value[:10]
	}
	return value
}
