package spreadsheet_test

import (
	"testing"

	"github.com/yudaprama/gooxml/schema/soo/sml"
	"github.com/yudaprama/gooxml/spreadsheet"
)

func TestDataValidationSetAllowBlank(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	dv := sheet.AddDataValidation()

	dv.SetAllowBlank(true)
	if dv.X().AllowBlankAttr == nil || !*dv.X().AllowBlankAttr {
		t.Errorf("expected allow blank true")
	}

	dv.SetAllowBlank(false)
	if dv.X().AllowBlankAttr != nil {
		t.Errorf("expected allow blank nil after unsetting")
	}
}

func TestDataValidationSetRange(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	dv := sheet.AddDataValidation()
	dv.SetRange("A1:A10")

	if dv.X().SqrefAttr == nil || len(dv.X().SqrefAttr) == 0 {
		t.Errorf("expected range to be set")
	}
}

func TestDataValidationSetList(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	dv := sheet.AddDataValidation()
	dvl := dv.SetList()

	if dv.X().TypeAttr != sml.ST_DataValidationTypeList {
		t.Errorf("expected type List")
	}

	dvl.SetValues([]string{"a", "b", "c"})
	if dv.X().Formula1 == nil {
		t.Errorf("expected formula1 to be set")
	}
}

func TestDataValidationListSetRange(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	dv := sheet.AddDataValidation()
	dvl := dv.SetList()

	dvl.SetRange("B1:B10")
	if dv.X().Formula1 == nil || *dv.X().Formula1 != "B1:B10" {
		t.Errorf("expected formula1 to be B1:B10")
	}
}

func TestDataValidationSetComparison(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	dv := sheet.AddDataValidation()
	dvc := dv.SetComparison(spreadsheet.DVCompareTypeWholeNumber, spreadsheet.DVCompareOpBetween)

	if dv.X().TypeAttr != sml.ST_DataValidationTypeWhole {
		t.Errorf("expected type Whole")
	}
	if dv.X().OperatorAttr != sml.ST_DataValidationOperatorBetween {
		t.Errorf("expected operator Between")
	}

	dvc.SetValue("1")
	dvc.SetValue2("10")
	if dv.X().Formula1 == nil || *dv.X().Formula1 != "1" {
		t.Errorf("expected formula1 to be 1")
	}
	if dv.X().Formula2 == nil || *dv.X().Formula2 != "10" {
		t.Errorf("expected formula2 to be 10")
	}
}
