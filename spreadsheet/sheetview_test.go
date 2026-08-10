package spreadsheet_test

import (
	"testing"

	"github.com/yudaprama/gooxml/schema/soo/sml"
	"github.com/yudaprama/gooxml/spreadsheet"
)

func TestSheetViewState(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	sv := sheet.InitialView()

	sv.SetState(sml.ST_PaneStateFrozen)
	if sv.X().Pane == nil {
		t.Errorf("expected pane to be set")
	}
	if sv.X().Pane.StateAttr != sml.ST_PaneStateFrozen {
		t.Errorf("expected frozen state")
	}
}

func TestSheetViewSplit(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	sv := sheet.InitialView()

	sv.SetXSplit(5)
	sv.SetYSplit(10)

	if sv.X().Pane == nil {
		t.Errorf("expected pane to be set")
	}
	if sv.X().Pane.XSplitAttr == nil || *sv.X().Pane.XSplitAttr != 5 {
		t.Errorf("expected XSplit 5")
	}
	if sv.X().Pane.YSplitAttr == nil || *sv.X().Pane.YSplitAttr != 10 {
		t.Errorf("expected YSplit 10")
	}
}

func TestSheetViewTopLeft(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	sv := sheet.InitialView()

	sv.SetTopLeft("B3")
	if sv.X().Pane == nil {
		t.Errorf("expected pane to be set")
	}
	if sv.X().Pane.TopLeftCellAttr == nil || *sv.X().Pane.TopLeftCellAttr != "B3" {
		t.Errorf("expected top left B3")
	}
}

func TestSheetViewZoom(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	sv := sheet.InitialView()

	sv.SetZoom(150)
	if sv.X().ZoomScaleAttr == nil || *sv.X().ZoomScaleAttr != 150 {
		t.Errorf("expected zoom 150")
	}
}

func TestSheetViewShowRuler(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	sv := sheet.InitialView()

	sv.SetShowRuler(false)
	if sv.X().ShowRulerAttr == nil || *sv.X().ShowRulerAttr {
		t.Errorf("expected ruler hidden")
	}

	sv.SetShowRuler(true)
	if sv.X().ShowRulerAttr != nil {
		t.Errorf("expected ruler shown (nil = default true)")
	}
}
