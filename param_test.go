package tagex

import (
	"errors"
	"reflect"
	"testing"
)

type DummyStruct struct {
	Name   string  `param:"name"`
	Age    int     `param:"age"`
	Score  float64 `param:"score"`
	Active bool    `param:"active"`
}

func TestProcessParams_Success(t *testing.T) {
	ds := DummyStruct{}
	args := map[string]string{
		"name":   "Alice",
		"age":    "30",
		"score":  "95.5",
		"active": "true",
	}
	ok, err := processParams(&ds, args)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !ok {
		t.Fatal("expected ok==true")
	}

	if ds.Name != "Alice" {
		t.Errorf("expected Name 'Alice', got %q", ds.Name)
	}
	if ds.Age != 30 {
		t.Errorf("expected Age 30, got %d", ds.Age)
	}
	if ds.Score != 95.5 {
		t.Errorf("expected Score 95.5, got %f", ds.Score)
	}
	if ds.Active != true {
		t.Errorf("expected Active true, got %v", ds.Active)
	}
}

func TestProcessParams_NonPointer(t *testing.T) {
	ds := DummyStruct{}
	_, err := processParams(ds, map[string]string{})
	if err == nil {
		t.Fatal("expected error for non-pointer input, got nil")
	}
}

func TestProcessParams_MissingParam(t *testing.T) {
	ds := DummyStruct{}
	args := map[string]string{ // "score" is missing
		"name":   "Alice",
		"age":    "30",
		"active": "true",
	}
	_, err := processParams(&ds, args)
	if err == nil {
		t.Fatal("expected error for missing parameter, got nil")
	}
	if !errors.As(err, &ParamError{}) {
		t.Errorf("expected error message to mention missing 'score', got: %v", err)
	}
}

type UnsupportedStruct struct {
	Numbers []int `param:"numbers"`
}

func TestProcessParams_UnsupportedType(t *testing.T) {
	us := UnsupportedStruct{}
	args := map[string]string{
		"numbers": "1,2,3",
	}
	_, err := processParams(&us, args)
	if err == nil {
		t.Fatal("expected error for unsupported type, got nil")
	}
	if !errors.As(err, &DirectiveFieldError{}) {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestSetVal_String(t *testing.T) {
	var s string
	v := reflect.ValueOf(&s).Elem()
	err := setVal(v, "hello", "s")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if s != "hello" {
		t.Errorf("expected s = 'hello', got %q", s)
	}
}

func TestSetVal_Int(t *testing.T) {
	var i int
	v := reflect.ValueOf(&i).Elem()
	err := setVal(v, "42", "i")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if i != 42 {
		t.Errorf("expected i = 42, got %d", i)
	}
}

func TestSetVal_Float64(t *testing.T) {
	var f float64
	v := reflect.ValueOf(&f).Elem()
	err := setVal(v, "3.14", "f")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if f != 3.14 {
		t.Errorf("expected f = 3.14, got %f", f)
	}
}

func TestSetVal_Bool(t *testing.T) {
	var b bool
	v := reflect.ValueOf(&b).Elem()
	err := setVal(v, "true", "b")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if b != true {
		t.Errorf("expected b = true, got %v", b)
	}
}

func TestSetVal_InvalidConversion(t *testing.T) {
	var i int
	v := reflect.ValueOf(&i).Elem()
	err := setVal(v, "not_an_int", "i")
	if err == nil {
		t.Fatal("expected error for invalid int conversion, got nil")
	}
	if !errors.As(err, &ConversionError{}) {
		t.Errorf("unexpected error message: %v", err)
	}
}
