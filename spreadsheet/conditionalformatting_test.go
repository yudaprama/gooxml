package spreadsheet_test

import (
	"testing"

	"github.com/yudaprama/gooxml/color"
	"github.com/yudaprama/gooxml/schema/soo/sml"
	"github.com/yudaprama/gooxml/spreadsheet"
)

func TestConditionalFormattingAddRule(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	cf := sheet.AddConditionalFormatting([]string{"A1:A10"})
	rule := cf.AddRule()

	if rule.X() == nil {
		t.Errorf("expected rule to be created")
	}
	if rule.Priority() <= 0 {
		t.Errorf("expected positive priority, got %d", rule.Priority())
	}
	if rule.Type() != sml.ST_CfTypeCellIs {
		t.Errorf("expected type CellIs")
	}
}

func TestConditionalFormattingMultipleRules(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	cf := sheet.AddConditionalFormatting([]string{"A1:A10"})
	r1 := cf.AddRule()
	r2 := cf.AddRule()

	if r1.Priority() >= r2.Priority() {
		t.Errorf("expected r1 priority %d < r2 priority %d", r1.Priority(), r2.Priority())
	}
}

func TestConditionalFormattingRuleSetType(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	cf := sheet.AddConditionalFormatting([]string{"A1:A10"})
	rule := cf.AddRule()

	rule.SetType(sml.ST_CfTypeTop10)
	if rule.Type() != sml.ST_CfTypeTop10 {
		t.Errorf("expected type Top10")
	}
}

func TestConditionalFormattingRuleSetOperator(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	cf := sheet.AddConditionalFormatting([]string{"A1:A10"})
	rule := cf.AddRule()

	rule.SetOperator(sml.ST_ConditionalFormattingOperatorLessThan)
	if rule.Operator() != sml.ST_ConditionalFormattingOperatorLessThan {
		t.Errorf("expected operator LessThan")
	}
}

func TestConditionalFormattingRuleSetConditionValue(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	cf := sheet.AddConditionalFormatting([]string{"A1:A10"})
	rule := cf.AddRule()

	rule.SetConditionValue("100")
	if len(rule.X().Formula) != 1 || rule.X().Formula[0] != "100" {
		t.Errorf("expected formula '100'")
	}
}

func TestConditionalFormattingRuleSetPriority(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	cf := sheet.AddConditionalFormatting([]string{"A1:A10"})
	rule := cf.AddRule()

	rule.SetPriority(5)
	if rule.Priority() != 5 {
		t.Errorf("expected priority 5, got %d", rule.Priority())
	}
}

func TestConditionalFormattingRuleSetColorScale(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	cf := sheet.AddConditionalFormatting([]string{"A1:A10"})
	rule := cf.AddRule()

	cs := rule.SetColorScale()
	if cs.X() == nil {
		t.Errorf("expected color scale to be created")
	}
	if rule.Type() != sml.ST_CfTypeColorScale {
		t.Errorf("expected type ColorScale")
	}

	cs.AddFormatValue(sml.ST_CfvoTypeMin, "0")
	cs.AddFormatValue(sml.ST_CfvoTypeMax, "100")
	cs.AddGradientStop(color.Red)
	cs.AddGradientStop(color.Green)

	if len(cs.X().Cfvo) != 2 {
		t.Errorf("expected 2 format values, got %d", len(cs.X().Cfvo))
	}
	if len(cs.X().Color) != 2 {
		t.Errorf("expected 2 gradient stops, got %d", len(cs.X().Color))
	}
}

func TestConditionalFormattingRuleSetIcons(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	cf := sheet.AddConditionalFormatting([]string{"A1:A10"})
	rule := cf.AddRule()

	is := rule.SetIcons()
	if is.X() == nil {
		t.Errorf("expected icon scale to be created")
	}
	if rule.Type() != sml.ST_CfTypeIconSet {
		t.Errorf("expected type IconSet")
	}

	is.SetIcons(sml.ST_IconSetType3TrafficLights1)
	if is.X().IconSetAttr != sml.ST_IconSetType3TrafficLights1 {
		t.Errorf("expected traffic lights icon set")
	}

	is.AddFormatValue(sml.ST_CfvoTypeNum, "0")
	is.AddFormatValue(sml.ST_CfvoTypeNum, "33")
	is.AddFormatValue(sml.ST_CfvoTypeNum, "67")
	if len(is.X().Cfvo) != 3 {
		t.Errorf("expected 3 format values, got %d", len(is.X().Cfvo))
	}
}

func TestConditionalFormattingRuleSetDataBar(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	cf := sheet.AddConditionalFormatting([]string{"A1:A10"})
	rule := cf.AddRule()

	db := rule.SetDataBar()
	if db.X() == nil {
		t.Errorf("expected data bar to be created")
	}
	if rule.Type() != sml.ST_CfTypeDataBar {
		t.Errorf("expected type DataBar")
	}

	db.SetShowValue(true)
	if db.X().ShowValueAttr == nil || !*db.X().ShowValueAttr {
		t.Errorf("expected show value true")
	}

	db.SetMinLength(10)
	db.SetMaxLength(90)
	if db.X().MinLengthAttr == nil || *db.X().MinLengthAttr != 10 {
		t.Errorf("expected min length 10")
	}
	if db.X().MaxLengthAttr == nil || *db.X().MaxLengthAttr != 90 {
		t.Errorf("expected max length 90")
	}

	db.AddFormatValue(sml.ST_CfvoTypeMin, "0")
	db.AddFormatValue(sml.ST_CfvoTypeMax, "100")
	if len(db.X().Cfvo) != 2 {
		t.Errorf("expected 2 format values, got %d", len(db.X().Cfvo))
	}

	db.SetColor(color.Blue)
	if db.X().Color == nil {
		t.Errorf("expected color to be set")
	}
}
