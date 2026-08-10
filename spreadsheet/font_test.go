package spreadsheet_test

import (
	"testing"

	"github.com/yudaprama/gooxml/color"
	"github.com/yudaprama/gooxml/spreadsheet"
)

func TestFontSetBold(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()

	f := wb.StyleSheet.AddFont()
	f.SetBold(true)

	if f.X().B == nil {
		t.Errorf("expected bold to be set")
	}

	f.SetBold(false)
	if f.X().B != nil {
		t.Errorf("expected bold to be nil after unsetting")
	}
}

func TestFontSetItalic(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()

	f := wb.StyleSheet.AddFont()
	f.SetItalic(true)

	if f.X().I == nil {
		t.Errorf("expected italic to be set")
	}

	f.SetItalic(false)
	if f.X().I != nil {
		t.Errorf("expected italic to be nil after unsetting")
	}
}

func TestFontSetName(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()

	f := wb.StyleSheet.AddFont()
	f.SetName("Arial")

	if f.X().Name == nil || f.X().Name[0].ValAttr != "Arial" {
		t.Errorf("expected font name Arial")
	}
}

func TestFontSetSize(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()

	f := wb.StyleSheet.AddFont()
	f.SetSize(12.0)

	if f.X().Sz == nil || f.X().Sz[0].ValAttr != 12.0 {
		t.Errorf("expected font size 12.0")
	}
}

func TestFontSetColor(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()

	f := wb.StyleSheet.AddFont()
	f.SetColor(color.Red)

	if f.X().Color == nil {
		t.Errorf("expected color to be set")
	}
}

func TestFontIndex(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()

	f1 := wb.StyleSheet.AddFont()
	f2 := wb.StyleSheet.AddFont()

	if f1.Index() >= f2.Index() {
		t.Errorf("expected f1 index %d < f2 index %d", f1.Index(), f2.Index())
	}
}
