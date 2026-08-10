package spreadsheet_test

import (
	"testing"

	"github.com/yudaprama/gooxml/measurement"
	"github.com/yudaprama/gooxml/spreadsheet"
)

func TestColumnSetWidth(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	col := sheet.Column(0)
	col.SetWidth(10 * measurement.Character)
	if col.X().WidthAttr == nil {
		t.Errorf("expected width to be set")
	}
}

func TestColumnSetStyle(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	cs := wb.StyleSheet.AddCellStyle()
	col := sheet.Column(0)
	col.SetStyle(cs)

	if col.X().StyleAttr == nil {
		t.Errorf("expected style to be set")
	}
}

func TestColumnSetHidden(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	col := sheet.Column(0)

	if col.X().HiddenAttr != nil {
		t.Errorf("expected not hidden by default")
	}

	col.SetHidden(true)
	if col.X().HiddenAttr == nil || !*col.X().HiddenAttr {
		t.Errorf("expected column hidden after setting")
	}

	col.SetHidden(false)
	if col.X().HiddenAttr != nil {
		t.Errorf("expected column not hidden after unsetting")
	}
}
