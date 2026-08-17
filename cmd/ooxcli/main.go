package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/yudaprama/gooxml/common"
	"github.com/yudaprama/gooxml/document"
	"github.com/yudaprama/gooxml/presentation"
	"github.com/yudaprama/gooxml/spreadsheet"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	cmd := args[0]
	args = args[1:]

	var err error
	switch cmd {
	case "extract":
		err = cmdExtract(args)
	case "edit":
		err = cmdEdit(args)
	case "info":
		err = cmdInfo(args)
	case "validate":
		err = cmdValidate(args)
	default:
		usage()
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `ooxcli — OOXML operations via github.com/yudaprama/gooxml

Usage:
  ooxcli extract [--save <output.md>] [--imagedir <dir>] [--baseurl <url>] <input.docx|xlsx|pptx>
                                       Extract text as Markdown (stdout, or save to file with --save)
  ooxcli edit <input.docx|xlsx|pptx> [--out <output>]            Apply edit operations (ops JSON via --ops or stdin)
  ooxcli info <input.docx|xlsx|pptx>                             Document info (JSON stdout)
  ooxcli validate <input.docx|xlsx|pptx>                         Validate document structure
`)
}

// --- helpers ----------------------------------------------------------------

func detectExt(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".docx", ".xlsx", ".pptx":
		return ext
	}
	return ""
}

func readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// --- extract ----------------------------------------------------------------

func cmdExtract(args []string) error {
	baseURL := "/files"
	savePath := ""
	imageDir := ""
	input := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--baseurl":
			i++
			if i >= len(args) {
				return errors.New("--baseurl requires a value")
			}
			baseURL = args[i]
		case "--save":
			i++
			if i >= len(args) {
				return errors.New("--save requires a value")
			}
			savePath = args[i]
		case "--imagedir":
			i++
			if i >= len(args) {
				return errors.New("--imagedir requires a value")
			}
			imageDir = args[i]
		default:
			if input == "" {
				input = args[i]
			}
		}
	}
	if input == "" {
		return errors.New("usage: ooxcli extract [--save <output.md>] [--imagedir <dir>] [--baseurl <url>] <input.docx|xlsx|pptx>")
	}

	ext := detectExt(input)
	if ext == "" {
		return fmt.Errorf("unsupported file type: %s (use .docx, .xlsx, or .pptx)", input)
	}

	switch ext {
	case ".docx":
		return extractDocx(input, baseURL, savePath, imageDir)
	case ".xlsx":
		return extractXlsx(input, baseURL, savePath, imageDir)
	case ".pptx":
		return extractPptx(input, baseURL, savePath, imageDir)
	}
	return nil
}

func extractDocx(path, baseURL, savePath, imageDir string) error {
	doc, err := document.Open(path)
	if err != nil {
		return fmt.Errorf("open docx: %w", err)
	}

	if savePath != "" {
		if imageDir == "" {
			imageDir = filepath.Join(filepath.Dir(savePath), "images")
		}
		if err := doc.SaveMarkdownWithImages(savePath, imageDir); err != nil {
			return fmt.Errorf("save markdown with images: %w", err)
		}
		fmt.Fprintf(os.Stderr, "saved: %s\n", savePath)
		fmt.Fprintf(os.Stderr, "images: %s\n", imageDir)
		return nil
	}

	md, err := doc.ToMarkdownWithImageURLs(baseURL)
	if err != nil {
		return fmt.Errorf("convert to markdown: %w", err)
	}
	fmt.Print(md)
	return nil
}

func extractXlsx(path, baseURL, savePath, imageDir string) error {
	wb, err := spreadsheet.Open(path)
	if err != nil {
		return fmt.Errorf("open xlsx: %w", err)
	}
	defer func() { _ = wb.Close() }()

	if savePath != "" {
		if imageDir == "" {
			imageDir = filepath.Join(filepath.Dir(savePath), "images")
		}
		if err := os.MkdirAll(imageDir, 0755); err != nil {
			return fmt.Errorf("create image directory: %w", err)
		}
		md, err := wb.ToMarkdownWithImages(imageDir)
		if err != nil {
			return fmt.Errorf("convert to markdown: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
		if err := os.WriteFile(savePath, []byte(md), 0644); err != nil {
			return fmt.Errorf("write markdown file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "saved: %s\n", savePath)
		fmt.Fprintf(os.Stderr, "images: %s\n", imageDir)
		return nil
	}

	md, err := wb.ToMarkdownWithImageURLs(baseURL)
	if err != nil {
		return fmt.Errorf("convert to markdown: %w", err)
	}
	fmt.Print(md)
	return nil
}

func extractPptx(path, baseURL, savePath, imageDir string) error {
	pres, err := presentation.Open(path)
	if err != nil {
		return fmt.Errorf("open pptx: %w", err)
	}

	if savePath != "" {
		if imageDir == "" {
			imageDir = filepath.Join(filepath.Dir(savePath), "images")
		}
		if err := os.MkdirAll(imageDir, 0755); err != nil {
			return fmt.Errorf("create image directory: %w", err)
		}
		md, err := pres.ToMarkdownWithImages(imageDir)
		if err != nil {
			return fmt.Errorf("convert to markdown: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
		if err := os.WriteFile(savePath, []byte(md), 0644); err != nil {
			return fmt.Errorf("write markdown file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "saved: %s\n", savePath)
		fmt.Fprintf(os.Stderr, "images: %s\n", imageDir)
		return nil
	}

	md, err := pres.ToMarkdownWithImageURLs(baseURL)
	if err != nil {
		return fmt.Errorf("convert to markdown: %w", err)
	}
	fmt.Print(md)
	return nil
}

// --- info -------------------------------------------------------------------

type infoResult struct {
	Path       string           `json:"path"`
	Type       string           `json:"type"`
	Properties map[string]string `json:"properties,omitempty"`
	Docx       *docxInfo        `json:"docx,omitempty"`
	Xlsx       *xlsxInfo        `json:"xlsx,omitempty"`
	Pptx       *pptxInfo        `json:"pptx,omitempty"`
}

type docxInfo struct {
	Paragraphs int `json:"paragraphs"`
}

type xlsxInfo struct {
	Sheets int `json:"sheets"`
	Rows   int `json:"rows"`
}

type pptxInfo struct {
	Slides int `json:"slides"`
}

func cmdInfo(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: ooxcli info <input.docx|xlsx|pptx>")
	}
	input := args[0]

	ext := detectExt(input)
	if ext == "" {
		return fmt.Errorf("unsupported file type: %s", input)
	}

	r := infoResult{Path: input, Type: ext[1:]}

	switch ext {
	case ".docx":
		doc, err := document.Open(input)
		if err != nil {
			return fmt.Errorf("open docx: %w", err)
		}
		paras := doc.Paragraphs()
		r.Docx = &docxInfo{Paragraphs: len(paras)}
		r.Properties = extractCoreProps(doc.CoreProperties)
	case ".xlsx":
		wb, err := spreadsheet.Open(input)
		if err != nil {
			return fmt.Errorf("open xlsx: %w", err)
		}
		defer func() { _ = wb.Close() }()
		sheets := wb.Sheets()
		totalRows := 0
		for _, s := range sheets {
			totalRows += len(s.Rows())
		}
		r.Xlsx = &xlsxInfo{Sheets: len(sheets), Rows: totalRows}

	case ".pptx":
		pres, err := presentation.Open(input)
		if err != nil {
			return fmt.Errorf("open pptx: %w", err)
		}
		slides := pres.Slides()
		r.Pptx = &pptxInfo{Slides: len(slides)}
	}

	out, _ := json.MarshalIndent(r, "", "  ")
	fmt.Println(string(out))
	return nil
}

func extractCoreProps(cp common.CoreProperties) map[string]string {
	m := map[string]string{}
	if v := cp.Title(); v != "" {
		m["title"] = v
	}
	if v := cp.Author(); v != "" {
		m["author"] = v
	}
	if v := cp.LastModifiedBy(); v != "" {
		m["last_modified_by"] = v
	}
	if v := cp.Description(); v != "" {
		m["description"] = v
	}
	if v := cp.Category(); v != "" {
		m["category"] = v
	}
	if v := cp.ContentStatus(); v != "" {
		m["content_status"] = v
	}
	if t := cp.Created(); !t.IsZero() {
		m["created"] = t.Format("2006-01-02T15:04:05Z")
	}
	if t := cp.Modified(); !t.IsZero() {
		m["modified"] = t.Format("2006-01-02T15:04:05Z")
	}
	return m
}

// --- validate ---------------------------------------------------------------

func cmdValidate(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: ooxcli validate <input.docx|xlsx|pptx>")
	}
	input := args[0]

	ext := detectExt(input)
	if ext == "" {
		return fmt.Errorf("unsupported file type: %s", input)
	}

	switch ext {
	case ".docx":
		doc, err := document.Open(input)
		if err != nil {
			return fmt.Errorf("open docx: %w", err)
		}
		if err := doc.Validate(); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
		// Check structural integrity
		if len(doc.Paragraphs()) == 0 {
			fmt.Fprintln(os.Stderr, "warn: document has no paragraphs")
		}
		// Read through paragraphs to check for issues
		for i, para := range doc.Paragraphs() {
			for _, run := range para.Runs() {
				_ = run.Text()
			}
			if i > 1000 {
				// Don't iterate entire document
				break
			}
		}

	case ".xlsx":
		wb, err := spreadsheet.Open(input)
		if err != nil {
			return fmt.Errorf("open xlsx: %w", err)
		}
		defer func() { _ = wb.Close() }()
		sheets := wb.Sheets()
		if len(sheets) == 0 {
			fmt.Fprintln(os.Stderr, "warn: workbook has no sheets")
		}

	case ".pptx":
		pres, err := presentation.Open(input)
		if err != nil {
			return fmt.Errorf("open pptx: %w", err)
		}
		slides := pres.Slides()
		if len(slides) == 0 {
			fmt.Fprintln(os.Stderr, "warn: presentation has no slides")
		}
	}

	fmt.Printf("%s: valid\n", input)
	return nil
}

// --- stdin pipe support -----------------------------------------------------

func readStdin() ([]byte, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return nil, errors.New("stdin is not a pipe")
	}
	return io.ReadAll(os.Stdin)
}

// --- bytes-based read (for pipe usage) --------------------------------------

func readDocxBytes(data []byte, baseURL string) (string, error) {
	doc, err := document.Read(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	return doc.ToMarkdownWithImageURLs(baseURL)
}

func readXlsxBytes(data []byte, baseURL string) (string, error) {
	wb, err := spreadsheet.Read(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	defer func() { _ = wb.Close() }()
	return wb.ToMarkdownWithImageURLs(baseURL)
}
