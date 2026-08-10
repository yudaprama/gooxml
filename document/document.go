// Copyright 2017 Baliance. All rights reserved.
//
// Use of this source code is governed by the terms of the Affero GNU General
// Public License version 3.0 as published by the Free Software Foundation and
// appearing in the file LICENSE included in the packaging of this file. A
// commercial license can be purchased by contacting sales@baliance.com.

package document

import (
	"archive/zip"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/yudaprama/gooxml"
	"github.com/yudaprama/gooxml/common"
	"github.com/yudaprama/gooxml/zippkg"

	"github.com/yudaprama/gooxml/schema/soo/dml"
	pic "github.com/yudaprama/gooxml/schema/soo/dml/picture"
	st "github.com/yudaprama/gooxml/schema/soo/ofc/sharedTypes"
	"github.com/yudaprama/gooxml/schema/soo/pkg/relationships"
	"github.com/yudaprama/gooxml/schema/soo/wml"
)

// Document is a text document that can be written out in the OOXML .docx
// format. It can be opened from a file on disk and modified, or created from
// scratch.
type Document struct {
	common.DocBase
	x *wml.Document

	Settings  Settings  // document settings
	Numbering Numbering // numbering styles within the doucment
	Styles    Styles    // styles that are use and can be used within the document

	headers []*wml.Hdr
	hdrRels []common.Relationships

	footers []*wml.Ftr

	docRels     common.Relationships
	themes      []*dml.Theme
	webSettings *wml.WebSettings
	fontTable   *wml.Fonts
	endNotes    *wml.Endnotes
	footNotes   *wml.Footnotes
}

// New constructs an empty document that content can be added to.
func New() *Document {

	d := &Document{x: wml.NewDocument()}
	d.ContentTypes = common.NewContentTypes()
	d.x.Body = wml.NewCT_Body()
	d.x.ConformanceAttr = st.ST_ConformanceClassTransitional
	d.docRels = common.NewRelationships()

	d.AppProperties = common.NewAppProperties()
	d.CoreProperties = common.NewCoreProperties()

	d.ContentTypes.AddOverride("/word/document.xml", "application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml")

	d.Settings = NewSettings()
	d.docRels.AddRelationship("settings.xml", gooxml.SettingsType)
	d.ContentTypes.AddOverride("/word/settings.xml", "application/vnd.openxmlformats-officedocument.wordprocessingml.settings+xml")

	d.Rels = common.NewRelationships()
	d.Rels.AddRelationship(gooxml.RelativeFilename(gooxml.DocTypeDocument, "", gooxml.CorePropertiesType, 0), gooxml.CorePropertiesType)
	d.Rels.AddRelationship("docProps/app.xml", gooxml.ExtendedPropertiesType)
	d.Rels.AddRelationship("word/document.xml", gooxml.OfficeDocumentType)

	d.Numbering = NewNumbering()
	d.Numbering.InitializeDefault()
	d.ContentTypes.AddOverride("/word/numbering.xml", "application/vnd.openxmlformats-officedocument.wordprocessingml.numbering+xml")
	d.docRels.AddRelationship("numbering.xml", gooxml.NumberingType)

	d.Styles = NewStyles()
	d.Styles.InitializeDefault()
	d.ContentTypes.AddOverride("/word/styles.xml", "application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml")
	d.docRels.AddRelationship("styles.xml", gooxml.StylesType)

	d.x.Body = wml.NewCT_Body()
	return d
}

// X returns the inner wrapped XML type.
func (d *Document) X() *wml.Document {
	return d.x
}

// AddHeader creates a header associated with the document, but doesn't add it
// to the document for display.
func (d *Document) AddHeader() Header {
	hdr := wml.NewHdr()
	d.headers = append(d.headers, hdr)
	path := fmt.Sprintf("header%d.xml", len(d.headers))
	d.docRels.AddRelationship(path, gooxml.HeaderType)

	d.ContentTypes.AddOverride("/word/"+path, "application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml")
	d.hdrRels = append(d.hdrRels, common.NewRelationships())

	return Header{d, hdr}
}

// Headers returns the headers defined in the document.
func (d *Document) Headers() []Header {
	ret := []Header{}
	for _, h := range d.headers {
		ret = append(ret, Header{d, h})
	}
	return ret
}

// ToMarkdown converts the document to a markdown string.
func (d *Document) ToMarkdown() string {
	var md strings.Builder
	if d.x.Body == nil {
		return ""
	}
	for _, ble := range d.x.Body.EG_BlockLevelElts {
		for _, c := range ble.EG_ContentBlockContent {
			// handle tables
			for _, tbl := range c.Tbl {
				md.WriteString(d.tableToMarkdown(tbl))
				md.WriteString("\n")
			}
			// handle paragraphs
			for _, p := range c.P {
				md.WriteString(d.paragraphToMarkdown(p))
			}
		}
	}
	return md.String()
}

// ToMarkdownWithImages converts the document to markdown with images extracted to a local directory.
// The imageDir parameter specifies where to save extracted images. Images will be saved with
// names like "image1.png", "image2.jpg", etc. and referenced in markdown with relative paths.
func (d *Document) ToMarkdownWithImages(imageDir string) (string, error) {
	// Create image directory if it doesn't exist
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create image directory: %w", err)
	}

	var md strings.Builder
	imageCounter := 0

	if d.x.Body == nil {
		return "", nil
	}

	for _, ble := range d.x.Body.EG_BlockLevelElts {
		for _, c := range ble.EG_ContentBlockContent {
			// handle tables
			for _, tbl := range c.Tbl {
				md.WriteString(d.tableToMarkdown(tbl))
				md.WriteString("\n")
			}

			// handle paragraphs
			for _, p := range c.P {
				paraMarkdown, err := d.paragraphToMarkdownWithImages(p, imageDir, &imageCounter)
				if err != nil {
					gooxml.Log("failed to process paragraph: %s", err)
					continue
				}
				md.WriteString(paraMarkdown)
			}
		}
	}

	return md.String(), nil
}

// ToMarkdownWithImageURLs converts the document to markdown with images served via local fileserver URLs.
// Images are automatically saved to the user data directory and become accessible via the Wails fileserver.
// The baseURL parameter should be "/files" to match the user data fileserver route configured in main.go.
//
// Example usage:
//
//	doc, err := document.Open("document.docx")
//	if err != nil {
//	    return err
//	}
//	markdown, err := doc.ToMarkdownWithImageURLs("/files")
//	// Images are saved to user data directory (e.g., ~/Library/Application Support/veridium/images/)
//	// and referenced as: ![alt text](/files/images/image1.png)
//
// Note: Static frontend assets are served by Wails' built-in asset server (no fileserver needed).
// User-generated images are served by the /files route from the user config directory.
func (d *Document) ToMarkdownWithImageURLs(baseURL string) (string, error) {
	// Get user config directory for storing user-generated content
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to current directory if we can't get user config dir
		userConfigDir = "."
	}

	// Create veridium app data directory
	appDataDir := filepath.Join(userConfigDir, "veridium")
	imageDir := filepath.Join(appDataDir, "images")

	// Create image directory if it doesn't exist
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create image directory: %w", err)
	}

	var md strings.Builder
	imageCounter := 0

	if d.x.Body == nil {
		return "", nil
	}

	for _, ble := range d.x.Body.EG_BlockLevelElts {
		for _, c := range ble.EG_ContentBlockContent {
			// handle tables
			for _, tbl := range c.Tbl {
				md.WriteString(d.tableToMarkdown(tbl))
				md.WriteString("\n")
			}

			// handle paragraphs
			for _, p := range c.P {
				paraMarkdown, err := d.paragraphToMarkdownWithImageURLs(p, imageDir, baseURL, &imageCounter)
				if err != nil {
					gooxml.Log("failed to process paragraph: %s", err)
					continue
				}
				md.WriteString(paraMarkdown)
			}
		}
	}

	return md.String(), nil
}

func (d *Document) paragraphToMarkdownWithImageURLs(p *wml.CT_P, imageDir string, baseURL string, imageCounter *int) (string, error) {
	var para strings.Builder
	style := ""
	if p.PPr != nil && p.PPr.PStyle != nil {
		style = p.PPr.PStyle.ValAttr
	}

	// Check for hyperlinks and regular content
	for _, ec := range p.EG_PContent {
		if ec.Hyperlink != nil {
			return d.hyperlinkToMarkdown(ec.Hyperlink), nil
		}
		for _, rc := range ec.EG_ContentRunContent {
			if rc.R != nil {
				// Check for drawings in this run
				runText, runImages := d.runToMarkdownWithImageURLs(rc.R, imageDir, baseURL, imageCounter)
				para.WriteString(runText)
				for _, img := range runImages {
					para.WriteString(img)
				}
			}
		}
	}

	text := strings.TrimSpace(para.String())
	if text == "" {
		return "", nil
	}

	// Check for numbering (lists)
	if p.PPr != nil && p.PPr.NumPr != nil {
		if p.PPr.NumPr.NumId != nil && p.PPr.NumPr.NumId.ValAttr > 0 {
			// This is a numbered list item
			return fmt.Sprintf("%d. %s\n", p.PPr.NumPr.NumId.ValAttr, text), nil
		}
	}

	// Check for indentation (blockquotes)
	isBlockquote := false
	if p.PPr != nil && p.PPr.Ind != nil {
		if (p.PPr.Ind.LeftAttr != nil && *p.PPr.Ind.LeftAttr.Int64 > 360) ||
			(p.PPr.Ind.StartAttr != nil && *p.PPr.Ind.StartAttr.Int64 > 360) { // 360 twips = 0.25 inches
			isBlockquote = true
		}
	}

	switch style {
	case "Heading1":
		return "# " + text + "\n\n", nil
	case "Heading2":
		return "## " + text + "\n\n", nil
	case "Heading3":
		return "### " + text + "\n\n", nil
	case "Heading4":
		return "#### " + text + "\n\n", nil
	case "Heading5":
		return "##### " + text + "\n\n", nil
	case "Heading6":
		return "###### " + text + "\n\n", nil
	default:
		if isBlockquote {
			return "> " + text + "\n\n", nil
		}
		return text + "\n\n", nil
	}
}

func (d *Document) paragraphToMarkdownWithImages(p *wml.CT_P, imageDir string, imageCounter *int) (string, error) {
	var para strings.Builder
	style := ""
	if p.PPr != nil && p.PPr.PStyle != nil {
		style = p.PPr.PStyle.ValAttr
	}

	// Check for hyperlinks and regular content
	for _, ec := range p.EG_PContent {
		if ec.Hyperlink != nil {
			return d.hyperlinkToMarkdown(ec.Hyperlink), nil
		}
		for _, rc := range ec.EG_ContentRunContent {
			if rc.R != nil {
				// Check for drawings in this run
				runText, runImages := d.runToMarkdownWithImages(rc.R, imageDir, imageCounter)
				para.WriteString(runText)
				for _, img := range runImages {
					para.WriteString(img)
				}
			}
		}
	}

	text := strings.TrimSpace(para.String())
	if text == "" {
		return "", nil
	}

	// Check for numbering (lists)
	if p.PPr != nil && p.PPr.NumPr != nil {
		if p.PPr.NumPr.NumId != nil && p.PPr.NumPr.NumId.ValAttr > 0 {
			// This is a numbered list item
			return fmt.Sprintf("%d. %s\n", p.PPr.NumPr.NumId.ValAttr, text), nil
		}
	}

	// Check for indentation (blockquotes)
	isBlockquote := false
	if p.PPr != nil && p.PPr.Ind != nil {
		if (p.PPr.Ind.LeftAttr != nil && *p.PPr.Ind.LeftAttr.Int64 > 360) ||
			(p.PPr.Ind.StartAttr != nil && *p.PPr.Ind.StartAttr.Int64 > 360) { // 360 twips = 0.25 inches
			isBlockquote = true
		}
	}

	switch style {
	case "Heading1":
		return "# " + text + "\n\n", nil
	case "Heading2":
		return "## " + text + "\n\n", nil
	case "Heading3":
		return "### " + text + "\n\n", nil
	case "Heading4":
		return "#### " + text + "\n\n", nil
	case "Heading5":
		return "##### " + text + "\n\n", nil
	case "Heading6":
		return "###### " + text + "\n\n", nil
	case "Code":
		return "```\n" + text + "\n```\n\n", nil
	default:
		if isBlockquote {
			return "> " + text + "\n\n", nil
		}
		// Check if all text is monospace (code span)
		if d.isMonospaceParagraph(p) {
			return "`" + text + "`\n\n", nil
		}
		return text + "\n\n", nil
	}
}

func (d *Document) extractInlineDrawingWithURL(drawing *wml.CT_Drawing, imageDir string, baseURL string, imageCounter *int) (string, error) {
	// Handle inline drawings
	for _, inline := range drawing.Inline {
		imageRef, found := d.getImageFromInlineDrawing(inline)
		if !found {
			continue
		}

		// Extract and save the image
		imagePath, altText, err := d.saveImageToDir(imageRef, imageDir, imageCounter)
		if err != nil {
			return "", fmt.Errorf("failed to save image: %w", err)
		}

		// Generate URL for the fileserver (images are saved to frontend/public/images/)
		imageURL := fmt.Sprintf("%s/images/%s", baseURL, imagePath)

		// Generate markdown image syntax with URL
		return fmt.Sprintf("![%s](%s)", altText, imageURL), nil
	}

	// Handle anchored drawings (floating images)
	for _, anchor := range drawing.Anchor {
		imageRef, found := d.getImageFromAnchoredDrawing(anchor)
		if !found {
			continue
		}

		// Extract and save the image
		imagePath, altText, err := d.saveImageToDir(imageRef, imageDir, imageCounter)
		if err != nil {
			return "", fmt.Errorf("failed to save image: %w", err)
		}

		// Generate URL for the fileserver (images are saved to frontend/public/images/)
		imageURL := fmt.Sprintf("%s/images/%s", baseURL, imagePath)

		// Generate markdown image syntax with URL
		return fmt.Sprintf("![%s](%s)", altText, imageURL), nil
	}

	return "", nil
}

func (d *Document) extractInlineDrawing(drawing *wml.CT_Drawing, imageDir string, imageCounter *int) (string, error) {
	// Handle inline drawings
	for _, inline := range drawing.Inline {
		imageRef, found := d.getImageFromInlineDrawing(inline)
		if !found {
			continue
		}

		// Extract and save the image
		imagePath, altText, err := d.saveImageToDir(imageRef, imageDir, imageCounter)
		if err != nil {
			return "", fmt.Errorf("failed to save image: %w", err)
		}

		// Generate markdown image syntax
		return fmt.Sprintf("![%s](%s)", altText, imagePath), nil
	}

	// Handle anchored drawings (floating images)
	for _, anchor := range drawing.Anchor {
		imageRef, found := d.getImageFromAnchoredDrawing(anchor)
		if !found {
			continue
		}

		// Extract and save the image
		imagePath, altText, err := d.saveImageToDir(imageRef, imageDir, imageCounter)
		if err != nil {
			return "", fmt.Errorf("failed to save image: %w", err)
		}

		// Generate markdown image syntax
		return fmt.Sprintf("![%s](%s)", altText, imagePath), nil
	}

	return "", nil
}

func (d *Document) getImageFromInlineDrawing(inline *wml.WdInline) (common.ImageRef, bool) {
	if inline.Graphic == nil || inline.Graphic.GraphicData == nil {
		return common.ImageRef{}, false
	}

	for _, any := range inline.Graphic.GraphicData.Any {
		if pic, ok := any.(*pic.Pic); ok {
			if pic.BlipFill != nil && pic.BlipFill.Blip != nil && pic.BlipFill.Blip.EmbedAttr != nil {
				return d.GetImageByRelID(*pic.BlipFill.Blip.EmbedAttr)
			}
		}
	}
	return common.ImageRef{}, false
}

func (d *Document) getImageFromAnchoredDrawing(anchor *wml.WdAnchor) (common.ImageRef, bool) {
	if anchor.Graphic == nil || anchor.Graphic.GraphicData == nil {
		return common.ImageRef{}, false
	}

	for _, any := range anchor.Graphic.GraphicData.Any {
		if pic, ok := any.(*pic.Pic); ok {
			if pic.BlipFill != nil && pic.BlipFill.Blip != nil && pic.BlipFill.Blip.EmbedAttr != nil {
				return d.GetImageByRelID(*pic.BlipFill.Blip.EmbedAttr)
			}
		}
	}
	return common.ImageRef{}, false
}

func (d *Document) saveImageToDir(imageRef common.ImageRef, imageDir string, imageCounter *int) (string, string, error) {
	*imageCounter++
	imageName := fmt.Sprintf("image%d.%s", *imageCounter, imageRef.Format())
	imagePath := filepath.Join(imageDir, imageName)

	// Copy the image from its current location to the target directory
	sourcePath := imageRef.Path()
	if sourcePath == "" {
		return "", "", fmt.Errorf("image has no source path")
	}

	// Read the image data
	imageData, err := ioutil.ReadFile(sourcePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read image data: %w", err)
	}

	// Write to the target directory
	err = ioutil.WriteFile(imagePath, imageData, 0644)
	if err != nil {
		return "", "", fmt.Errorf("failed to write image to directory: %w", err)
	}

	// Get alt text (use filename if no description available)
	altText := imageName
	if imageRef.RelID() != "" {
		// Try to get a better description from relationships
		for _, rel := range d.docRels.Relationships() {
			if rel.ID() == imageRef.RelID() && rel.Target() != "" {
				// Use the target filename as alt text
				if baseName := filepath.Base(rel.Target()); baseName != "" {
					altText = strings.TrimSuffix(baseName, filepath.Ext(baseName))
				}
				break
			}
		}
	}

	return imageName, altText, nil
}

func (d *Document) hyperlinkToMarkdown(h *wml.CT_Hyperlink) string {
	var linkText strings.Builder
	for _, rc := range h.EG_ContentRunContent {
		if rc.R != nil {
			linkText.WriteString(d.runToMarkdown(rc.R))
		}
	}
	text := strings.TrimSpace(linkText.String())
	if text == "" {
		text = "link"
	}

	// Get the URL from relationships
	url := ""
	if h.IdAttr != nil {
		for _, rel := range d.docRels.Relationships() {
			if rel.ID() == *h.IdAttr {
				url = rel.Target()
				break
			}
		}
	}

	if url == "" {
		return text
	}

	return fmt.Sprintf("[%s](%s)", text, url)
}

func (d *Document) isMonospaceParagraph(p *wml.CT_P) bool {
	for _, ec := range p.EG_PContent {
		for _, rc := range ec.EG_ContentRunContent {
			if rc.R != nil && rc.R.RPr != nil && rc.R.RPr.RFonts != nil {
				if rc.R.RPr.RFonts.AsciiAttr != nil && strings.Contains(strings.ToLower(*rc.R.RPr.RFonts.AsciiAttr), "mono") {
					return true
				}
			}
		}
	}
	return false
}

func (d *Document) paragraphToMarkdown(p *wml.CT_P) string {
	var para strings.Builder
	style := ""
	if p.PPr != nil && p.PPr.PStyle != nil {
		style = p.PPr.PStyle.ValAttr
	}

	// Check for hyperlinks and regular content
	for _, ec := range p.EG_PContent {
		if ec.Hyperlink != nil {
			return d.hyperlinkToMarkdown(ec.Hyperlink)
		}
		for _, rc := range ec.EG_ContentRunContent {
			if rc.R != nil {
				para.WriteString(d.runToMarkdown(rc.R))
			}
		}
	}

	text := strings.TrimSpace(para.String())
	if text == "" {
		return ""
	}

	// Check for numbering (lists)
	if p.PPr != nil && p.PPr.NumPr != nil {
		if p.PPr.NumPr.NumId != nil && p.PPr.NumPr.NumId.ValAttr > 0 {
			// This is a numbered list item
			return fmt.Sprintf("%d. %s\n", p.PPr.NumPr.NumId.ValAttr, text)
		}
	}

	// Check for indentation (blockquotes)
	isBlockquote := false
	if p.PPr != nil && p.PPr.Ind != nil {
		if (p.PPr.Ind.LeftAttr != nil && *p.PPr.Ind.LeftAttr.Int64 > 360) ||
			(p.PPr.Ind.StartAttr != nil && *p.PPr.Ind.StartAttr.Int64 > 360) { // 360 twips = 0.25 inches
			isBlockquote = true
		}
	}

	switch style {
	case "Heading1":
		return "# " + text + "\n\n"
	case "Heading2":
		return "## " + text + "\n\n"
	case "Heading3":
		return "### " + text + "\n\n"
	case "Heading4":
		return "#### " + text + "\n\n"
	case "Heading5":
		return "##### " + text + "\n\n"
	case "Heading6":
		return "###### " + text + "\n\n"
	case "Code":
		return "```\n" + text + "\n```\n\n"
	default:
		if isBlockquote {
			return "> " + text + "\n\n"
		}
		// Check if all text is monospace (code span)
		if d.isMonospaceParagraph(p) {
			return "`" + text + "`\n\n"
		}
		return text + "\n\n"
	}
}

func (d *Document) runToMarkdownWithImageURLs(r *wml.CT_R, imageDir string, baseURL string, imageCounter *int) (string, []string) {
	var run strings.Builder
	var images []string

	for _, ic := range r.EG_RunInnerContent {
		if ic.Drawing != nil {
			// Handle drawing (image)
			imageMarkdown, err := d.extractInlineDrawingWithURL(ic.Drawing, imageDir, baseURL, imageCounter)
			if err != nil {
				gooxml.Log("failed to extract drawing: %s", err)
				continue
			}
			if imageMarkdown != "" {
				images = append(images, imageMarkdown)
			}
		} else if ic.T != nil {
			text := ic.T.Content
			if r.RPr != nil {
				// Handle strikethrough
				if r.RPr.Strike != nil || r.RPr.Dstrike != nil {
					text = "~~" + text + "~~"
				}
				// Handle bold
				if r.RPr.B != nil {
					text = "**" + text + "**"
				}
				// Handle italic
				if r.RPr.I != nil {
					text = "*" + text + "*"
				}
				// Handle code (monospace)
				if r.RPr.RFonts != nil && (r.RPr.RFonts.AsciiAttr != nil && strings.Contains(strings.ToLower(*r.RPr.RFonts.AsciiAttr), "mono")) {
					text = "`" + text + "`"
				}
			}
			run.WriteString(text)
		}
	}
	return run.String(), images
}

func (d *Document) runToMarkdownWithImages(r *wml.CT_R, imageDir string, imageCounter *int) (string, []string) {
	var run strings.Builder
	var images []string

	for _, ic := range r.EG_RunInnerContent {
		if ic.Drawing != nil {
			// Handle drawing (image)
			imageMarkdown, err := d.extractInlineDrawing(ic.Drawing, imageDir, imageCounter)
			if err != nil {
				gooxml.Log("failed to extract drawing: %s", err)
				continue
			}
			if imageMarkdown != "" {
				images = append(images, imageMarkdown)
			}
		} else if ic.T != nil {
			text := ic.T.Content
			if r.RPr != nil {
				// Handle strikethrough
				if r.RPr.Strike != nil || r.RPr.Dstrike != nil {
					text = "~~" + text + "~~"
				}
				// Handle bold
				if r.RPr.B != nil {
					text = "**" + text + "**"
				}
				// Handle italic
				if r.RPr.I != nil {
					text = "*" + text + "*"
				}
				// Handle code (monospace)
				if r.RPr.RFonts != nil && (r.RPr.RFonts.AsciiAttr != nil && strings.Contains(strings.ToLower(*r.RPr.RFonts.AsciiAttr), "mono")) {
					text = "`" + text + "`"
				}
			}
			run.WriteString(text)
		}
	}
	return run.String(), images
}

func (d *Document) runToMarkdown(r *wml.CT_R) string {
	text, _ := d.runToMarkdownWithImages(r, "", nil)
	return text
}

func (d *Document) tableToMarkdown(tbl *wml.CT_Tbl) string {
	var md strings.Builder
	rows := [][]string{}
	for _, rc := range tbl.EG_ContentRowContent {
		for _, tr := range rc.Tr {
			var row []string
			for _, ecc := range tr.EG_ContentCellContent {
				for _, tc := range ecc.Tc {
					var cell strings.Builder
					for _, ble := range tc.EG_BlockLevelElts {
						for _, cbc := range ble.EG_ContentBlockContent {
							for _, p := range cbc.P {
								cell.WriteString(d.paragraphToMarkdown(p))
							}
						}
					}
					row = append(row, strings.TrimSpace(strings.ReplaceAll(cell.String(), "\n\n", " ")))
				}
			}
			if len(row) > 0 {
				rows = append(rows, row)
			}
		}
	}

	if len(rows) == 0 {
		return ""
	}

	// header row
	md.WriteString("| " + strings.Join(rows[0], " | ") + " |\n")

	// separator
	seps := make([]string, len(rows[0]))
	for i := range seps {
		seps[i] = "---"
	}
	md.WriteString("| " + strings.Join(seps, " | ") + " |\n")

	// data rows
	for _, row := range rows[1:] {
		md.WriteString("| " + strings.Join(row, " | ") + " |\n")
	}

	return md.String()
}

// Footers returns the footers defined in the document.
func (d *Document) Footers() []Footer {
	ret := []Footer{}
	for _, f := range d.footers {
		ret = append(ret, Footer{d, f})
	}
	return ret
}

// AddFooter creates a Footer associated with the document, but doesn't add it
// to the document for display.
func (d *Document) AddFooter() Footer {
	ftr := wml.NewFtr()
	d.footers = append(d.footers, ftr)
	path := fmt.Sprintf("footer%d.xml", len(d.footers))
	d.docRels.AddRelationship(path, gooxml.FooterType)
	d.ContentTypes.AddOverride("/word/"+path, "application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml")
	return Footer{d, ftr}
}

// BodySection returns the default body section used for all preceding
// paragraphs until the previous Section. If there is no previous sections, the
// body section applies to the entire document.
func (d *Document) BodySection() Section {
	if d.x.Body.SectPr == nil {
		d.x.Body.SectPr = wml.NewCT_SectPr()
	}
	return Section{d, d.x.Body.SectPr}
}

// Save writes the document to an io.Writer in the Zip package format.
func (d *Document) Save(w io.Writer) error {
	if err := d.x.Validate(); err != nil {
		gooxml.Log("validation error in document: %s", err)
	}
	dt := gooxml.DocTypeDocument

	z := zip.NewWriter(w)
	defer z.Close()
	if err := zippkg.MarshalXML(z, gooxml.BaseRelsFilename, d.Rels.X()); err != nil {
		return err
	}
	if err := zippkg.MarshalXMLByType(z, dt, gooxml.ExtendedPropertiesType, d.AppProperties.X()); err != nil {
		return err
	}
	if err := zippkg.MarshalXMLByType(z, dt, gooxml.CorePropertiesType, d.CoreProperties.X()); err != nil {
		return err
	}
	if d.Thumbnail != nil {
		tn, err := z.Create("docProps/thumbnail.jpeg")
		if err != nil {
			return err
		}
		if err := jpeg.Encode(tn, d.Thumbnail, nil); err != nil {
			return err
		}
	}
	if err := zippkg.MarshalXMLByType(z, dt, gooxml.SettingsType, d.Settings.X()); err != nil {
		return err
	}
	documentFn := gooxml.AbsoluteFilename(dt, gooxml.OfficeDocumentType, 0)
	if err := zippkg.MarshalXML(z, documentFn, d.x); err != nil {
		return err
	}
	if err := zippkg.MarshalXML(z, zippkg.RelationsPathFor(documentFn), d.docRels.X()); err != nil {
		return err
	}

	if d.Numbering.X() != nil {
		if err := zippkg.MarshalXMLByType(z, dt, gooxml.NumberingType, d.Numbering.X()); err != nil {
			return err
		}
	}
	if err := zippkg.MarshalXMLByType(z, dt, gooxml.StylesType, d.Styles.X()); err != nil {
		return err
	}

	if d.webSettings != nil {
		if err := zippkg.MarshalXMLByType(z, dt, gooxml.WebSettingsType, d.webSettings); err != nil {
			return err
		}
	}
	if d.fontTable != nil {
		if err := zippkg.MarshalXMLByType(z, dt, gooxml.FontTableType, d.fontTable); err != nil {
			return err
		}
	}
	if d.endNotes != nil {
		if err := zippkg.MarshalXMLByType(z, dt, gooxml.EndNotesType, d.endNotes); err != nil {
			return err
		}
	}
	if d.footNotes != nil {
		if err := zippkg.MarshalXMLByType(z, dt, gooxml.FootNotesType, d.footNotes); err != nil {
			return err
		}
	}
	for i, thm := range d.themes {
		if err := zippkg.MarshalXMLByTypeIndex(z, dt, gooxml.ThemeType, i+1, thm); err != nil {
			return err
		}
	}
	for i, hdr := range d.headers {
		fn := gooxml.AbsoluteFilename(dt, gooxml.HeaderType, i+1)
		if err := zippkg.MarshalXML(z, fn, hdr); err != nil {
			return err
		}
		if !d.hdrRels[i].IsEmpty() {
			zippkg.MarshalXML(z, zippkg.RelationsPathFor(fn), d.hdrRels[i].X())
		}
	}
	for i, ftr := range d.footers {
		if err := zippkg.MarshalXMLByTypeIndex(z, dt, gooxml.FooterType, i+1, ftr); err != nil {
			return err
		}
	}

	for i, img := range d.Images {
		fn := fmt.Sprintf("word/media/image%d.png", i+1)
		if img.Path() != "" {
			if err := zippkg.AddFileFromDisk(z, fn, img.Path()); err != nil {
				return err
			}
		} else {
			gooxml.Log("unsupported image source: %+v", img)
		}
	}

	if err := zippkg.MarshalXML(z, gooxml.ContentTypesFilename, d.ContentTypes.X()); err != nil {
		return err
	}
	if err := d.WriteExtraFiles(z); err != nil {
		return err
	}
	return z.Close()
}

// AddTable adds a new table to the document body.
func (d *Document) AddTable() Table {
	elts := wml.NewEG_BlockLevelElts()
	d.x.Body.EG_BlockLevelElts = append(d.x.Body.EG_BlockLevelElts, elts)
	c := wml.NewEG_ContentBlockContent()
	elts.EG_ContentBlockContent = append(elts.EG_ContentBlockContent, c)
	tbl := wml.NewCT_Tbl()
	c.Tbl = append(c.Tbl, tbl)
	return Table{d, tbl}
}

// AddParagraph adds a new paragraph to the document body.
func (d *Document) AddParagraph() Paragraph {
	elts := wml.NewEG_BlockLevelElts()
	d.x.Body.EG_BlockLevelElts = append(d.x.Body.EG_BlockLevelElts, elts)
	c := wml.NewEG_ContentBlockContent()
	elts.EG_ContentBlockContent = append(elts.EG_ContentBlockContent, c)
	p := wml.NewCT_P()
	c.P = append(c.P, p)
	return Paragraph{d, p}
}

// RemoveParagraph removes a paragraph from a document.
func (d *Document) RemoveParagraph(p Paragraph) {
	if d.x.Body == nil {
		return
	}

	for _, ble := range d.x.Body.EG_BlockLevelElts {
		for _, c := range ble.EG_ContentBlockContent {
			for i, pa := range c.P {
				// do we need to remove this paragraph
				if pa == p.x {
					copy(c.P[i:], c.P[i+1:])
					c.P = c.P[0 : len(c.P)-1]
					return
				}
			}

			if c.Sdt != nil && c.Sdt.SdtContent != nil && c.Sdt.SdtContent.P != nil {
				for i, pa := range c.Sdt.SdtContent.P {
					if pa == p.x {
						copy(c.P[i:], c.P[i+1:])
						c.P = c.P[0 : len(c.P)-1]
						return
					}
				}
			}
		}
	}
}

// StructuredDocumentTags returns the structured document tags in the document
// which are commonly used in document templates.
func (d *Document) StructuredDocumentTags() []StructuredDocumentTag {
	ret := []StructuredDocumentTag{}
	for _, ble := range d.x.Body.EG_BlockLevelElts {
		for _, c := range ble.EG_ContentBlockContent {
			if c.Sdt != nil {
				ret = append(ret, StructuredDocumentTag{d, c.Sdt})
			}
		}
	}
	return ret
}

// Paragraphs returns all of the paragraphs in the document body.
func (d *Document) Paragraphs() []Paragraph {
	ret := []Paragraph{}
	if d.x.Body == nil {
		return nil
	}
	for _, ble := range d.x.Body.EG_BlockLevelElts {
		for _, c := range ble.EG_ContentBlockContent {
			for _, p := range c.P {
				ret = append(ret, Paragraph{d, p})
			}
		}
	}
	return ret
}

// SaveToFile writes the document out to a file.
func (d *Document) SaveToFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return d.Save(f)
}

// Open opens and reads a document from a file (.docx).
func Open(filename string) (*Document, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening %s: %s", filename, err)
	}
	defer f.Close()
	fi, err := os.Stat(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening %s: %s", filename, err)
	}
	_ = fi
	return Read(f, fi.Size())
}

// OpenTemplate opens a document, removing all content so it can be used as a
// template.  Since Word removes unused styles from a document upon save, to
// create a template in Word add a paragraph with every style of interest.  When
// opened with OpenTemplate the document's styles will be available but the
// content will be gone.
func OpenTemplate(filename string) (*Document, error) {
	d, err := Open(filename)
	if err != nil {
		return nil, err
	}
	d.x.Body = wml.NewCT_Body()
	return d, nil
}

// Read reads a document from an io.Reader.
func Read(r io.ReaderAt, size int64) (*Document, error) {
	doc := New()
	// numbering is not required
	doc.Numbering.x = nil

	td, err := ioutil.TempDir("", "gooxml-docx")
	if err != nil {
		return nil, err
	}
	doc.TmpPath = td

	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("parsing zip: %s", err)
	}

	files := []*zip.File{}
	files = append(files, zr.File...)

	decMap := zippkg.DecodeMap{}
	decMap.SetOnNewRelationshipFunc(doc.onNewRelationship)
	// we should discover all contents by starting with these two files
	decMap.AddTarget(gooxml.ContentTypesFilename, doc.ContentTypes.X(), "", 0)
	decMap.AddTarget(gooxml.BaseRelsFilename, doc.Rels.X(), "", 0)
	if err := decMap.Decode(files); err != nil {
		return nil, err
	}

	for _, f := range files {
		if f == nil {
			continue
		}
		if err := doc.AddExtraFileFromZip(f); err != nil {
			return nil, err
		}
	}
	return doc, nil
}

// Validate validates the structure and in cases where it't possible, the ranges
// of elements within a document. A validation error dones't mean that the
// document won't work in MS Word or LibreOffice, but it's worth checking into.
func (d *Document) Validate() error {
	if d == nil || d.x == nil {
		return errors.New("document not initialized correctly, nil base")
	}

	for _, v := range []func() error{d.validateTableCells, d.validateBookmarks} {
		if err := v(); err != nil {
			return err
		}
	}
	if err := d.x.Validate(); err != nil {
		return err
	}
	return nil
}

func (d *Document) validateBookmarks() error {
	bmnames := make(map[string]struct{})
	for _, bm := range d.Bookmarks() {
		if _, ok := bmnames[bm.Name()]; ok {
			return fmt.Errorf("duplicate bookmark %s found", bm.Name())
		}
		bmnames[bm.Name()] = struct{}{}
	}
	return nil
}
func (d *Document) validateTableCells() error {
	for _, elt := range d.x.Body.EG_BlockLevelElts {
		for _, c := range elt.EG_ContentBlockContent {
			for _, t := range c.Tbl {
				for _, rc := range t.EG_ContentRowContent {
					for _, row := range rc.Tr {
						hasCell := false
						for _, ecc := range row.EG_ContentCellContent {
							cellHasPara := false
							for _, cell := range ecc.Tc {
								hasCell = true
								for _, cellElt := range cell.EG_BlockLevelElts {
									for _, cellCont := range cellElt.EG_ContentBlockContent {
										if len(cellCont.P) > 0 {
											cellHasPara = true
											break
										}
									}
								}
							}
							if !cellHasPara {
								return errors.New("table cell must contain a paragraph")
							}
						}
						// OSX Word requires this and won't open the file otherwise
						if !hasCell {
							return errors.New("table row must contain a cell")
						}
					}
				}
			}
		}
	}
	return nil
}

// AddImage adds an image to the document package, returning a reference that
// can be used to add the image to a run and place it in the document contents.
func (d *Document) AddImage(i common.Image) (common.ImageRef, error) {
	r := common.MakeImageRef(i, &d.DocBase, d.docRels)
	if i.Path == "" {
		return r, errors.New("image must have a path")
	}

	if i.Format == "" {
		return r, errors.New("image must have a valid format")
	}
	if i.Size.X == 0 || i.Size.Y == 0 {
		return r, errors.New("image must have a valid size")
	}

	d.Images = append(d.Images, r)
	fn := fmt.Sprintf("media/image%d.%s", len(d.Images), i.Format)
	d.docRels.AddRelationship(fn, gooxml.ImageType)
	d.ContentTypes.EnsureDefault("png", "image/png")
	d.ContentTypes.EnsureDefault("jpeg", "image/jpeg")
	d.ContentTypes.EnsureDefault("jpg", "image/jpeg")
	d.ContentTypes.EnsureDefault("wmf", "image/x-wmf")
	d.ContentTypes.EnsureDefault(i.Format, "image/"+i.Format)
	return r, nil
}

// GetImageByRelID returns an ImageRef with the associated relation ID in the
// document.
func (d *Document) GetImageByRelID(relID string) (common.ImageRef, bool) {
	for _, img := range d.Images {
		if img.RelID() == relID {
			return img, true
		}
	}
	return common.ImageRef{}, false
}

// FormFields extracts all of the fields from a document.  They can then be
// manipulated via the methods on the field and the document saved.
func (d *Document) FormFields() []FormField {
	ret := []FormField{}
	for _, p := range d.Paragraphs() {
		runs := p.Runs()
		for i, r := range runs {
			for _, ic := range r.x.EG_RunInnerContent {
				// skip non form fields
				if ic.FldChar == nil || ic.FldChar.FfData == nil {
					continue
				}

				// found a begin form field
				if ic.FldChar.FldCharTypeAttr == wml.ST_FldCharTypeBegin {
					// ensure it has a name
					if len(ic.FldChar.FfData.Name) == 0 || ic.FldChar.FfData.Name[0].ValAttr == nil {
						continue
					}

					field := FormField{x: ic.FldChar.FfData}
					// for text input boxes, we need a pointer to where to set
					// the text as well
					if ic.FldChar.FfData.TextInput != nil {

						// ensure we always have at lest two IC's
						for j := i + 1; j < len(runs)-1; j++ {
							if len(runs[j].x.EG_RunInnerContent) == 0 {
								continue
							}
							ic := runs[j].x.EG_RunInnerContent[0]
							// look for the 'separate' field
							if ic.FldChar != nil && ic.FldChar.FldCharTypeAttr == wml.ST_FldCharTypeSeparate {
								if len(runs[j+1].x.EG_RunInnerContent) == 0 {
									continue
								}
								// the value should be the text in the next inner content that is not a field char
								if runs[j+1].x.EG_RunInnerContent[0].FldChar == nil {
									field.textIC = runs[j+1].x.EG_RunInnerContent[0]
								}
							}
						}
					}
					ret = append(ret, field)
				}
			}
		}
	}
	return ret
}

func (d *Document) onNewRelationship(decMap *zippkg.DecodeMap, target, typ string, files []*zip.File, rel *relationships.Relationship, src zippkg.Target) error {
	dt := gooxml.DocTypeDocument

	switch typ {
	case gooxml.OfficeDocumentType:
		d.x = wml.NewDocument()
		decMap.AddTarget(target, d.x, typ, 0)
		// look for the document relationships file as well
		decMap.AddTarget(zippkg.RelationsPathFor(target), d.docRels.X(), typ, 0)
		rel.TargetAttr = gooxml.RelativeFilename(dt, src.Typ, typ, 0)

	case gooxml.CorePropertiesType:
		decMap.AddTarget(target, d.CoreProperties.X(), typ, 0)
		rel.TargetAttr = gooxml.RelativeFilename(dt, src.Typ, typ, 0)

	case gooxml.ExtendedPropertiesType:
		decMap.AddTarget(target, d.AppProperties.X(), typ, 0)
		rel.TargetAttr = gooxml.RelativeFilename(dt, src.Typ, typ, 0)

	case gooxml.ThumbnailType:
		// read our thumbnail
		for i, f := range files {
			if f == nil {
				continue
			}
			if f.Name == target {
				rc, err := f.Open()
				if err != nil {
					return fmt.Errorf("error reading thumbnail: %s", err)
				}
				d.Thumbnail, _, err = image.Decode(rc)
				rc.Close()
				if err != nil {
					return fmt.Errorf("error decoding thumbnail: %s", err)
				}
				files[i] = nil
			}
		}

	case gooxml.SettingsType:
		decMap.AddTarget(target, d.Settings.X(), typ, 0)
		rel.TargetAttr = gooxml.RelativeFilename(dt, src.Typ, typ, 0)

	case gooxml.NumberingType:
		d.Numbering = NewNumbering()
		decMap.AddTarget(target, d.Numbering.X(), typ, 0)
		rel.TargetAttr = gooxml.RelativeFilename(dt, src.Typ, typ, 0)

	case gooxml.StylesType:
		d.Styles.Clear()
		decMap.AddTarget(target, d.Styles.X(), typ, 0)
		rel.TargetAttr = gooxml.RelativeFilename(dt, src.Typ, typ, 0)

	case gooxml.HeaderType:
		hdr := wml.NewHdr()
		decMap.AddTarget(target, hdr, typ, uint32(len(d.headers)))
		d.headers = append(d.headers, hdr)
		rel.TargetAttr = gooxml.RelativeFilename(dt, src.Typ, typ, len(d.headers))

		// look for header rels
		hdrRel := common.NewRelationships()
		decMap.AddTarget(zippkg.RelationsPathFor(target), hdrRel.X(), typ, 0)
		d.hdrRels = append(d.hdrRels, hdrRel)

	case gooxml.FooterType:
		ftr := wml.NewFtr()
		decMap.AddTarget(target, ftr, typ, uint32(len(d.footers)))
		d.footers = append(d.footers, ftr)
		rel.TargetAttr = gooxml.RelativeFilename(dt, src.Typ, typ, len(d.footers))

	case gooxml.ThemeType:
		thm := dml.NewTheme()
		decMap.AddTarget(target, thm, typ, uint32(len(d.themes)))
		d.themes = append(d.themes, thm)
		rel.TargetAttr = gooxml.RelativeFilename(dt, src.Typ, typ, len(d.themes))

	case gooxml.WebSettingsType:
		d.webSettings = wml.NewWebSettings()
		decMap.AddTarget(target, d.webSettings, typ, 0)
		rel.TargetAttr = gooxml.RelativeFilename(dt, src.Typ, typ, 0)

	case gooxml.FontTableType:
		d.fontTable = wml.NewFonts()
		decMap.AddTarget(target, d.fontTable, typ, 0)
		rel.TargetAttr = gooxml.RelativeFilename(dt, src.Typ, typ, 0)

	case gooxml.EndNotesType:
		d.endNotes = wml.NewEndnotes()
		decMap.AddTarget(target, d.endNotes, typ, 0)
		rel.TargetAttr = gooxml.RelativeFilename(dt, src.Typ, typ, 0)

	case gooxml.FootNotesType:
		d.footNotes = wml.NewFootnotes()
		decMap.AddTarget(target, d.footNotes, typ, 0)
		rel.TargetAttr = gooxml.RelativeFilename(dt, src.Typ, typ, 0)

	case gooxml.ImageType:
		for i, f := range files {
			if f == nil {
				continue
			}
			if f.Name == target {
				path, err := zippkg.ExtractToDiskTmp(f, d.TmpPath)
				if err != nil {
					return err
				}
				img, err := common.ImageFromFile(path)
				if err != nil {
					return err
				}
				iref := common.MakeImageRef(img, &d.DocBase, d.docRels)
				d.Images = append(d.Images, iref)
				files[i] = nil
			}
		}
		rel.TargetAttr = gooxml.RelativeFilename(dt, src.Typ, typ, len(d.Images))
	default:
		gooxml.Log("unsupported relationship type: %s tgt: %s", typ, target)
	}
	return nil
}

// InsertParagraphAfter adds a new empty paragraph after the relativeTo
// paragraph.
func (d *Document) InsertParagraphAfter(relativeTo Paragraph) Paragraph {
	return d.insertParagraph(relativeTo, false)
}

// InsertParagraphBefore adds a new empty paragraph before the relativeTo
// paragraph.
func (d *Document) InsertParagraphBefore(relativeTo Paragraph) Paragraph {
	return d.insertParagraph(relativeTo, true)
}

func (d *Document) insertParagraph(relativeTo Paragraph, before bool) Paragraph {
	if d.x.Body == nil {
		return d.AddParagraph()
	}

	for _, ble := range d.x.Body.EG_BlockLevelElts {
		for _, c := range ble.EG_ContentBlockContent {
			for i, p := range c.P {
				// foudn the paragraph
				if p == relativeTo.X() {
					p := wml.NewCT_P()
					c.P = append(c.P, nil)
					if before {
						copy(c.P[i+1:], c.P[i:])
						c.P[i] = p
					} else {
						copy(c.P[i+2:], c.P[i+1:])
						c.P[i+1] = p
					}
					return Paragraph{d, p}
				}
			}

			if c.Sdt != nil && c.Sdt.SdtContent != nil && c.Sdt.SdtContent.P != nil {
				for i, p := range c.Sdt.SdtContent.P {
					if p == relativeTo.X() {
						p := wml.NewCT_P()
						c.Sdt.SdtContent.P = append(c.Sdt.SdtContent.P, nil)
						if before {
							copy(c.Sdt.SdtContent.P[i+1:], c.Sdt.SdtContent.P[i:])
							c.Sdt.SdtContent.P[i] = p
						} else {
							copy(c.Sdt.SdtContent.P[i+2:], c.Sdt.SdtContent.P[i+1:])
							c.Sdt.SdtContent.P[i+1] = p
						}
						return Paragraph{d, p}
					}
				}
			}
		}
	}
	return d.AddParagraph()
}

// AddHyperlink adds a hyperlink to a document. Adding the hyperlink to a document
// and setting it on a cell is more efficient than setting hyperlinks directly
// on a cell.
func (d Document) AddHyperlink(url string) common.Hyperlink {
	return d.docRels.AddHyperlink(url)
}

// Bookmarks returns all of the bookmarks defined in the document.
func (d Document) Bookmarks() []Bookmark {
	if d.x.Body == nil {
		return nil
	}
	ret := []Bookmark{}
	for _, ble := range d.x.Body.EG_BlockLevelElts {
		for _, bc := range ble.EG_ContentBlockContent {
			// bookmarks within paragraphs
			for _, p := range bc.P {
				for _, ec := range p.EG_PContent {
					for _, ecr := range ec.EG_ContentRunContent {
						for _, re := range ecr.EG_RunLevelElts {
							for _, rm := range re.EG_RangeMarkupElements {
								if rm.BookmarkStart != nil {
									ret = append(ret, Bookmark{rm.BookmarkStart})
								}
							}
						}
					}
				}
			}
			for _, re := range bc.EG_RunLevelElts {
				for _, rm := range re.EG_RangeMarkupElements {
					if rm.BookmarkStart != nil {
						ret = append(ret, Bookmark{rm.BookmarkStart})
					}
				}
			}
		}
	}
	return ret
}
