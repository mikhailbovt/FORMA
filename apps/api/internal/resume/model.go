package resume

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("resume not found")

type Resume struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Document  Document  `json:"document"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Input struct {
	Title    string   `json:"title"`
	Document Document `json:"document"`
}

type Document struct {
	Version        int             `json:"version"`
	Basics         Basics          `json:"basics"`
	Summary        string          `json:"summary,omitempty"`
	Experience     []Experience    `json:"experience,omitempty"`
	Projects       []Project       `json:"projects,omitempty"`
	Education      []Education     `json:"education,omitempty"`
	Skills         []SkillGroup    `json:"skills,omitempty"`
	Portfolio      []PortfolioItem `json:"portfolio,omitempty"`
	Certifications []Certification `json:"certifications,omitempty"`
	Languages      []Language      `json:"languages,omitempty"`
	CustomSections []CustomSection `json:"custom_sections,omitempty"`
	Order          []string        `json:"order,omitempty"`
	HiddenSections []string        `json:"hidden_sections,omitempty"`
	Template       string          `json:"template"`
	PageSize       string          `json:"page_size"`
	Language       string          `json:"language"`
}

type Basics struct {
	Name     string `json:"name,omitempty"`
	Headline string `json:"headline,omitempty"`
	Email    string `json:"email,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Location string `json:"location,omitempty"`
	Website  string `json:"website,omitempty"`
	PhotoURL string `json:"photo_url,omitempty"`
	Links    []Link `json:"links,omitempty"`
}

type Link struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type Experience struct {
	ID         string   `json:"id"`
	Company    string   `json:"company,omitempty"`
	Position   string   `json:"position,omitempty"`
	Location   string   `json:"location,omitempty"`
	StartDate  string   `json:"start_date,omitempty"`
	EndDate    string   `json:"end_date,omitempty"`
	Current    bool     `json:"current,omitempty"`
	Summary    string   `json:"summary,omitempty"`
	Highlights []string `json:"highlights,omitempty"`
}

type Project struct {
	ID         string   `json:"id"`
	Name       string   `json:"name,omitempty"`
	Role       string   `json:"role,omitempty"`
	URL        string   `json:"url,omitempty"`
	StartDate  string   `json:"start_date,omitempty"`
	EndDate    string   `json:"end_date,omitempty"`
	Summary    string   `json:"summary,omitempty"`
	Highlights []string `json:"highlights,omitempty"`
	Keywords   []string `json:"keywords,omitempty"`
}

type Education struct {
	ID          string   `json:"id"`
	Institution string   `json:"institution,omitempty"`
	StudyType   string   `json:"study_type,omitempty"`
	Area        string   `json:"area,omitempty"`
	Location    string   `json:"location,omitempty"`
	StartDate   string   `json:"start_date,omitempty"`
	EndDate     string   `json:"end_date,omitempty"`
	Score       string   `json:"score,omitempty"`
	Highlights  []string `json:"highlights,omitempty"`
}

type SkillGroup struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Keywords []string `json:"keywords,omitempty"`
	Level    string   `json:"level,omitempty"`
}

type PortfolioItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url,omitempty"`
}

type Certification struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Issuer string `json:"issuer,omitempty"`
	Date   string `json:"date,omitempty"`
	URL    string `json:"url,omitempty"`
}

type Language struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Fluency string `json:"fluency,omitempty"`
}

type CustomSection struct {
	ID    string       `json:"id"`
	Title string       `json:"title"`
	Items []CustomItem `json:"items,omitempty"`
}

type CustomItem struct {
	ID       string   `json:"id"`
	Title    string   `json:"title,omitempty"`
	Subtitle string   `json:"subtitle,omitempty"`
	Date     string   `json:"date,omitempty"`
	URL      string   `json:"url,omitempty"`
	Summary  string   `json:"summary,omitempty"`
	Bullets  []string `json:"bullets,omitempty"`
}

func (in *Input) NormalizeAndValidate() map[string]string {
	in.Title = strings.TrimSpace(in.Title)
	problems := make(map[string]string)
	if in.Title == "" {
		problems["title"] = "is required"
	} else if utf8.RuneCountInString(in.Title) > 200 {
		problems["title"] = "must be at most 200 characters"
	}

	doc := &in.Document
	if doc.Version == 0 {
		doc.Version = 1
	}
	if doc.Version != 1 {
		problems["document.version"] = "must be 1"
	}
	doc.Template = strings.TrimSpace(doc.Template)
	if doc.Template == "" {
		doc.Template = "forma"
	}
	doc.PageSize = strings.ToUpper(strings.TrimSpace(doc.PageSize))
	if doc.PageSize == "" {
		doc.PageSize = "A4"
	}
	if doc.PageSize != "A4" && doc.PageSize != "LETTER" {
		problems["document.page_size"] = "must be A4 or LETTER"
	}
	doc.Language = strings.TrimSpace(doc.Language)
	if doc.Language == "" {
		doc.Language = "en"
	}
	if doc.Basics.PhotoURL != "" {
		if !strings.HasPrefix(doc.Basics.PhotoURL, "data:image/jpeg;base64,") {
			problems["document.basics.photo_url"] = "must be a locally encoded JPEG image"
		} else if len(doc.Basics.PhotoURL) > 700_000 {
			problems["document.basics.photo_url"] = "must be smaller than 700 KB"
		}
	}

	encoded, err := json.Marshal(doc)
	if err != nil {
		problems["document"] = "cannot be encoded"
	} else if len(encoded) > 1<<20 {
		problems["document"] = "must be at most 1 MiB"
	}
	if len(doc.Experience) > 100 || len(doc.Projects) > 100 || len(doc.CustomSections) > 50 {
		problems["document"] = "contains too many entries"
	}
	return problems
}

func DecodeDocument(raw []byte) (Document, error) {
	var document Document
	if err := json.Unmarshal(raw, &document); err != nil {
		return Document{}, fmt.Errorf("decode resume document: %w", err)
	}
	return document, nil
}
