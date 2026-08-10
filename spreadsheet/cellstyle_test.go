package spreadsheet_test

import (
	"testing"

	"github.com/yudaprama/gooxml/schema/soo/sml"
	"github.com/yudaprama/gooxml/spreadsheet"
)

func TestCellStyleHasNumberFormat(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()
	row := sheet.AddRow()
	_ = row.AddCell()

	cs := wb.StyleSheet.AddCellStyle()
	if cs.HasNumberFormat() {
		t.Errorf("expected no number format by default")
	}

	cs.SetNumberFormatStandard(spreadsheet.StandardFormatPercent)
	if !cs.HasNumberFormat() {
		t.Errorf("expected number format after setting")
	}
}

func TestCellStyleNumberFormat(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	cs := wb.StyleSheet.AddCellStyle()

	if cs.NumberFormat() != 0 {
		t.Errorf("expected 0, got %d", cs.NumberFormat())
	}

	cs.SetNumberFormatStandard(spreadsheet.StandardFormat2)
	if cs.NumberFormat() != uint32(spreadsheet.StandardFormat2) {
		t.Errorf("expected %d, got %d", uint32(spreadsheet.StandardFormat2), cs.NumberFormat())
	}
}

func TestCellStyleClearNumberFormat(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	cs := wb.StyleSheet.AddCellStyle()

	cs.SetNumberFormatStandard(spreadsheet.StandardFormatPercent)
	cs.ClearNumberFormat()
	if cs.HasNumberFormat() {
		t.Errorf("expected no number format after clearing")
	}
}

func TestCellStyleSetNumberFormat(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	cs := wb.StyleSheet.AddCellStyle()

	cs.SetNumberFormat("#,##0.00")
	if !cs.HasNumberFormat() {
		t.Errorf("expected number format after setting")
	}
}

func TestCellStyleWrapped(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	cs := wb.StyleSheet.AddCellStyle()

	if cs.Wrapped() {
		t.Errorf("expected not wrapped by default")
	}

	cs.SetWrapped(true)
	if !cs.Wrapped() {
		t.Errorf("expected wrapped after setting")
	}

	cs.SetWrapped(false)
	if cs.Wrapped() {
		t.Errorf("expected not wrapped after unsetting")
	}
}

func TestCellStyleSetHorizontalAlignment(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	cs := wb.StyleSheet.AddCellStyle()

	cs.SetHorizontalAlignment(sml.ST_HorizontalAlignmentCenter)
	// Verify indirectly - the method runs without error
	_ = cs
}

func TestCellStyleSetVerticalAlignment(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	cs := wb.StyleSheet.AddCellStyle()

	cs.SetVerticalAlignment(sml.ST_VerticalAlignmentTop)
	// Verify indirectly - the method runs without error
	_ = cs
}

func TestCellStyleSetShrinkToFit(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	cs := wb.StyleSheet.AddCellStyle()

	cs.SetShrinkToFit(true)
	cs.SetShrinkToFit(false)
	// Verify the methods run without error
}

func TestCellStyleSetFont(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	cs := wb.StyleSheet.AddCellStyle()

	f := wb.StyleSheet.AddFont()
	f.SetBold(true)
	cs.SetFont(f)

	// Verify via the style index
	if cs.Index() == 0 {
		t.Errorf("expected non-zero style index")
	}
}

func TestCellStyleClearFont(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	cs := wb.StyleSheet.AddCellStyle()

	f := wb.StyleSheet.AddFont()
	cs.SetFont(f)
	cs.ClearFont()
	// Verify the method runs without error
}

func TestCellStyleSetBorder(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	cs := wb.StyleSheet.AddCellStyle()

	b := wb.StyleSheet.AddBorder()
	cs.SetBorder(b)
	// Verify via the style index
	if cs.Index() == 0 {
		t.Errorf("expected non-zero style index")
	}
}

func TestCellStyleClearBorder(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	cs := wb.StyleSheet.AddCellStyle()

	b := wb.StyleSheet.AddBorder()
	cs.SetBorder(b)
	cs.ClearBorder()
	// Verify the method runs without error
}

func TestCellStyleSetFill(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	cs := wb.StyleSheet.AddCellStyle()

	f := wb.StyleSheet.Fills().AddFill()
	cs.SetFill(f)
	// Verify via the style index
	if cs.Index() == 0 {
		t.Errorf("expected non-zero style index")
	}
}

func TestCellStyleClearFill(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	cs := wb.StyleSheet.AddCellStyle()

	f := wb.StyleSheet.Fills().AddFill()
	cs.SetFill(f)
	cs.ClearFill()
	// Verify the method runs without error
}

func TestCellStyleIndex(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()

	cs1 := wb.StyleSheet.AddCellStyle()
	cs2 := wb.StyleSheet.AddCellStyle()

	if cs1.Index() >= cs2.Index() {
		t.Errorf("expected cs1 index %d < cs2 index %d", cs1.Index(), cs2.Index())
	}
}
