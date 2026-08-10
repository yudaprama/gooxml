package spreadsheet_test

import (
	"testing"

	"github.com/yudaprama/gooxml/color"
	"github.com/yudaprama/gooxml/measurement"
	"github.com/yudaprama/gooxml/schema/soo/sml"
	"github.com/yudaprama/gooxml/spreadsheet"
)

func TestRichTextAddRun(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()
	row := sheet.AddRow()
	cell := row.AddCell()

	rt := cell.SetRichTextString()

	run := rt.AddRun()
	run.SetText("hello")

	run2 := rt.AddRun()
	run2.SetText(" world")

	if len(rt.X().R) != 2 {
		t.Errorf("expected 2 runs, got %d", len(rt.X().R))
	}
}

func TestRichTextRunSetBold(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()
	row := sheet.AddRow()
	cell := row.AddCell()

	rt := cell.SetRichTextString()
	run := rt.AddRun()
	run.SetBold(true)

	if run.X().RPr == nil || run.X().RPr.B == nil {
		t.Errorf("expected bold to be set")
	}
	if run.X().RPr.B.ValAttr == nil || !*run.X().RPr.B.ValAttr {
		t.Errorf("expected bold value true")
	}
}

func TestRichTextRunSetItalic(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()
	row := sheet.AddRow()
	cell := row.AddCell()

	rt := cell.SetRichTextString()
	run := rt.AddRun()
	run.SetItalic(true)

	if run.X().RPr == nil || run.X().RPr.I == nil {
		t.Errorf("expected italic to be set")
	}
}

func TestRichTextRunSetColor(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()
	row := sheet.AddRow()
	cell := row.AddCell()

	rt := cell.SetRichTextString()
	run := rt.AddRun()
	run.SetColor(color.Red)

	if run.X().RPr == nil || run.X().RPr.Color == nil {
		t.Errorf("expected color to be set")
	}
}

func TestRichTextRunSetFont(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()
	row := sheet.AddRow()
	cell := row.AddCell()

	rt := cell.SetRichTextString()
	run := rt.AddRun()
	run.SetFont("Courier New")

	if run.X().RPr == nil || run.X().RPr.RFont == nil {
		t.Errorf("expected font to be set")
	}
	if run.X().RPr.RFont.ValAttr != "Courier New" {
		t.Errorf("expected font name 'Courier New', got '%s'", run.X().RPr.RFont.ValAttr)
	}
}

func TestRichTextRunSetSize(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()
	row := sheet.AddRow()
	cell := row.AddCell()

	rt := cell.SetRichTextString()
	run := rt.AddRun()
	run.SetSize(14 * measurement.Point)

	if run.X().RPr == nil || run.X().RPr.Sz == nil {
		t.Errorf("expected size to be set")
	}
}

func TestRichTextRunSetUnderline(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()
	row := sheet.AddRow()
	cell := row.AddCell()

	rt := cell.SetRichTextString()
	run := rt.AddRun()
	run.SetUnderline(sml.ST_UnderlineValuesSingle)

	if run.X().RPr == nil || run.X().RPr.U == nil {
		t.Errorf("expected underline to be set")
	}
}
