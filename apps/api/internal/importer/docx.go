package importer

import (
	"context"
	"encoding/xml"
	"io"
	"strings"
)

func parseDOCX(ctx context.Context, filename string, archive *safeArchive) (draft, error) {
	file := archive.files["word/document.xml"]
	data, err := archive.read(file, 8<<20)
	if err != nil {
		return draft{}, invalid("invalid_docx", "The DOCX document body could not be read safely.", err)
	}
	lines, err := extractDOCXParagraphs(ctx, data)
	if err != nil {
		return draft{}, invalid("invalid_docx", "The DOCX document body is malformed.", err)
	}
	result, err := mapTextResume(filename, lines, "docx", "word/document.xml")
	if err != nil {
		return draft{}, err
	}
	result.warn("docx_formatting_not_preserved", "DOCX formatting, headers, footers, images, comments, and tracked deletions were not imported.", "")
	return result, nil
}

func extractDOCXParagraphs(ctx context.Context, data []byte) ([]string, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	decoder.Strict = true
	var lines []string
	var paragraph strings.Builder
	inParagraph := false
	inText := false
	deletedDepth := 0
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch typed := token.(type) {
		case xml.StartElement:
			switch typed.Name.Local {
			case "p":
				if !inParagraph {
					paragraph.Reset()
					inParagraph = true
				}
			case "del":
				deletedDepth++
			case "t":
				inText = inParagraph && deletedDepth == 0
			case "tab":
				if inParagraph && deletedDepth == 0 {
					paragraph.WriteString(" ")
				}
			case "br":
				if inParagraph && deletedDepth == 0 {
					paragraph.WriteString("\n")
				}
			}
		case xml.CharData:
			if inText {
				paragraph.Write([]byte(typed))
			}
		case xml.EndElement:
			switch typed.Name.Local {
			case "t":
				inText = false
			case "del":
				if deletedDepth > 0 {
					deletedDepth--
				}
			case "p":
				if inParagraph {
					for _, line := range strings.Split(paragraph.String(), "\n") {
						lines = append(lines, strings.TrimSpace(line))
					}
					inParagraph = false
				}
			}
		}
	}
	return lines, nil
}
