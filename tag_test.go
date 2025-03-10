package tagex

import (
	"strings"
	"testing"
)

type ValImplTest struct {
	Number int    `val:"range, min=0, max=3"`
	Word   string `val:"length, min=2, max=5"`
}

func TestNewTag(t *testing.T) {
	tag := NewTag(valTagKey)
	if tag.Key != valTagKey {
		t.Errorf("expected key %q, got %q", valTagKey, tag.Key)
	}
}

func TestSetAndGetDirective(t *testing.T) {
	tag := NewTag(valTagKey)
	RegisterDirective[int](&tag, &RangeDirective{})

	expect := "range"
	got, ok := tag.get(expect)
	if !ok {
		t.Fatalf("expected directive %s to be registered", expect)
	}
	if got == nil {
		t.Fatal("got nil directive")
	}
}

func TestProcessStruct_InvalidInput(t *testing.T) {
	tag := NewTag(valTagKey)

	_, err := tag.ProcessStruct("not-a-struct")
	if err == nil {
		t.Fatal("expected error for non-struct input")
	}
	if !strings.Contains(err.Error(), "expected a struct") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestProcessStruct_Success(t *testing.T) {
	tag := NewTag(valTagKey)
	RegisterDirective[int](&tag, &RangeDirective{})
	RegisterDirective[string](&tag, &LengthDirective{})

	ts := ValImplTest{
		Number: 2,
		Word:   "tagex",
	}
	ok, err := tag.ProcessStruct(&ts)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !ok {
		t.Fatal("expected ok to be true")
	}
}

func TestProcessStruct_Failure(t *testing.T) {
	tag := NewTag("val")

	ts := ValImplTest{
		Number: 2,
	}
	ok, err := tag.ProcessStruct(&ts)
	if err == nil {
		t.Fatal("expected error due to missing directive")
	}
	if ok {
		t.Fatal("expected ok to be false")
	}
	if !strings.Contains(err.Error(), "unknown directive") {
		t.Errorf("unexpected error message: %v", err)
	}
}
