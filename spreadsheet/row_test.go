package spreadsheet_test

import (
	"testing"

	"github.com/yudaprama/gooxml/measurement"
	"github.com/yudaprama/gooxml/spreadsheet"
)

func TestRowNumber(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	r := sheet.AddRow()
	if r.RowNumber() != 1 {
		t.Errorf("expected row number 1, got %d", r.RowNumber())
	}

	r2 := sheet.AddRow()
	if r2.RowNumber() != 2 {
		t.Errorf("expected row number 2, got %d", r2.RowNumber())
	}
}

func TestRowSetHeight(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()
	row := sheet.AddRow()

	row.SetHeight(20 * measurement.Point)
	if row.X().HtAttr == nil {
		t.Errorf("expected height to be set")
	}
}

func TestRowSetHeightAuto(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()
	row := sheet.AddRow()

	row.SetHeight(20 * measurement.Point)
	row.SetHeightAuto()
	if row.X().HtAttr != nil {
		t.Errorf("expected height to be nil after auto")
	}
}

func TestRowHidden(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()
	row := sheet.AddRow()

	if row.IsHidden() {
		t.Errorf("expected row not hidden by default")
	}

	row.SetHidden(true)
	if !row.IsHidden() {
		t.Errorf("expected row hidden after setting")
	}

	row.SetHidden(false)
	if row.IsHidden() {
		t.Errorf("expected row not hidden after unsetting")
	}
}

func TestRowAddCell(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()
	row := sheet.AddRow()

	cell := row.AddCell()
	if cell.Reference() != "A1" {
		t.Errorf("expected cell reference A1, got %s", cell.Reference())
	}

	cell2 := row.AddCell()
	if cell2.Reference() != "B1" {
		t.Errorf("expected cell reference B1, got %s", cell2.Reference())
	}
}

func TestRowCells(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()
	row := sheet.AddRow()

	if len(row.Cells()) != 0 {
		t.Errorf("expected 0 cells initially")
	}

	row.AddCell()
	row.AddCell()
	if len(row.Cells()) != 2 {
		t.Errorf("expected 2 cells after adding")
	}
}

func TestRowAddNamedCell(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()
	row := sheet.AddRow()

	cell := row.AddNamedCell("C")
	if cell.Reference() != "C1" {
		t.Errorf("expected cell reference C1, got %s", cell.Reference())
	}
}

func TestRowCell(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()
	row := sheet.AddRow()

	cell := row.Cell("A")
	if cell.Reference() != "A1" {
		t.Errorf("expected cell reference A1, got %s", cell.Reference())
	}

	cell2 := row.Cell("A")
	if cell2.Reference() != "A1" {
		t.Errorf("expected same cell reference A1, got %s", cell2.Reference())
	}
}
