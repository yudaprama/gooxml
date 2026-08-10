package document_test

import (
	"os"
	"strings"
	"testing"

	"github.com/yudaprama/gooxml/document"
)

func TestToMarkdown(t *testing.T) {
	doc := document.New()

	// Add a heading
	para := doc.AddParagraph()
	para.SetStyle("Heading1")
	run := para.AddRun()
	run.AddText("Test Heading")

	// Add a paragraph with bold, italic, strikethrough, and code
	para = doc.AddParagraph()
	run = para.AddRun()
	run.AddText("Normal text ")
	run = para.AddRun()
	run.Properties().SetBold(true)
	run.AddText("bold text")
	run = para.AddRun()
	run.AddText(" ")
	run = para.AddRun()
	run.Properties().SetItalic(true)
	run.AddText("italic text")
	run = para.AddRun()
	run.AddText(" ")
	run = para.AddRun()
	run.Properties().SetStrikeThrough(true)
	run.AddText("strikethrough")
	run = para.AddRun()
	run.AddText(" ")
	run = para.AddRun()
	run.Properties().SetFontFamily("Courier New")
	run.AddText("monospace")

	// Add a blockquote (indented paragraph)
	para = doc.AddParagraph()
	para.Properties().SetStartIndent(720) // 0.5 inches
	run = para.AddRun()
	run.AddText("This is a blockquote")

	// Add a table
	tbl := doc.AddTable()
	row := tbl.AddRow()
	cell := row.AddCell()
	para = cell.AddParagraph()
	para.AddRun().AddText("Header 1")
	cell = row.AddCell()
	para = cell.AddParagraph()
	para.AddRun().AddText("Header 2")

	row = tbl.AddRow()
	cell = row.AddCell()
	para = cell.AddParagraph()
	para.AddRun().AddText("Data 1")
	cell = row.AddCell()
	para = cell.AddParagraph()
	para.AddRun().AddText("Data 2")

	md := doc.ToMarkdown()
	expected := `# Test Heading

Normal text **bold text** *italic text* ~~strikethrough~~ monospace

> This is a blockquote

| Header 1 | Header 2 |
| --- | --- |
| Data 1 | Data 2 |

`
	if md != expected {
		t.Errorf("Expected:\n%q\nGot:\n%q", expected, md)
	}
}

func TestToMarkdownWithImages(t *testing.T) {
	doc := document.New()

	// Create a temporary directory for images
	tmpDir := t.TempDir()

	// Add a paragraph with text
	para := doc.AddParagraph()
	run := para.AddRun()
	run.AddText("This is a test document with images.")

	// Test with empty document (no images)
	md, err := doc.ToMarkdownWithImages(tmpDir)
	if err != nil {
		t.Fatalf("ToMarkdownWithImages failed: %v", err)
	}

	// Should contain the text
	if !strings.Contains(md, "This is a test document with images.") {
		t.Errorf("Expected text not found in markdown output: %s", md)
	}

	// Check that no images directory was created (since there are no images)
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read temp dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected no files in temp dir, got %d", len(entries))
	}
}
