package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yudaprama/gooxml/color"
	"github.com/yudaprama/gooxml/document"
	"github.com/yudaprama/gooxml/measurement"
	"github.com/yudaprama/gooxml/presentation"
	sharedTypes "github.com/yudaprama/gooxml/schema/soo/ofc/sharedTypes"
	"github.com/yudaprama/gooxml/schema/soo/sml"
	"github.com/yudaprama/gooxml/schema/soo/wml"
	"github.com/yudaprama/gooxml/spreadsheet"
)

// The types below mirror components/tool/officeedit so the same declarative
// JSON operation payload works against the ooxcli binary.

type RunSpec struct {
	Text          string `json:"text"`
	Bold          bool   `json:"bold,omitempty"`
	Italic        bool   `json:"italic,omitempty"`
	Size          int    `json:"size,omitempty"`
	Color         string `json:"color,omitempty"`
	Font          string `json:"font,omitempty"`
	Highlight     string `json:"highlight,omitempty"`
	Underline     bool   `json:"underline,omitempty"`
	Strikethrough bool   `json:"strikethrough,omitempty"`
	Superscript   bool   `json:"superscript,omitempty"`
	Subscript     bool   `json:"subscript,omitempty"`
}

type ParagraphSpec struct {
	Type string    `json:"type"`
	Runs []RunSpec `json:"runs"`
}

type CellSpec struct {
	Text string `json:"text"`
}

type RowSpec struct {
	Cells []CellSpec `json:"cells"`
}

type CellRowSpec struct {
	Values []any `json:"values"`
}

type CellSpecXlsx struct {
	Cell  string `json:"cell"`
	Value any    `json:"value"`
}

type SlideSpec struct {
	Title string   `json:"title,omitempty"`
	Body  []string `json:"body,omitempty"`
}

type EditOp struct {
	Type          string          `json:"type"`
	Find          string          `json:"find,omitempty"`
	Replace       string          `json:"replace,omitempty"`
	Paragraphs    []ParagraphSpec `json:"paragraphs,omitempty"`
	Rows          []RowSpec       `json:"rows,omitempty"`
	CellRows      []CellRowSpec   `json:"cell_rows,omitempty"`
	Cells         []CellSpecXlsx  `json:"cells,omitempty"`
	Slides        []SlideSpec     `json:"slides,omitempty"`
	Sheet         string          `json:"sheet,omitempty"`
	Alignment     string          `json:"alignment,omitempty"`
	SpacingBefore int             `json:"spacing_before,omitempty"`
	SpacingAfter  int             `json:"spacing_after,omitempty"`
	IndentLeft    int             `json:"indent_left,omitempty"`
	IndentRight   int             `json:"indent_right,omitempty"`
}

func cmdEdit(args []string) error {
	input := ""
	output := ""
	opsJSON := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--out":
			i++
			if i >= len(args) {
				return errors.New("--out requires a value")
			}
			output = args[i]
		case "--ops":
			i++
			if i >= len(args) {
				return errors.New("--ops requires a value")
			}
			opsJSON = args[i]
		default:
			if input == "" {
				input = args[i]
			}
		}
	}
	if input == "" {
		return errors.New("usage: ooxcli edit <input.docx|xlsx|pptx> [--out <output>] [--ops <json>]  (ops JSON read from stdin when --ops is omitted)")
	}

	var ops []EditOp
	if opsJSON == "" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read ops from stdin: %w", err)
		}
		opsJSON = string(data)
	}
	if strings.TrimSpace(opsJSON) == "" {
		return errors.New("no operations provided (empty --ops or stdin)")
	}
	if err := json.Unmarshal([]byte(opsJSON), &ops); err != nil {
		return fmt.Errorf("parse operations: %w", err)
	}
	if len(ops) == 0 {
		return errors.New("operations must not be empty")
	}

	target := input
	if output != "" {
		target = output
	}

	ext := fileExtEdit(input)
	switch ext {
	case "docx":
		return editDocx(input, target, ops)
	case "xlsx":
		return editXlsx(input, target, ops)
	case "pptx":
		return editPptx(input, target, ops)
	default:
		return fmt.Errorf("ooxcli edit supports .docx, .xlsx, .pptx (got .%s)", ext)
	}
}

func fileExtEdit(filename string) string {
	dot := strings.LastIndex(filename, ".")
	if dot < 0 {
		return ""
	}
	return strings.ToLower(filename[dot+1:])
}

// --- docx ---

func editDocx(input, output string, ops []EditOp) error {
	doc, err := document.Open(input)
	if err != nil {
		return fmt.Errorf("open docx: %w", err)
	}
	for _, op := range ops {
		switch op.Type {
		case "replace_text":
			if err := applyReplaceTextDocx(doc, op.Find, op.Replace); err != nil {
				return err
			}
		case "append_paragraphs":
			applyAppendParagraphsDocx(doc, op.Paragraphs)
		case "append_table":
			applyAppendTableDocx(doc, op.Rows)
		case "delete_paragraph":
			applyDeleteParagraphDocx(doc, op.Find)
		case "format_paragraph":
			applyFormatParagraphDocx(doc, op.Find, op)
		default:
			return fmt.Errorf("unknown docx operation %q", op.Type)
		}
	}
	return doc.SaveToFile(output)
}

func applyReplaceTextDocx(doc *document.Document, find, replace string) error {
	if find == "" {
		return nil
	}
	paras := doc.Paragraphs()
	for _, p := range doc.StructuredDocumentTags() {
		paras = append(paras, p.Paragraphs()...)
	}
	for _, tbl := range allTables(doc) {
		for _, p := range tableParagraphs(doc, tbl) {
			paras = append(paras, p)
		}
	}
	for _, para := range paras {
		replaceInParagraph(para, find, replace)
	}
	return nil
}

func allTables(doc *document.Document) []document.Table {
	var tables []document.Table
	if doc.X().Body == nil {
		return tables
	}
	for _, ble := range doc.X().Body.EG_BlockLevelElts {
		for _, cbc := range ble.EG_ContentBlockContent {
			for _, tbl := range cbc.Tbl {
				tables = append(tables, document.NewTable(doc, tbl))
			}
		}
	}
	return tables
}

func tableParagraphs(doc *document.Document, tbl document.Table) []document.Paragraph {
	var paras []document.Paragraph
	for _, rc := range tbl.X().EG_ContentRowContent {
		for _, tr := range rc.Tr {
			for _, ecc := range tr.EG_ContentCellContent {
				for _, tc := range ecc.Tc {
					for _, blk := range tc.EG_BlockLevelElts {
						for _, cbc := range blk.EG_ContentBlockContent {
							for _, p := range cbc.P {
								paras = append(paras, document.NewParagraph(doc, p))
							}
						}
					}
				}
			}
		}
	}
	return paras
}

func replaceInParagraph(para document.Paragraph, find, replace string) {
	runs := para.Runs()
	if len(runs) == 0 {
		return
	}
	for _, r := range runs {
		txt := r.Text()
		if strings.Contains(txt, find) {
			r.ClearContent()
			r.AddText(strings.ReplaceAll(txt, find, replace))
			return
		}
	}
	full := joinRuns(runs)
	if !strings.Contains(full, find) {
		return
	}
	newText := strings.ReplaceAll(full, find, replace)
	firstRun := runs[0]
	newRun := para.AddRun()
	copyRunFormatting(firstRun, newRun)
	newRun.AddText(newText)
	for _, r := range runs {
		para.RemoveRun(r)
	}
}

func copyRunFormatting(src, dst document.Run) {
	sp := src.Properties()
	dp := dst.Properties()
	if sp.IsBold() {
		dp.SetBold(true)
	}
	if sp.IsItalic() {
		dp.SetItalic(true)
	}
	fonts := sp.Fonts()
	if fonts.X() != nil {
		if fonts.X().AsciiAttr != nil {
			dp.SetFontFamily(*fonts.X().AsciiAttr)
		}
	}
}

func joinRuns(runs []document.Run) string {
	var b strings.Builder
	for _, r := range runs {
		b.WriteString(r.Text())
	}
	return b.String()
}

func applyAppendParagraphsDocx(doc *document.Document, specs []ParagraphSpec) {
	for _, spec := range specs {
		para := doc.AddParagraph()
		switch spec.Type {
		case "heading1":
			para.SetStyle("Heading1")
		case "heading2":
			para.SetStyle("Heading2")
		case "heading3":
			para.SetStyle("Heading3")
		case "title":
			para.SetStyle("Title")
		case "bullet":
			applyBulletStyle(doc, para)
		}
		for _, rs := range spec.Runs {
			run := para.AddRun()
			run.AddText(rs.Text)
			applyRunStyle(run, rs)
		}
	}
}

func applyBulletStyle(doc *document.Document, para document.Paragraph) {
	para.SetStyle("ListParagraph")
	num := doc.Numbering
	defs := num.Definitions()
	for _, d := range defs {
		for _, lvl := range d.Levels() {
			if lvl.X().NumFmt != nil && lvl.X().NumFmt.ValAttr == wml.ST_NumberFormatBullet {
				para.SetNumberingDefinition(d)
				return
			}
		}
	}
	nd := num.AddDefinition()
	lvl := nd.AddLevel()
	lvl.SetFormat(wml.ST_NumberFormatBullet)
	lvl.SetText("\u2022")
	lvl.SetAlignment(wml.ST_JcLeft)
	para.SetNumberingDefinition(nd)
}

func applyAppendTableDocx(doc *document.Document, rows []RowSpec) {
	tbl := doc.AddTable()
	tblProps := tbl.Properties()
	tblProps.SetWidthPercent(100)
	tblProps.Borders().SetAll(wml.ST_BorderSingle, color.RGB(0, 0, 0), 4*measurement.Point)
	for _, rowSpec := range rows {
		row := tbl.AddRow()
		for _, cellSpec := range rowSpec.Cells {
			cell := row.AddCell()
			para := cell.AddParagraph()
			run := para.AddRun()
			run.AddText(cellSpec.Text)
		}
	}
}

func applyDeleteParagraphDocx(doc *document.Document, find string) {
	if find == "" {
		return
	}
	var toDelete []document.Paragraph
	for _, p := range doc.Paragraphs() {
		if containsText(p, find) {
			toDelete = append(toDelete, p)
		}
	}
	for _, sdt := range doc.StructuredDocumentTags() {
		for _, p := range sdt.Paragraphs() {
			if containsText(p, find) {
				toDelete = append(toDelete, p)
			}
		}
	}
	for i := len(toDelete) - 1; i >= 0; i-- {
		doc.RemoveParagraph(toDelete[i])
	}
}

func containsText(para document.Paragraph, text string) bool {
	for _, r := range para.Runs() {
		if strings.Contains(r.Text(), text) {
			return true
		}
	}
	return false
}

func applyFormatParagraphDocx(doc *document.Document, find string, op EditOp) {
	if find == "" {
		return
	}
	paras := doc.Paragraphs()
	for _, p := range doc.StructuredDocumentTags() {
		paras = append(paras, p.Paragraphs()...)
	}
	for _, para := range paras {
		if !containsText(para, find) {
			continue
		}
		props := para.Properties()
		switch op.Alignment {
		case "left":
			props.SetAlignment(wml.ST_JcLeft)
		case "center":
			props.SetAlignment(wml.ST_JcCenter)
		case "right":
			props.SetAlignment(wml.ST_JcRight)
		case "justify":
			props.SetAlignment(wml.ST_JcBoth)
		}
		if op.SpacingBefore > 0 || op.SpacingAfter > 0 {
			props.SetSpacing(
				measurement.Distance(op.SpacingBefore)*measurement.Point,
				measurement.Distance(op.SpacingAfter)*measurement.Point,
			)
		}
		if op.IndentLeft > 0 {
			props.SetStartIndent(measurement.Distance(op.IndentLeft) * measurement.Point)
		}
		if op.IndentRight > 0 {
			props.SetEndIndent(measurement.Distance(op.IndentRight) * measurement.Point)
		}
	}
}

func applyRunStyle(run document.Run, rs RunSpec) {
	props := run.Properties()
	if rs.Bold {
		props.SetBold(true)
	}
	if rs.Italic {
		props.SetItalic(true)
	}
	if rs.Size > 0 {
		props.SetSize(measurement.Distance(rs.Size) * measurement.Point)
	}
	if rs.Color != "" {
		if c, err := parseHexColor(rs.Color); err == nil {
			props.SetColor(c)
		}
	}
	if rs.Font != "" {
		props.SetFontFamily(rs.Font)
	}
	if rs.Highlight != "" && rs.Highlight != "none" {
		switch rs.Highlight {
		case "yellow":
			props.SetHighlight(wml.ST_HighlightColorYellow)
		case "green":
			props.SetHighlight(wml.ST_HighlightColorGreen)
		case "cyan":
			props.SetHighlight(wml.ST_HighlightColorCyan)
		case "magenta":
			props.SetHighlight(wml.ST_HighlightColorMagenta)
		case "blue":
			props.SetHighlight(wml.ST_HighlightColorBlue)
		case "red":
			props.SetHighlight(wml.ST_HighlightColorRed)
		case "darkBlue":
			props.SetHighlight(wml.ST_HighlightColorDarkBlue)
		case "darkCyan":
			props.SetHighlight(wml.ST_HighlightColorDarkCyan)
		case "darkGreen":
			props.SetHighlight(wml.ST_HighlightColorDarkGreen)
		case "darkMagenta":
			props.SetHighlight(wml.ST_HighlightColorDarkMagenta)
		case "darkRed":
			props.SetHighlight(wml.ST_HighlightColorDarkRed)
		case "darkYellow":
			props.SetHighlight(wml.ST_HighlightColorDarkYellow)
		case "darkGray":
			props.SetHighlight(wml.ST_HighlightColorDarkGray)
		case "lightGray":
			props.SetHighlight(wml.ST_HighlightColorLightGray)
		case "black":
			props.SetHighlight(wml.ST_HighlightColorBlack)
		case "white":
			props.SetHighlight(wml.ST_HighlightColorWhite)
		}
	}
	if rs.Underline {
		props.SetUnderline(wml.ST_UnderlineSingle, color.Color{})
	}
	if rs.Strikethrough {
		props.SetStrikeThrough(true)
	}
	if rs.Superscript {
		props.SetVerticalAlignment(sharedTypes.ST_VerticalAlignRunSuperscript)
	}
	if rs.Subscript {
		props.SetVerticalAlignment(sharedTypes.ST_VerticalAlignRunSubscript)
	}
}

func parseHexColor(s string) (color.Color, error) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return color.Color{}, fmt.Errorf("invalid hex color: %s", s)
	}
	var r, g, b uint8
	if _, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b); err != nil {
		return color.Color{}, fmt.Errorf("invalid hex color: %s: %w", s, err)
	}
	return color.RGB(r, g, b), nil
}

// --- xlsx ---

func editXlsx(input, output string, ops []EditOp) error {
	wb, err := spreadsheet.Open(input)
	if err != nil {
		return fmt.Errorf("open xlsx: %w", err)
	}
	for _, op := range ops {
		switch op.Type {
		case "replace_text":
			applyReplaceTextXlsx(wb, op.Find, op.Replace)
		case "append_rows":
			if err := applyAppendRowsXlsx(wb, op.Sheet, op.CellRows); err != nil {
				return err
			}
		case "set_cell":
			if err := applySetCellXlsx(wb, op.Sheet, op.Cells); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown xlsx operation %q", op.Type)
		}
	}
	for _, sheet := range wb.Sheets() {
		s := sheet.X()
		if s != nil && s.Dimension == nil {
			s.Dimension = sml.NewCT_SheetDimension()
		}
	}
	return wb.SaveToFile(output)
}

func applyReplaceTextXlsx(wb *spreadsheet.Workbook, find, replace string) {
	if find == "" {
		return
	}
	for _, sheet := range wb.Sheets() {
		for _, row := range sheet.Rows() {
			for _, cell := range row.Cells() {
				val := cell.GetFormattedValue()
				if strings.Contains(val, find) {
					newVal := strings.ReplaceAll(val, find, replace)
					if f, err := parseFloat(newVal); err == nil {
						cell.SetNumber(f)
					} else if newVal == "true" {
						cell.SetBool(true)
					} else if newVal == "false" {
						cell.SetBool(false)
					} else {
						cell.SetInlineString(newVal)
					}
				}
			}
		}
	}
}

func applyAppendRowsXlsx(wb *spreadsheet.Workbook, sheetName string, rows []CellRowSpec) error {
	sheet, err := wb.GetSheet(sheetName)
	if err != nil {
		sheets := wb.Sheets()
		if len(sheets) == 0 {
			sheet = wb.AddSheet()
		} else {
			sheet = sheets[0]
		}
	}
	for _, rowSpec := range rows {
		row := sheet.AddRow()
		for _, val := range rowSpec.Values {
			cell := row.AddCell()
			setCellValue(cell, val)
		}
	}
	return nil
}

func applySetCellXlsx(wb *spreadsheet.Workbook, sheetName string, cells []CellSpecXlsx) error {
	sheet, err := wb.GetSheet(sheetName)
	if err != nil {
		sheets := wb.Sheets()
		if len(sheets) == 0 {
			return fmt.Errorf("no sheets in workbook")
		}
		sheet = sheets[0]
	}
	for _, c := range cells {
		if c.Cell == "" {
			continue
		}
		colStr, rowIdx := parseCellRef(c.Cell)
		if rowIdx == 0 {
			return fmt.Errorf("invalid cell reference %q", c.Cell)
		}
		var row spreadsheet.Row
		found := false
		for _, r := range sheet.Rows() {
			if r.RowNumber() == rowIdx {
				row = r
				found = true
				break
			}
		}
		if !found {
			row = sheet.AddNumberedRow(rowIdx)
		}
		cell := row.Cell(colStr)
		if cell.X() == nil {
			cell = row.AddNamedCell(colStr)
		}
		setCellValue(cell, c.Value)
	}
	return nil
}

func parseCellRef(ref string) (string, uint32) {
	col := ""
	row := uint32(0)
	for _, ch := range ref {
		if ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z' {
			col += string(ch)
		} else if ch >= '0' && ch <= '9' {
			row = row*10 + uint32(ch-'0')
		}
	}
	return strings.ToUpper(col), row
}

func setCellValue(cell spreadsheet.Cell, val any) {
	if val == nil {
		return
	}
	switch v := val.(type) {
	case float64:
		cell.SetNumber(v)
	case bool:
		cell.SetBool(v)
	case string:
		if f, err := parseFloat(v); err == nil {
			cell.SetNumber(f)
		} else {
			cell.SetInlineString(v)
		}
	}
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

// --- pptx ---

func editPptx(input, output string, ops []EditOp) error {
	pres, err := presentation.Open(input)
	if err != nil {
		return fmt.Errorf("open pptx: %w", err)
	}
	for _, op := range ops {
		switch op.Type {
		case "replace_text":
			applyReplaceTextPptx(pres, op.Find, op.Replace)
		case "append_slides":
			applyAppendSlidesPptx(pres, op.Slides)
		case "remove_slide":
			applyRemoveSlidePptx(pres, op.Find)
		default:
			return fmt.Errorf("unknown pptx operation %q", op.Type)
		}
	}
	return pres.SaveToFile(output)
}

func applyReplaceTextPptx(pres *presentation.Presentation, find, replace string) {
	if find == "" {
		return
	}
	for _, slide := range pres.Slides() {
		replaceInSlideContent(slide, find, replace)
	}
}

func replaceInSlideContent(slide presentation.Slide, find, replace string) {
	for _, choice := range slide.X().CSld.SpTree.Choice {
		for _, sp := range choice.Sp {
			if sp.TxBody == nil {
				continue
			}
			for _, p := range sp.TxBody.P {
				for _, tr := range p.EG_TextRun {
					if tr.R != nil && strings.Contains(tr.R.T, find) {
						tr.R.T = strings.ReplaceAll(tr.R.T, find, replace)
					}
				}
			}
		}
		for _, grpSp := range choice.GrpSp {
			if grpSp != nil {
				for _, grpChoice := range grpSp.Choice {
					for _, innerSp := range grpChoice.Sp {
						if innerSp.TxBody == nil {
							continue
						}
						for _, p := range innerSp.TxBody.P {
							for _, tr := range p.EG_TextRun {
								if tr.R != nil && strings.Contains(tr.R.T, find) {
									tr.R.T = strings.ReplaceAll(tr.R.T, find, replace)
								}
							}
						}
					}
				}
			}
		}
	}
}

func applyAppendSlidesPptx(pres *presentation.Presentation, slides []SlideSpec) {
	for _, spec := range slides {
		slide := pres.AddSlide()
		tb := slide.AddTextBox()
		if spec.Title != "" {
			p := tb.AddParagraph()
			run := p.AddRun()
			run.SetText(spec.Title)
			run.Properties().SetBold(true)
			run.Properties().SetSize(24 * measurement.Point)
		}
		for _, body := range spec.Body {
			p := tb.AddParagraph()
			run := p.AddRun()
			run.SetText(body)
		}
	}
}

func applyRemoveSlidePptx(pres *presentation.Presentation, find string) {
	if find == "" {
		return
	}
	slides := pres.Slides()
	for i := len(slides) - 1; i >= 0; i-- {
		slide := slides[i]
		if slideContainsText(slide, find) {
			pres.RemoveSlide(slide)
		}
	}
}

func slideContainsText(slide presentation.Slide, text string) bool {
	for _, choice := range slide.X().CSld.SpTree.Choice {
		for _, sp := range choice.Sp {
			if sp.TxBody == nil {
				continue
			}
			for _, p := range sp.TxBody.P {
				for _, tr := range p.EG_TextRun {
					if tr.R != nil && strings.Contains(tr.R.T, text) {
						return true
					}
				}
			}
		}
	}
	return false
}
