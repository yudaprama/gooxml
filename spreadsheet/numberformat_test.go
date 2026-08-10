package spreadsheet_test

import (
	"testing"

	"github.com/yudaprama/gooxml/spreadsheet"
)

func TestNumberFormatSetGetFormat(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()

	nf := wb.StyleSheet.AddNumberFormat()
	nf.SetFormat("#,##0.00")

	if nf.GetFormat() != "#,##0.00" {
		t.Errorf("expected format '#,##0.00', got '%s'", nf.GetFormat())
	}
}

func TestNumberFormatID(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()

	nf := wb.StyleSheet.AddNumberFormat()
	if nf.ID() == 0 {
		t.Errorf("expected non-zero ID for custom format")
	}
}
