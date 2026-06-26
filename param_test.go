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
	err := ProcessParams(&ds, args)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
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
	err := ProcessParams(ds, map[string]string{})
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
	err := ProcessParams(&ds, args)
	if err == nil {
		t.Fatal("expected error for missing parameter, got nil")
	}
	var missingErr *MissingParamError
	if !errors.As(err, &missingErr) {
		t.Errorf("expected error message to mention missing 'score', got: %v", err)
	}
}

func TestProcessParams_Public_Success(t *testing.T) {
	ds := DummyStruct{}
	args := map[string]string{
		"name":   "Alice",
		"age":    "30",
		"score":  "95.5",
		"active": "true",
	}
	if err := ProcessParams(&ds, args); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestProcessParams_Public_Error(t *testing.T) {
	ds := DummyStruct{}
	args := map[string]string{
		"name":   "Alice",
		"age":    "30",
		"active": "true",
	}
	err := ProcessParams(&ds, args)
	if err == nil {
		t.Fatal("expected error for missing parameter, got nil")
	}
	var missingErr *MissingParamError
	if !errors.As(err, &missingErr) {
		t.Fatalf("expected MissingParamError, got: %v", err)
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
	err := ProcessParams(&us, args)
	if err == nil {
		t.Fatal("expected error for unsupported type, got nil")
	}
	var typeErr *UnsupportedParamTypeError
	if !errors.As(err, &typeErr) {
		t.Errorf("unexpected error message: %v", err)
	}
}

type OptionalStruct struct {
	Name  string `param:"name, required=false"`
	Count int    `param:"count, default=5"`
}

func TestProcessParams_RequiredFalse_SkipsMissing(t *testing.T) {
	os := OptionalStruct{}
	err := ProcessParams(&os, map[string]string{})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if os.Name != "" {
		t.Errorf("expected Name to remain empty, got %q", os.Name)
	}
	if os.Count != 5 {
		t.Errorf("expected Count to be 5 from default, got %d", os.Count)
	}
}

func TestProcessParams_DefaultOverridesMissing(t *testing.T) {
	os := OptionalStruct{}
	args := map[string]string{
		"name": "Bob",
	}
	err := ProcessParams(&os, args)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if os.Name != "Bob" {
		t.Errorf("expected Name 'Bob', got %q", os.Name)
	}
	if os.Count != 5 {
		t.Errorf("expected Count to be 5 from default, got %d", os.Count)
	}
}

func TestProcessParams_RequiredFalse_UsesProvided(t *testing.T) {
	os := OptionalStruct{}
	args := map[string]string{
		"name": "Alice",
	}
	err := ProcessParams(&os, args)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if os.Name != "Alice" {
		t.Errorf("expected Name 'Alice', got %q", os.Name)
	}
}

func TestProcessParams_Default_UsesProvided(t *testing.T) {
	os := OptionalStruct{}
	args := map[string]string{
		"count": "7",
	}
	err := ProcessParams(&os, args)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if os.Count != 7 {
		t.Errorf("expected Count to be 7 from provided value, got %d", os.Count)
	}
}

type ConflictingStruct struct {
	Count int `param:"count, required=true, default=5"`
}

func TestProcessParams_RequiredAndDefaultConflict(t *testing.T) {
	cs := ConflictingStruct{}
	err := ProcessParams(&cs, map[string]string{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var conflictErr *ParamConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("expected ParamConflictError, got: %v", err)
	}
}

type BadRequiredStruct struct {
	Name string `param:"name, required=maybe"`
}

func TestProcessParams_RequiredParseError(t *testing.T) {
	br := BadRequiredStruct{}
	err := ProcessParams(&br, map[string]string{"name": "Alice"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var convErr *ConversionError
	if !errors.As(err, &convErr) {
		t.Fatalf("expected ConversionError, got: %v", err)
	}
}

func TestSetVal_String(t *testing.T) {
	var s string
	v := reflect.ValueOf(&s).Elem()
	err := DefaultConvert(v, "hello", "value")
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
	err := DefaultConvert(v, "42", "value")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if i != 42 {
		t.Errorf("expected i = 42, got %d", i)
	}
}

func TestSetVal_Int64(t *testing.T) {
	var i int64
	v := reflect.ValueOf(&i).Elem()
	err := DefaultConvert(v, "123", "value")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if i != 123 {
		t.Errorf("expected i = 123, got %d", i)
	}
}

func TestSetVal_Float64(t *testing.T) {
	var f float64
	v := reflect.ValueOf(&f).Elem()
	err := DefaultConvert(v, "3.14", "value")
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
	err := DefaultConvert(v, "true", "value")
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
	err := DefaultConvert(v, "not_an_int", "value")
	if err == nil {
		t.Fatal("expected error for invalid int conversion, got nil")
	}
	var convErr *ConversionError
	if !errors.As(err, &convErr) {
		t.Errorf("unexpected error message: %v", err)
	}
}
