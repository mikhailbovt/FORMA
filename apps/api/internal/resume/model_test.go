package resume

import (
	"strings"
	"testing"
)

func TestProfilePhotoValidation(t *testing.T) {
	valid := Input{Title: "Resume", Document: Document{Basics: Basics{PhotoURL: "data:image/jpeg;base64,cGhvdG8="}}}
	if problems := valid.NormalizeAndValidate(); problems["document.basics.photo_url"] != "" {
		t.Fatalf("valid photo rejected: %#v", problems)
	}

	invalid := Input{Title: "Resume", Document: Document{Basics: Basics{PhotoURL: "https://example.com/photo.svg"}}}
	if problems := invalid.NormalizeAndValidate(); problems["document.basics.photo_url"] == "" {
		t.Fatal("external or SVG photo URL was accepted")
	}

	tooLarge := Input{Title: "Resume", Document: Document{Basics: Basics{PhotoURL: "data:image/jpeg;base64," + strings.Repeat("a", 700_001)}}}
	if problems := tooLarge.NormalizeAndValidate(); problems["document.basics.photo_url"] == "" {
		t.Fatal("oversized photo was accepted")
	}
}
