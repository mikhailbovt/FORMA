package importer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/resume"
)

func parseJSON(filename string, data []byte) (draft, error) {
	if int64(len(data)) > MaxTextBytes*2 {
		return draft{}, invalid("json_too_large", "JSON imports must be at most 2 MiB.", nil)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var root map[string]any
	if err := decoder.Decode(&root); err != nil {
		return draft{}, invalid("invalid_json", "The file is not valid JSON.", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return draft{}, invalid("invalid_json", "The file must contain one JSON object.", err)
	}
	if len(root) == 0 {
		return draft{}, invalid("invalid_json", "The JSON object is empty.", nil)
	}

	if format, _ := root["format"].(string); format == "forma.resume" {
		wrapped, ok := root["resume"].(map[string]any)
		if !ok {
			return draft{}, invalid("invalid_forma_json", "The Forma export is missing its resume object.", nil)
		}
		root = wrapped
	}
	if _, hasDocument := root["document"]; hasDocument {
		return parseFormaObject(filename, root)
	}
	if _, hasWork := root["work"]; hasWork || root["basics"] != nil || root["education"] != nil || root["skills"] != nil {
		return parseJSONResume(filename, root)
	}
	return draft{}, invalid("unknown_json_schema", "JSON must be a Forma export or a JSON Resume document.", nil)
}

type flexibleResume struct {
	Title    string           `json:"title"`
	Document flexibleDocument `json:"document"`
}

type flexibleDocument struct {
	Version        int                    `json:"version"`
	Basics         flexibleBasics         `json:"basics"`
	Summary        string                 `json:"summary"`
	Experience     []resume.Experience    `json:"experience"`
	Projects       []flexibleProject      `json:"projects"`
	Education      []flexibleEducation    `json:"education"`
	Skills         []flexibleSkill        `json:"skills"`
	Portfolio      []resume.PortfolioItem `json:"portfolio"`
	Certifications []resume.Certification `json:"certifications"`
	Languages      []flexibleLanguage     `json:"languages"`
	CustomSections []resume.CustomSection `json:"custom_sections"`
	Order          []string               `json:"order"`
	SectionOrder   []string               `json:"section_order"`
	HiddenSections []string               `json:"hidden_sections"`
	Template       string                 `json:"template"`
	PageSize       string                 `json:"page_size"`
	PaperSize      string                 `json:"paper_size"`
	Language       string                 `json:"language"`
}

type flexibleBasics struct {
	Name     string            `json:"name"`
	FullName string            `json:"full_name"`
	Headline string            `json:"headline"`
	Email    string            `json:"email"`
	Phone    string            `json:"phone"`
	Location string            `json:"location"`
	Website  string            `json:"website"`
	PhotoURL string            `json:"photo_url"`
	Links    []resume.Link     `json:"links"`
	Profiles []flexibleProfile `json:"profiles"`
}

type flexibleProfile struct {
	Network string `json:"network"`
	Label   string `json:"label"`
	URL     string `json:"url"`
}

type flexibleProject struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Role         string   `json:"role"`
	URL          string   `json:"url"`
	StartDate    string   `json:"start_date"`
	EndDate      string   `json:"end_date"`
	Summary      string   `json:"summary"`
	Description  string   `json:"description"`
	Highlights   []string `json:"highlights"`
	Keywords     []string `json:"keywords"`
	Technologies []string `json:"technologies"`
}

type flexibleEducation struct {
	ID          string   `json:"id"`
	Institution string   `json:"institution"`
	StudyType   string   `json:"study_type"`
	Degree      string   `json:"degree"`
	Area        string   `json:"area"`
	Location    string   `json:"location"`
	StartDate   string   `json:"start_date"`
	EndDate     string   `json:"end_date"`
	Score       string   `json:"score"`
	Highlights  []string `json:"highlights"`
}

type flexibleSkill struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Keywords []string `json:"keywords"`
	Items    []string `json:"items"`
	Level    string   `json:"level"`
}

type flexibleLanguage struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Language string `json:"language"`
	Fluency  string `json:"fluency"`
}

func parseFormaObject(filename string, root map[string]any) (draft, error) {
	raw, err := json.Marshal(root)
	if err != nil {
		return draft{}, invalid("invalid_forma_json", "The Forma export could not be decoded.", err)
	}
	var source flexibleResume
	if err := json.Unmarshal(raw, &source); err != nil {
		return draft{}, invalid("invalid_forma_json", "The Forma export could not be decoded.", err)
	}
	document := resume.Document{
		Version: source.Document.Version,
		Basics: resume.Basics{
			Name: firstNonEmpty(source.Document.Basics.Name, source.Document.Basics.FullName), Headline: source.Document.Basics.Headline,
			Email: source.Document.Basics.Email, Phone: source.Document.Basics.Phone, Location: source.Document.Basics.Location,
			Website: source.Document.Basics.Website, PhotoURL: source.Document.Basics.PhotoURL, Links: source.Document.Basics.Links,
		},
		Summary: source.Document.Summary, Experience: source.Document.Experience, Portfolio: source.Document.Portfolio,
		Certifications: source.Document.Certifications, CustomSections: source.Document.CustomSections,
		HiddenSections: source.Document.HiddenSections, Template: source.Document.Template,
		PageSize: firstNonEmpty(source.Document.PageSize, strings.ToUpper(source.Document.PaperSize)), Language: source.Document.Language,
	}
	for _, profile := range source.Document.Basics.Profiles {
		document.Basics.Links = append(document.Basics.Links, resume.Link{Label: firstNonEmpty(profile.Network, profile.Label, "Link"), URL: profile.URL})
	}
	for _, item := range source.Document.Projects {
		document.Projects = append(document.Projects, resume.Project{ID: item.ID, Name: item.Name, Role: item.Role, URL: item.URL,
			StartDate: item.StartDate, EndDate: item.EndDate, Summary: firstNonEmpty(item.Summary, item.Description),
			Highlights: item.Highlights, Keywords: firstNonEmptySlice(item.Keywords, item.Technologies)})
	}
	for _, item := range source.Document.Education {
		document.Education = append(document.Education, resume.Education{ID: item.ID, Institution: item.Institution,
			StudyType: firstNonEmpty(item.StudyType, item.Degree), Area: item.Area, Location: item.Location,
			StartDate: item.StartDate, EndDate: item.EndDate, Score: item.Score, Highlights: item.Highlights})
	}
	for _, item := range source.Document.Skills {
		document.Skills = append(document.Skills, resume.SkillGroup{ID: item.ID, Name: item.Name,
			Keywords: firstNonEmptySlice(item.Keywords, item.Items), Level: item.Level})
	}
	for _, item := range source.Document.Languages {
		document.Languages = append(document.Languages, resume.Language{ID: item.ID, Name: firstNonEmpty(item.Name, item.Language), Fluency: item.Fluency})
	}
	document.Order = firstNonEmptySlice(source.Document.Order, source.Document.SectionOrder)

	result := draft{parserID: "forma_json", candidate: resume.Input{Title: source.Title, Document: document}}
	if result.candidate.Title == "" {
		result.candidate.Title = importedTitle(filename, document.Basics.Name)
	}
	collectMappings(&result, "document", document, "forma_json", "document", "exact")
	return result, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstNonEmptySlice[T any](values ...[]T) []T {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return nil
}

func collectMappings(result *draft, rootPath string, value any, source, locator, status string) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return
	}
	var generic any
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if decoder.Decode(&generic) != nil {
		return
	}
	collectMappingValue(result, rootPath, generic, source, locator, status)
}

func collectMappingValue(result *draft, path string, value any, source, locator, status string) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			collectMappingValue(result, path+"."+key, child, source, locator, status)
		}
	case []any:
		for index, child := range typed {
			collectMappingValue(result, fmt.Sprintf("%s[%d]", path, index), child, source, locator, status)
		}
	case string:
		if strings.TrimSpace(typed) != "" {
			result.mapped(path, source, locator, status)
		}
	case json.Number:
		result.mapped(path, source, locator, status)
	case bool:
		if typed {
			result.mapped(path, source, locator, status)
		}
	}
}
