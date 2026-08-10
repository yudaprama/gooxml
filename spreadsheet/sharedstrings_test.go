package spreadsheet_test

import (
	"testing"

	"github.com/yudaprama/gooxml/spreadsheet"
)

func TestSharedStringsAddAndGet(t *testing.T) {
	ss := spreadsheet.NewSharedStrings()

	id1 := ss.AddString("hello")
	if id1 != 0 {
		t.Errorf("expected first string ID 0, got %d", id1)
	}

	id2 := ss.AddString("world")
	if id2 != 1 {
		t.Errorf("expected second string ID 1, got %d", id2)
	}

	s, err := ss.GetString(0)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if s != "hello" {
		t.Errorf("expected 'hello', got '%s'", s)
	}

	s, err = ss.GetString(1)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if s != "world" {
		t.Errorf("expected 'world', got '%s'", s)
	}
}

func TestSharedStringsDedup(t *testing.T) {
	ss := spreadsheet.NewSharedStrings()

	id1 := ss.AddString("hello")
	id2 := ss.AddString("hello")

	if id1 != id2 {
		t.Errorf("expected same ID for duplicate string, got %d and %d", id1, id2)
	}
}

func TestSharedStringsCount(t *testing.T) {
	ss := spreadsheet.NewSharedStrings()

	ss.AddString("a")
	ss.AddString("b")
	ss.AddString("c")

	if ss.X().CountAttr == nil || *ss.X().CountAttr != 3 {
		t.Errorf("expected count 3")
	}
	if ss.X().UniqueCountAttr == nil || *ss.X().UniqueCountAttr != 3 {
		t.Errorf("expected unique count 3")
	}
}

func TestSharedStringsGetInvalidIndex(t *testing.T) {
	ss := spreadsheet.NewSharedStrings()

	_, err := ss.GetString(-1)
	if err == nil {
		t.Errorf("expected error for negative index")
	}

	_, err = ss.GetString(100)
	if err == nil {
		t.Errorf("expected error for out-of-range index")
	}
}
