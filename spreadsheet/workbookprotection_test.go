package spreadsheet_test

import (
	"testing"

	"github.com/yudaprama/gooxml/spreadsheet"
)

func TestWorkbookProtectionLockStructure(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()

	wp := wb.Protection()

	if wp.IsStructureLocked() {
		t.Errorf("expected structure not locked by default")
	}

	wp.LockStructure(true)
	if !wp.IsStructureLocked() {
		t.Errorf("expected structure locked after setting")
	}

	wp.LockStructure(false)
	if wp.IsStructureLocked() {
		t.Errorf("expected structure not locked after unsetting")
	}
}

func TestWorkbookProtectionLockWindow(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()

	wp := wb.Protection()

	if wp.IsWindowLocked() {
		t.Errorf("expected window not locked by default")
	}

	wp.LockWindow(true)
	if !wp.IsWindowLocked() {
		t.Errorf("expected window locked after setting")
	}

	wp.LockWindow(false)
	if wp.IsWindowLocked() {
		t.Errorf("expected window not locked after unsetting")
	}
}

func TestWorkbookProtectionPassword(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()

	wp := wb.Protection()

	if wp.PasswordHash() != "" {
		t.Errorf("expected empty password hash by default")
	}

	wp.SetPassword("secret")
	if wp.PasswordHash() == "" {
		t.Errorf("expected non-empty password hash after setting")
	}
}

func TestWorkbookProtectionPasswordHash(t *testing.T) {
	wb := spreadsheet.New()
	defer wb.Close()

	wp := wb.Protection()

	wp.SetPasswordHash("1234")
	if wp.PasswordHash() != "1234" {
		t.Errorf("expected hash 1234, got %s", wp.PasswordHash())
	}
}
