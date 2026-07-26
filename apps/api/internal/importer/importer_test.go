package importer

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/forma-resume/forma-smart-resume-builder/apps/api/internal/resume"
)

func TestFormaJSONIsCanonicalAndDeterministic(t *testing.T) {
	input := resume.Input{Title: "Backend CV", Document: resume.Document{
		Version: 1, Basics: resume.Basics{Name: "Ada Lovelace", Headline: "Engineer", Email: "ada@example.test",
			Links: []resume.Link{{Label: "GitHub", URL: "https://example.test/ada"}}},
		Summary:    "Builds reliable systems.",
		Experience: []resume.Experience{{ID: "exp-source", Company: "Analytical Engines", Position: "Engineer", StartDate: "2020-01", Current: true, Highlights: []string{"Reduced runtime by 20%."}}},
		Template:   "forma", PageSize: "A4", Language: "en", Order: []string{"summary", "experience"},
	}}
	data, err := json.Marshal(map[string]any{"format": "forma.resume", "version": 1, "resume": input})
	if err != nil {
		t.Fatal(err)
	}
	first, err := PreviewFile(context.Background(), "ada.json", "application/json", data)
	if err != nil {
		t.Fatal(err)
	}
	second, err := PreviewFile(context.Background(), "ada.json", "application/json", data)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("same import did not produce a deterministic preview")
	}
	if first.Parser.ID != "forma_json" || first.Candidate.Title != input.Title || !reflect.DeepEqual(first.Candidate.Document, input.Document) {
		t.Fatalf("preview = %#v", first)
	}
	if len(first.SourceSHA256) != 64 || len(first.Mappings) == 0 {
		t.Fatalf("provenance missing: %#v", first)
	}
	if first.Warnings == nil {
		t.Fatal("warnings must be an empty JSON array, not null")
	}
}

func TestFormaUIAliasesMapToCanonicalFields(t *testing.T) {
	data := []byte(`{"format":"forma.resume","version":1,"resume":{"title":"UI CV","document":{"basics":{"full_name":"Grace Hopper","profiles":[{"network":"LinkedIn","url":"https://linkedin.com/in/grace"}]},"projects":[{"id":"p1","name":"Compiler","description":"Built it","technologies":["Go"]}],"education":[{"id":"e1","institution":"Yale","degree":"PhD"}],"skills":[{"id":"s1","name":"Languages","items":["COBOL"]}],"languages":[{"id":"l1","language":"English"}],"section_order":["projects"],"paper_size":"letter"}}}`)
	preview, err := PreviewFile(context.Background(), "ui.json", "application/json", data)
	if err != nil {
		t.Fatal(err)
	}
	document := preview.Candidate.Document
	if document.Basics.Name != "Grace Hopper" || document.Projects[0].Summary != "Built it" || document.Education[0].StudyType != "PhD" || document.Skills[0].Keywords[0] != "COBOL" || document.Languages[0].Name != "English" || document.PageSize != "LETTER" {
		t.Fatalf("canonical document = %#v", document)
	}
}

func TestJSONResumeMappingAndPrivacyWarning(t *testing.T) {
	data := []byte(`{
		"basics":{"name":"Lin Chen","label":"Platform Engineer","email":"lin@example.test","summary":"Builds platforms.","image":"https://remote.test/photo.jpg","profiles":[{"network":"GitHub","url":"https://github.com/lin"}]},
		"work":[{"name":"Example Co","position":"Engineer","startDate":"2022-01-01","highlights":["Shipped APIs"]}],
		"education":[{"institution":"Example University","studyType":"BSc","area":"Computer Science"}],
		"skills":[{"name":"Backend","keywords":["Go","PostgreSQL"]}],
		"references":[{"name":"Another Person","reference":"Private reference"}]
	}`)
	preview, err := PreviewFile(context.Background(), "resume.json", "application/json", data)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Parser.ID != "json_resume" || preview.Candidate.Document.Experience[0].Company != "Example Co" || preview.Candidate.Document.Experience[0].StartDate != "2022-01-01" {
		t.Fatalf("preview = %#v", preview)
	}
	assertWarning(t, preview.Warnings, "external_image_omitted")
	assertWarning(t, preview.Warnings, "references_omitted")
}

func TestLinkedInArchiveReadsOnlyAllowlistedCSVs(t *testing.T) {
	data := makeZIP(t, map[string]string{
		"Profile.csv":        "First Name,Last Name,Headline,Summary,Geo Location,Websites\nAda,Lovelace,Engineer,Builds systems,London,https://ada.example\n",
		"Positions.csv":      "Company Name,Title,Description,Location,Started On,Finished On\nAnalytical Engines,Engineer,Built engines,London,2021-01,\n",
		"Education.csv":      "School Name,Degree Name,Field Of Study,Start Date,End Date\nUniversity,BSc,Mathematics,2015,2019\n",
		"Skills.csv":         "Name\nGo\nPostgreSQL\n",
		"Languages.csv":      "Name,Proficiency\nEnglish,Native\n",
		"Certifications.csv": "Name,Authority,Started On,Url\nCloud Cert,Issuer,2024-01,https://cert.example\n",
		"Projects.csv":       "Title,Description,Started On,Finished On,Url\nForma,Resume builder,2025,,https://forma.example\n",
		"Messages.csv":       "From,Content\nSomeone,This must never be parsed\n",
	})
	preview, err := PreviewFile(context.Background(), "linkedin.zip", "application/zip", data)
	if err != nil {
		t.Fatal(err)
	}
	document := preview.Candidate.Document
	if preview.Parser.ID != "linkedin_archive" || document.Basics.Name != "Ada Lovelace" || len(document.Experience) != 1 || len(document.Skills) != 1 || len(document.Projects) != 1 {
		t.Fatalf("preview = %#v", preview)
	}
	encoded, _ := json.Marshal(preview)
	if strings.Contains(string(encoded), "This must never be parsed") || strings.Contains(string(encoded), "Someone") {
		t.Fatalf("ignored account data leaked into preview: %s", encoded)
	}
	assertWarning(t, preview.Warnings, "linkedin_archive_allowlist")
}

func TestArchiveRejectsTraversalAndUnknownCollections(t *testing.T) {
	traversal := makeZIP(t, map[string]string{"../Profile.csv": "First Name\nAda\n"})
	if _, err := PreviewFile(context.Background(), "linkedin.zip", "application/zip", traversal); ErrorCode(err) != "unsafe_archive_path" {
		t.Fatalf("traversal error = %v (%s)", err, ErrorCode(err))
	}
	driveTraversal := makeZIP(t, map[string]string{`C:\Profile.csv`: "First Name\nAda\n"})
	if _, err := PreviewFile(context.Background(), "linkedin.zip", "application/zip", driveTraversal); ErrorCode(err) != "unsafe_archive_path" {
		t.Fatalf("drive traversal error = %v (%s)", err, ErrorCode(err))
	}
	unknown := makeZIP(t, map[string]string{"Connections.csv": "First Name\nOther\n"})
	if _, err := PreviewFile(context.Background(), "linkedin.zip", "application/zip", unknown); ErrorCode(err) != "linkedin_files_missing" {
		t.Fatalf("unknown archive error = %v (%s)", err, ErrorCode(err))
	}
	duplicate := makeZIP(t, map[string]string{"Profile.csv": "First Name\nAda\n", "nested/Profile.csv": "First Name\nGrace\n"})
	if _, err := PreviewFile(context.Background(), "linkedin.zip", "application/zip", duplicate); ErrorCode(err) != "duplicate_linkedin_file" {
		t.Fatalf("duplicate logical file error = %v (%s)", err, ErrorCode(err))
	}
}

func TestLinkedInArchiveAcceptsCaseInsensitiveCSVExtensions(t *testing.T) {
	data := makeZIP(t, map[string]string{"PROFILE.CSV": "First Name,Last Name\nAda,Lovelace\n"})
	preview, err := PreviewFile(context.Background(), "linkedin.zip", "application/zip", data)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Candidate.Document.Basics.Name != "Ada Lovelace" {
		t.Fatalf("name = %q", preview.Candidate.Document.Basics.Name)
	}
}

func TestDOCXExtractsTextWithoutDeletedContent(t *testing.T) {
	documentXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
	<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>
	<w:p><w:r><w:t>Ada Lovelace</w:t></w:r></w:p><w:p><w:r><w:t>Platform Engineer</w:t></w:r></w:p>
	<w:p><w:r><w:t>ada@example.test</w:t></w:r></w:p><w:p><w:r><w:t>SUMMARY</w:t></w:r></w:p>
	<w:p><w:r><w:t>Builds reliable systems.</w:t></w:r><w:del><w:r><w:t>Deleted secret</w:t></w:r></w:del></w:p>
	<w:p><w:r><w:t>EXPERIENCE</w:t></w:r></w:p><w:p><w:r><w:t>Engineer at Example Co</w:t></w:r></w:p>
	<w:p><w:r><w:t>2022 - Present</w:t></w:r></w:p><w:p><w:r><w:t>Shipped APIs</w:t></w:r></w:p>
	</w:body></w:document>`
	data := makeZIP(t, map[string]string{"[Content_Types].xml": "<Types/>", "word/document.xml": documentXML})
	preview, err := PreviewFile(context.Background(), "resume.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", data)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Parser.ID != "docx" || preview.Candidate.Document.Basics.Name != "Ada Lovelace" || len(preview.Candidate.Document.Experience) != 1 {
		t.Fatalf("preview = %#v", preview)
	}
	encoded, _ := json.Marshal(preview)
	if strings.Contains(string(encoded), "Deleted secret") {
		t.Fatalf("tracked deletion leaked: %s", encoded)
	}
	assertWarning(t, preview.Warnings, "docx_formatting_not_preserved")
}

func TestPDFTextLayerIsBestEffortAndWarned(t *testing.T) {
	data := makeTextPDF([]string{
		"Ada Lovelace", "Platform Engineer", "ada@example.test", "SUMMARY", "Builds reliable systems.",
		"EXPERIENCE", "Engineer at Example Co", "2022 - Present", "Shipped APIs",
	})
	preview, err := PreviewFile(context.Background(), "resume.pdf", "application/pdf", data)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Parser.ID != "pdf_text" || preview.Candidate.Document.Basics.Name == "" {
		t.Fatalf("preview = %#v", preview)
	}
	assertWarning(t, preview.Warnings, "pdf_text_order_unreliable")
	assertWarning(t, preview.Warnings, "heuristic_mapping_requires_review")
}

func TestMalformedInputsReturnStableErrorsWithoutPanics(t *testing.T) {
	tests := []struct {
		name, filename string
		data           []byte
	}{
		{"empty", "resume.json", nil},
		{"bad json", "resume.json", []byte(`{"basics":`)},
		{"bad zip", "resume.zip", []byte("PK\x03\x04broken")},
		{"bad pdf", "resume.pdf", []byte("%PDF-broken")},
		{"unknown", "resume.txt", []byte("hello")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := PreviewFile(context.Background(), test.filename, "", test.data); err == nil || ErrorCode(err) == "" {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func makeZIP(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func makeTextPDF(lines []string) []byte {
	escape := func(value string) string {
		value = strings.ReplaceAll(value, "\\", "\\\\")
		value = strings.ReplaceAll(value, "(", "\\(")
		return strings.ReplaceAll(value, ")", "\\)")
	}
	var content strings.Builder
	content.WriteString("BT /F1 12 Tf 72 750 Td ")
	for index, line := range lines {
		if index > 0 {
			content.WriteString("0 -18 Td ")
		}
		content.WriteString("(" + escape(line) + ") Tj ")
	}
	content.WriteString("ET")
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", content.Len(), content.String()),
	}
	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for index, object := range objects {
		offsets[index+1] = pdf.Len()
		fmt.Fprintf(&pdf, "%d 0 obj\n%s\nendobj\n", index+1, object)
	}
	xref := pdf.Len()
	fmt.Fprintf(&pdf, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for index := 1; index <= len(objects); index++ {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", offsets[index])
	}
	fmt.Fprintf(&pdf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return pdf.Bytes()
}

func assertWarning(t *testing.T, warnings []Warning, code string) {
	t.Helper()
	for _, warning := range warnings {
		if warning.Code == code {
			return
		}
	}
	t.Fatalf("warning %q missing from %#v", code, warnings)
}

func ExamplePreviewFile() {
	preview, _ := PreviewFile(context.Background(), "resume.json", "application/json", []byte(`{"basics":{"name":"Ada Lovelace"}}`))
	fmt.Println(preview.Parser.ID)
	// Output: json_resume
}

func FuzzPreviewFileNeverPanics(f *testing.F) {
	f.Add("resume.json", []byte(`{"basics":{"name":"Ada"}}`))
	f.Add("resume.zip", []byte("PK\x03\x04broken"))
	f.Add("resume.pdf", []byte("%PDF-broken"))
	f.Fuzz(func(t *testing.T, filename string, data []byte) {
		if len(data) > 1<<20 {
			t.Skip()
		}
		_, _ = PreviewFile(context.Background(), filename, "", data)
	})
}
