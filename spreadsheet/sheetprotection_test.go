package spreadsheet_test

import (
	"testing"

	"github.com/yudaprama/gooxml/spreadsheet"
)

func TestSheetProtectionLockSheet(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	sp := sheet.Protection()

	if sp.IsSheetLocked() {
		t.Errorf("expected sheet not locked by default")
	}

	sp.LockSheet(true)
	if !sp.IsSheetLocked() {
		t.Errorf("expected sheet locked after setting")
	}

	sp.LockSheet(false)
	if sp.IsSheetLocked() {
		t.Errorf("expected sheet not locked after unsetting")
	}
}

func TestSheetProtectionLockObject(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	sp := sheet.Protection()

	if sp.IsObjectLocked() {
		t.Errorf("expected objects not locked by default")
	}

	sp.LockObject(true)
	if !sp.IsObjectLocked() {
		t.Errorf("expected objects locked after setting")
	}

	sp.LockObject(false)
	if sp.IsObjectLocked() {
		t.Errorf("expected objects not locked after unsetting")
	}
}

func TestSheetProtectionPassword(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	sp := sheet.Protection()

	if sp.PasswordHash() != "" {
		t.Errorf("expected empty password hash by default")
	}

	sp.SetPassword("test")
	if sp.PasswordHash() == "" {
		t.Errorf("expected non-empty password hash after setting")
	}
}

func TestSheetProtectionPasswordHash(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()
	sheet := wb.AddSheet()

	sp := sheet.Protection()

	sp.SetPasswordHash("ABCD")
	if sp.PasswordHash() != "ABCD" {
		t.Errorf("expected hash ABCD, got %s", sp.PasswordHash())
	}
}
