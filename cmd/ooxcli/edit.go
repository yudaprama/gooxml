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

// OpResult is the per-operation outcome inside an EditResult.
type OpResult struct {
	Index    int    `json:"index"`
	Type     string `json:"type"`
	Status   string `json:"status"` // "applied", "no_match", "error"
	Modified int    `json:"modified"`
	Error    string `json:"error,omitempty"`
}

// EditResult is the JSON summary emitted by "ooxcli edit --json" for
// pipeline consumers (e.g. the Rust OfficeEditTool).
type EditResult struct {
	FilePath     string     `json:"file_path"`
	OutputPath   string     `json:"output_path"`
	Success      bool       `json:"success"`
	RowsModified int        `json:"rows_modified"`
	Operations   []OpResult `json:"operations"`
	ErrorSummary string     `json:"error_summary,omitempty"`
}

func cmdEdit(args []string) error {
	jsonOut := false
	rest := make([]string, 0, len(args))
	for _, a := range args {
		if a == "--json" {
			jsonOut = true
			continue
		}
		rest = append(rest, a)
	}

	res, err := applyEdit(rest)
	if jsonOut && res != nil {
		out, merr := json.MarshalIndent(res, "", "  ")
		if merr != nil {
			return fmt.Errorf("marshal edit result: %w", merr)
		}
		fmt.Println(string(out))
	}
	return err
}

func applyEdit(args []string) (*EditResult, error) {
	input := ""
	output := ""
	opsJSON := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--out":
			i++
			if i >= len(args) {
				return nil, errors.New("--out requires a value")
			}
			output = args[i]
		case "--ops":
			i++
			if i >= len(args) {
				return nil, errors.New("--ops requires a value")
			}
			opsJSON = args[i]
		default:
			if input == "" {
				input = args[i]
			}
		}
	}
	if input == "" {
		return nil, errors.New("usage: ooxcli edit <input.docx|xlsx|pptx> [--out <output>] [--ops <json>]  (ops JSON read from stdin when --ops is omitted)")
	}

	var ops []EditOp
	if opsJSON == "" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("read ops from stdin: %w", err)
		}
		opsJSON = string(data)
	}
	if strings.TrimSpace(opsJSON) == "" {
		return nil, errors.New("no operations provided (empty --ops or stdin)")
	}
	if err := json.Unmarshal([]byte(opsJSON), &ops); err != nil {
		return nil, fmt.Errorf("parse operations: %w", err)
	}
	if len(ops) == 0 {
		return nil, errors.New("operations must not be empty")
	}

	target := input
	if output != "" {
		target = output
	}

	res := &EditResult{
		FilePath:   input,
		OutputPath: target,
		Operations: []OpResult{},
	}
	ext := fileExtEdit(input)
	var err error
	switch ext {
	case "docx":
		res.Operations, err = editDocx(input, target, ops)
	case "xlsx":
		res.Operations, err = editXlsx(input, target, ops)
	case "pptx":
		res.Operations, err = editPptx(input, target, ops)
	default:
		err = fmt.Errorf("ooxcli edit supports .docx, .xlsx, .pptx (got .%s)", ext)
	}
	if err != nil {
		res.ErrorSummary = err.Error()
		return res, err
	}
	res.Success = true
	for _, op := range res.Operations {
		res.RowsModified += op.Modified
	}
	return res, nil
}

func makeOpResult(index int, opType string, modified int, err error) OpResult {
	r := OpResult{Index: index, Type: opType, Modified: modified, Status: "applied"}
	if err != nil {
		r.Status = "error"
		r.Error = err.Error()
	} else if modified == 0 {
		r.Status = "no_match"
	}
	return r
}

func fileExtEdit(filename string) string {
	dot := strings.LastIndex(filename, ".")
	if dot < 0 {
		return ""
	}
	return strings.ToLower(filename[dot+1:])
}

// --- docx ---

func editDocx(input, output string, ops []EditOp) ([]OpResult, error) {
	doc, err := document.Open(input)
	if err != nil {
		return nil, fmt.Errorf("open docx: %w", err)
	}
	var results []OpResult
	for i, op := range ops {
		var modified int
		switch op.Type {
		case "replace_text":
			modified, err = applyReplaceTextDocx(doc, op.Find, op.Replace)
		case "append_paragraphs":
			modified = applyAppendParagraphsDocx(doc, op.Paragraphs)
		case "append_table":
			modified = applyAppendTableDocx(doc, op.Rows)
		case "delete_paragraph":
			modified = applyDeleteParagraphDocx(doc, op.Find)
		case "format_paragraph":
			modified = applyFormatParagraphDocx(doc, op.Find, op)
		default:
			err = fmt.Errorf("unknown docx operation %q", op.Type)
		}
		results = append(results, makeOpResult(i, op.Type, modified, err))
		if err != nil {
			return results, err
		}
	}
	if err := doc.SaveToFile(output); err != nil {
		return results, fmt.Errorf("save docx: %w", err)
	}
	return results, nil
}

func applyReplaceTextDocx(doc *document.Document, find, replace string) (int, error) {
	if find == "" {
		return 0, nil
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
	n := 0
	for _, para := range paras {
		n += replaceInParagraph(para, find, replace)
	}
	return n, nil
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

func replaceInParagraph(para document.Paragraph, find, replace string) int {
	runs := para.Runs()
	if len(runs) == 0 {
		return 0
	}
	for _, r := range runs {
		txt := r.Text()
		if strings.Contains(txt, find) {
			n := strings.Count(txt, find)
			r.ClearContent()
			r.AddText(strings.ReplaceAll(txt, find, replace))
			return n
		}
	}
	full := joinRuns(runs)
	if !strings.Contains(full, find) {
		return 0
	}
	n := strings.Count(full, find)
	newText := strings.ReplaceAll(full, find, replace)
	firstRun := runs[0]
	newRun := para.AddRun()
	copyRunFormatting(firstRun, newRun)
	newRun.AddText(newText)
	for _, r := range runs {
		para.RemoveRun(r)
	}
	return n
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

func applyAppendParagraphsDocx(doc *document.Document, specs []ParagraphSpec) int {
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
	return len(specs)
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

func applyAppendTableDocx(doc *document.Document, rows []RowSpec) int {
	if len(rows) == 0 {
		return 0
	}
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
	return len(rows)
}

func applyDeleteParagraphDocx(doc *document.Document, find string) int {
	if find == "" {
		return 0
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
	return len(toDelete)
}

func containsText(para document.Paragraph, text string) bool {
	for _, r := range para.Runs() {
		if strings.Contains(r.Text(), text) {
			return true
		}
	}
	return false
}

func applyFormatParagraphDocx(doc *document.Document, find string, op EditOp) int {
	if find == "" {
		return 0
	}
	paras := doc.Paragraphs()
	for _, p := range doc.StructuredDocumentTags() {
		paras = append(paras, p.Paragraphs()...)
	}
	n := 0
	for _, para := range paras {
		if !containsText(para, find) {
			continue
		}
		n++
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
	return n
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

func editXlsx(input, output string, ops []EditOp) ([]OpResult, error) {
	wb, err := spreadsheet.Open(input)
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	var results []OpResult
	for i, op := range ops {
		var modified int
		switch op.Type {
		case "replace_text":
			modified = applyReplaceTextXlsx(wb, op.Find, op.Replace)
		case "append_rows":
			modified, err = applyAppendRowsXlsx(wb, op.Sheet, op.CellRows)
		case "set_cell":
			modified, err = applySetCellXlsx(wb, op.Sheet, op.Cells)
		default:
			err = fmt.Errorf("unknown xlsx operation %q", op.Type)
		}
		results = append(results, makeOpResult(i, op.Type, modified, err))
		if err != nil {
			return results, err
		}
	}
	for _, sheet := range wb.Sheets() {
		s := sheet.X()
		if s != nil && s.Dimension == nil {
			s.Dimension = sml.NewCT_SheetDimension()
		}
	}
	if err := wb.SaveToFile(output); err != nil {
		return results, fmt.Errorf("save xlsx: %w", err)
	}
	return results, nil
}

func applyReplaceTextXlsx(wb *spreadsheet.Workbook, find, replace string) int {
	if find == "" {
		return 0
	}
	n := 0
	for _, sheet := range wb.Sheets() {
		for _, row := range sheet.Rows() {
			for _, cell := range row.Cells() {
				val := cell.GetFormattedValue()
				if strings.Contains(val, find) {
					n++
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
	return n
}

func applyAppendRowsXlsx(wb *spreadsheet.Workbook, sheetName string, rows []CellRowSpec) (int, error) {
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
	return len(rows), nil
}

func applySetCellXlsx(wb *spreadsheet.Workbook, sheetName string, cells []CellSpecXlsx) (int, error) {
	sheet, err := wb.GetSheet(sheetName)
	if err != nil {
		sheets := wb.Sheets()
		if len(sheets) == 0 {
			return 0, fmt.Errorf("no sheets in workbook")
		}
		sheet = sheets[0]
	}
	n := 0
	for _, c := range cells {
		if c.Cell == "" {
			continue
		}
		colStr, rowIdx := parseCellRef(c.Cell)
		if rowIdx == 0 {
			return n, fmt.Errorf("invalid cell reference %q", c.Cell)
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
		n++
	}
	return n, nil
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

func editPptx(input, output string, ops []EditOp) ([]OpResult, error) {
	pres, err := presentation.Open(input)
	if err != nil {
		return nil, fmt.Errorf("open pptx: %w", err)
	}
	var results []OpResult
	for i, op := range ops {
		var modified int
		switch op.Type {
		case "replace_text":
			modified = applyReplaceTextPptx(pres, op.Find, op.Replace)
		case "append_slides":
			modified = applyAppendSlidesPptx(pres, op.Slides)
		case "remove_slide":
			modified = applyRemoveSlidePptx(pres, op.Find)
		default:
			err = fmt.Errorf("unknown pptx operation %q", op.Type)
		}
		results = append(results, makeOpResult(i, op.Type, modified, err))
		if err != nil {
			return results, err
		}
	}
	if err := pres.SaveToFile(output); err != nil {
		return results, fmt.Errorf("save pptx: %w", err)
	}
	return results, nil
}

func applyReplaceTextPptx(pres *presentation.Presentation, find, replace string) int {
	if find == "" {
		return 0
	}
	n := 0
	for _, slide := range pres.Slides() {
		n += replaceInSlideContent(slide, find, replace)
	}
	return n
}

func replaceInSlideContent(slide presentation.Slide, find, replace string) int {
	n := 0
	for _, choice := range slide.X().CSld.SpTree.Choice {
		for _, sp := range choice.Sp {
			if sp.TxBody == nil {
				continue
			}
			for _, p := range sp.TxBody.P {
				for _, tr := range p.EG_TextRun {
					if tr.R != nil && strings.Contains(tr.R.T, find) {
						n++
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
									n++
									tr.R.T = strings.ReplaceAll(tr.R.T, find, replace)
								}
							}
						}
					}
				}
			}
		}
	}
	return n
}

func applyAppendSlidesPptx(pres *presentation.Presentation, slides []SlideSpec) int {
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
	return len(slides)
}

func applyRemoveSlidePptx(pres *presentation.Presentation, find string) int {
	if find == "" {
		return 0
	}
	removed := 0
	slides := pres.Slides()
	for i := len(slides) - 1; i >= 0; i-- {
		slide := slides[i]
		if slideContainsText(slide, find) {
			pres.RemoveSlide(slide)
			removed++
		}
	}
	return removed
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
