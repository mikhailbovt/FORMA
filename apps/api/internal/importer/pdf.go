package importer

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	pdfreader "github.com/ledongthuc/pdf"
)

const maxPDFPages = 25

func parsePDF(ctx context.Context, filename string, data []byte) (draft, error) {
	reader, err := pdfreader.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "encrypt") || strings.Contains(lower, "password") {
			return draft{}, invalid("encrypted_pdf", "Password-protected PDFs cannot be imported.", err)
		}
		return draft{}, invalid("invalid_pdf", "The PDF is malformed or unsupported.", err)
	}
	pages := reader.NumPage()
	if pages == 0 {
		return draft{}, invalid("pdf_no_pages", "The PDF contains no readable pages.", nil)
	}
	if pages > maxPDFPages {
		return draft{}, invalid("pdf_page_limit", fmt.Sprintf("PDF imports are limited to %d pages.", maxPDFPages), nil)
	}
	content, err := extractPDFText(ctx, reader, pages)
	if err != nil {
		return draft{}, err
	}
	if !utf8.Valid(content) {
		content = []byte(strings.ToValidUTF8(string(content), "�"))
	}
	text := strings.ReplaceAll(string(content), "\x00", "")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	if len([]rune(strings.TrimSpace(text))) < 20 {
		return draft{}, invalid("pdf_no_text_layer", "No usable text layer was found. Export a text PDF or use a local OCR tool first.", nil)
	}
	result, err := mapTextResume(filename, strings.Split(text, "\n"), "pdf_text", "pdf:text-layer")
	if err != nil {
		return draft{}, err
	}
	result.warn("pdf_text_order_unreliable", "PDF columns, reading order, bullets, and visual grouping may not match the extracted text. Review every mapped field.", "")
	return result, nil
}

func extractPDFText(ctx context.Context, reader *pdfreader.Reader, pages int) ([]byte, error) {
	var text strings.Builder
	fonts := make(map[string]*pdfreader.Font)
	for pageNumber := 1; pageNumber <= pages; pageNumber++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		page := reader.Page(pageNumber)
		for _, name := range page.Fonts() {
			if _, exists := fonts[name]; !exists {
				font := page.Font(name)
				fonts[name] = &font
			}
		}
		pageText, err := page.GetPlainText(fonts)
		if err != nil {
			return nil, invalid("pdf_text_extraction_failed", "The PDF text layer could not be extracted.", err)
		}
		if int64(text.Len()+len(pageText)) > MaxTextBytes {
			return nil, invalid("pdf_text_limit", "The extracted PDF text exceeds the 1 MiB safety limit.", nil)
		}
		text.WriteString(pageText)
	}
	return []byte(text.String()), nil
}
