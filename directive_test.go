package tagex

import (
	"errors"
	"fmt"
	"reflect"
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
	if !errors.As(err, &ParamError{}) {
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
	if !errors.As(err, &DirectiveError{}) {
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
	mode DirectiveMode
}

func (d *dummyDirective) Name() string {
	return d.name
}

func (d *dummyDirective) Mode() DirectiveMode {
	return d.mode
}

func (d *dummyDirective) Handle(val int) (int, error) {
	if val != 42 {
		return val, fmt.Errorf("dummyDirective: expected 42, got %d", val)
	}
	return val, nil
}

func TestDirectiveWrapper_HandleAny_Success(t *testing.T) {
	dd := &dummyDirective{
		name: "dummy",
		mode: EvalMode,
	}
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
	if !errors.As(err, &DirectiveError{}) {
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
	if !errors.As(err, &HandleError{}) {
		t.Errorf("unexpected error message: %v", err)
	}
}

type embedded struct {
	Inner string
}

type targetStruct struct {
	Str        string
	Num        int
	Ptr        *string
	Iface      any
	Embed      embedded
	unexported string
}

func Test_valSet(t *testing.T) {
	testStr := "updated"
	tests := []struct {
		name      string
		fieldName string
		value     any
		newValue  any
		wantErr   bool
		check     func(ts targetStruct) bool
	}{
		{
			name:      "primitive string",
			fieldName: "Str",
			value:     "initial",
			newValue:  "updated",
			wantErr:   false,
			check:     func(ts targetStruct) bool { return ts.Str == "updated" },
		},
		{
			name:      "primitive int",
			fieldName: "Num",
			value:     42,
			newValue:  99,
			wantErr:   false,
			check:     func(ts targetStruct) bool { return ts.Num == 99 },
		},
		{
			name:      "pointer to string",
			fieldName: "Ptr",
			value:     nil,
			newValue:  &testStr,
			wantErr:   false,
			check:     func(ts targetStruct) bool { return ts.Ptr != nil && *ts.Ptr == "updated" },
		},
		{
			name:      "interface value",
			fieldName: "Iface",
			value:     nil,
			newValue:  123,
			wantErr:   false,
			check:     func(ts targetStruct) bool { return ts.Iface == 123 },
		},
		{
			name:      "embedded struct",
			fieldName: "Embed",
			value:     embedded{Inner: "old"},
			newValue:  embedded{Inner: "new"},
			wantErr:   false,
			check:     func(ts targetStruct) bool { return ts.Embed.Inner == "new" },
		},
		{
			name:      "unexported field (cannot set)",
			fieldName: "unexported",
			value:     "hidden",
			newValue:  "newval",
			wantErr:   true,
			check:     func(ts targetStruct) bool { return ts.unexported == "hidden" },
		},
		{
			name:      "mismatched type",
			fieldName: "Str",
			value:     "initial",
			newValue:  999,
			wantErr:   true,
			check:     func(ts targetStruct) bool { return ts.Str == "initial" },
		},

		{
			name:      "nil to string field",
			fieldName: "Str",
			value:     "initial",
			newValue:  nil,
			wantErr:   true,
			check:     func(ts targetStruct) bool { return ts.Str == "initial" },
		},

		{
			name:      "int32 to int",
			fieldName: "Num",
			value:     42,
			newValue:  int32(99),
			wantErr:   true,
			check:     func(ts targetStruct) bool { return ts.Num == 42 },
		},
		{
			name:      "string to []byte (convertible but not assignable)",
			fieldName: "Iface",
			value:     nil,
			newValue:  []byte("hi"),
			wantErr:   false,
			check:     func(ts targetStruct) bool { return reflect.DeepEqual(ts.Iface, []byte("hi")) },
		},
		{
			name:      "pointer mismatch: string pointer to string field",
			fieldName: "Str",
			value:     "initial",
			newValue:  &testStr,
			wantErr:   true,
			check:     func(ts targetStruct) bool { return ts.Str == "initial" },
		},
		{
			name:      "struct pointer to interface",
			fieldName: "Iface",
			value:     nil,
			newValue:  &embedded{Inner: "hi"},
			wantErr:   false, // Should succeed — interfaces accept pointers too
			check: func(ts targetStruct) bool {
				emb, ok := ts.Iface.(*embedded)
				return ok && emb.Inner == "hi"
			},
		},
		{
			name:      "primitive assigned to embedded struct",
			fieldName: "Embed",
			value:     embedded{Inner: "old"},
			newValue:  42,
			wantErr:   true,
			check:     func(ts targetStruct) bool { return ts.Embed.Inner == "old" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := targetStruct{
				Str:        "initial",
				Num:        42,
				unexported: "hidden",
				Embed:      embedded{Inner: "old"},
			}

			val := reflect.ValueOf(&ts).Elem().FieldByName(tt.fieldName)
			if !val.IsValid() {
				t.Fatalf("invalid field: %s", tt.fieldName)
			}

			err := callValSetWithGeneric(tt.newValue, val)
			if tt.wantErr && err == nil {
				t.Errorf("expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.check(ts) {
				t.Errorf("value not set as expected for field %s", tt.fieldName)
			}
		})
	}
}

func callValSetWithGeneric(input any, val reflect.Value) error {
	switch v := input.(type) {
	case string:
		return valSet[string](val, v)
	case int:
		return valSet[int](val, v)
	case int32:
		return valSet[int32](val, v)
	case *string:
		return valSet[*string](val, v)
	case embedded:
		return valSet[embedded](val, v)
	case any:
		return valSet[any](val, v)
	default:
		return errors.New("unsupported test type")
	}
}
