package tagex

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestKV_Valid(t *testing.T) {
	k, v, err := kv("foo=bar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if k != "foo" || v != "bar" {
		t.Errorf("expected (foo,bar), got (%q,%q)", k, v)
	}
}

func TestKV_Invalid(t *testing.T) {
	_, _, err := kv("malformed")
	if err == nil {
		t.Fatal("expected error for malformed kv pair")
	}
	if !strings.Contains(err.Error(), "malformed") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestExtractPairs(t *testing.T) {
	input := []string{"key1=value1", "key2 = value2"}
	pairs, err := extractPairs(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pairs["key1"] != "value1" || pairs["key2"] != "value2" {
		t.Errorf("unexpected pairs: %+v", pairs)
	}
}

func TestSplitTagValue_Success(t *testing.T) {
	tagStr := "dummy, key=val, foo=bar"
	id, args, err := splitTagValue(tagStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "dummy" {
		t.Errorf("expected id %q, got %q", "dummy", id)
	}
	if args["key"] != "val" || args["foo"] != "bar" {
		t.Errorf("unexpected args: %+v", args)
	}
}

func TestSplitTagValue_NoDirective(t *testing.T) {
	_, _, err := splitTagValue("")
	if err == nil {
		t.Fatal("expected error for empty directive")
	}
	if !errors.Is(err, errors.New("no directive set")) && !strings.Contains(err.Error(), "no directive") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValParse_Success(t *testing.T) {
	var num int
	v := reflect.ValueOf(&num).Elem()
	v.SetInt(123) // pre-set value (will be overwritten)
	parsed, err := valParse[int](v)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed != 123 {
		t.Errorf("expected 123, got %d", parsed)
	}
}

func TestValParse_Failure(t *testing.T) {
	// Test with an unexported field so CanInterface returns false.
	type testStruct struct {
		private int
	}
	ts := testStruct{private: 5}
	v := reflect.ValueOf(ts).FieldByName("private")
	_, err := valParse[int](v)
	if err == nil {
		t.Fatal("expected error for unexported field")
	}
}

type dummyDirective struct {
	name string
}

func (d *dummyDirective) Name() string {
	return d.name
}

func (d *dummyDirective) Handle(val int) (int, error) {
	if val != 42 {
		return val, fmt.Errorf("dummyDirective: expected 42, got %d", val)
	}
	return val, nil
}

func TestDirectiveWrapper_HandleAny_Success(t *testing.T) {
	dd := &dummyDirective{name: "dummy"}
	wrapper := directiveWrapper[int]{Directive: dd}

	var val = 42
	v := reflect.ValueOf(val)

	err := wrapper.HandleAny(v)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	underlying := wrapper.Unwrap()
	if underlying != dd {
		t.Errorf("expected underlying directive %v, got %v", dd, underlying)
	}
}

func TestProcessDirective_Success(t *testing.T) {
	tag := &Tag{}
	dd := &dummyDirective{name: "dummy"}
	RegisterDirective[int](tag, dd)

	tagValue := "dummy, pass=true"

	fieldVal := 42
	v := reflect.ValueOf(fieldVal)

	err := processDirective(tag, tagValue, v)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestProcessDirective_UnknownDirective(t *testing.T) {
	tag := &Tag{}
	tagValue := "nonexistent, pass=true"
	fieldVal := 42
	v := reflect.ValueOf(fieldVal)

	err := processDirective(tag, tagValue, v)
	if err == nil {
		t.Fatal("expected error for unknown directive, got nil")
	}
	if !strings.Contains(err.Error(), "unknown directive") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestProcessDirective_FailingHandleAny(t *testing.T) {
	dd := &dummyDirective{name: "dummy"}
	tag := &Tag{}
	RegisterDirective[int](tag, dd)

	tagValue := "dummy, pass=true"
	fieldVal := 100
	v := reflect.ValueOf(fieldVal)

	err := processDirective(tag, tagValue, v)
	if err == nil {
		t.Fatal("expected error from HandleAny, got nil")
	}
	if !strings.Contains(err.Error(), "expected 42") {
		t.Errorf("unexpected error message: %v", err)
	}
}
