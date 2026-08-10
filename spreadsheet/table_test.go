package spreadsheet_test

import (
	"testing"

	"github.com/yudaprama/gooxml/spreadsheet"
)

func TestTable(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()

	tables := wb.Tables()
	if len(tables) != 0 {
		t.Errorf("expected 0 tables initially")
	}
}
