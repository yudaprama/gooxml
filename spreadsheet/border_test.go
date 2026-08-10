package spreadsheet_test

import (
	"testing"

	"github.com/yudaprama/gooxml/color"
	"github.com/yudaprama/gooxml/schema/soo/sml"
	"github.com/yudaprama/gooxml/spreadsheet"
)

func TestBorderInitializeDefaults(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()

	b := wb.StyleSheet.AddBorder()
	b.InitializeDefaults()

	if b.X().Left == nil {
		t.Errorf("expected left border to be initialized")
	}
	if b.X().Right == nil {
		t.Errorf("expected right border to be initialized")
	}
	if b.X().Top == nil {
		t.Errorf("expected top border to be initialized")
	}
	if b.X().Bottom == nil {
		t.Errorf("expected bottom border to be initialized")
	}
	if b.X().Diagonal == nil {
		t.Errorf("expected diagonal border to be initialized")
	}
}

func TestBorderSetLeft(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()

	b := wb.StyleSheet.AddBorder()
	b.SetLeft(sml.ST_BorderStyleThin, color.Red)

	if b.X().Left == nil {
		t.Errorf("expected left border to be set")
	}
	if b.X().Left.StyleAttr != sml.ST_BorderStyleThin {
		t.Errorf("expected thin border style")
	}
}

func TestBorderSetRight(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()

	b := wb.StyleSheet.AddBorder()
	b.SetRight(sml.ST_BorderStyleThick, color.Blue)

	if b.X().Right == nil {
		t.Errorf("expected right border to be set")
	}
	if b.X().Right.StyleAttr != sml.ST_BorderStyleThick {
		t.Errorf("expected thick border style")
	}
}

func TestBorderSetTop(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()

	b := wb.StyleSheet.AddBorder()
	b.SetTop(sml.ST_BorderStyleDashed, color.Green)

	if b.X().Top == nil {
		t.Errorf("expected top border to be set")
	}
	if b.X().Top.StyleAttr != sml.ST_BorderStyleDashed {
		t.Errorf("expected dashed border style")
	}
}

func TestBorderSetBottom(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()

	b := wb.StyleSheet.AddBorder()
	b.SetBottom(sml.ST_BorderStyleDotted, color.Black)

	if b.X().Bottom == nil {
		t.Errorf("expected bottom border to be set")
	}
	if b.X().Bottom.StyleAttr != sml.ST_BorderStyleDotted {
		t.Errorf("expected dotted border style")
	}
}

func TestBorderSetDiagonal(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()

	b := wb.StyleSheet.AddBorder()
	b.SetDiagonal(sml.ST_BorderStyleDouble, color.Red, true, true)

	if b.X().Diagonal == nil {
		t.Errorf("expected diagonal border to be set")
	}
	if b.X().Diagonal.StyleAttr != sml.ST_BorderStyleDouble {
		t.Errorf("expected double border style")
	}
	if b.X().DiagonalUpAttr == nil || !*b.X().DiagonalUpAttr {
		t.Errorf("expected diagonal up")
	}
	if b.X().DiagonalDownAttr == nil || !*b.X().DiagonalDownAttr {
		t.Errorf("expected diagonal down")
	}
}

func TestBorderIndex(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()

	b1 := wb.StyleSheet.AddBorder()
	b2 := wb.StyleSheet.AddBorder()

	if b1.Index() >= b2.Index() {
		t.Errorf("expected b1 index %d < b2 index %d", b1.Index(), b2.Index())
	}
}
