package tagex

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type ValImplTest struct {
	Number int    `val:"range, min=0, max=3"`
	Word   string `val:"length, min=2, max=5"`
}

type MultiplyDirective struct {
	Factor int `param:"factor"`
}

func (d *MultiplyDirective) Name() string {
	return "mul"
}

func (d *MultiplyDirective) Mode() DirectiveMode {
	return MutMode
}

func (d *MultiplyDirective) Handle(val int) (int, error) {
	return val * d.Factor, nil
}

type AddDirective struct {
	Addend int `param:"addend"`
}

func (d *AddDirective) Name() string {
	return "add"
}

func (d *AddDirective) Mode() DirectiveMode {
	return MutMode
}

func (d *AddDirective) Handle(val int) (int, error) {
	return val + d.Addend, nil
}

type SumDirective struct {
	Addends []int `param:"addends"`
}

func (d *SumDirective) Name() string {
	return "sum"
}

func (d *SumDirective) Mode() DirectiveMode {
	return MutMode
}

func (d *SumDirective) Handle(val int) (int, error) {
	total := val
	for _, addend := range d.Addends {
		total += addend
	}
	return total, nil
}

func (d *SumDirective) ConvertParam(field reflect.StructField, fieldValue reflect.Value, raw string) error {
	if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.Int {
		parts := strings.Split(raw, "|")
		addends := make([]int, 0, len(parts))
		for _, part := range parts {
			value := strings.TrimSpace(part)
			if value == "" {
				return NewConversionError(field, raw, "[]int")
			}
			num, err := strconv.Atoi(value)
			if err != nil {
				return NewConversionError(field, raw, "[]int")
			}
			addends = append(addends, num)
		}
		fieldValue.Set(reflect.ValueOf(addends))
		return nil
	}

	return DefaultConvert(fieldValue, raw, field.Tag.Get(paramKey))
}

var _ PreProcessor = (*PrePostTestStruct)(nil)
var _ SuccessPostProcessor = (*PrePostTestStruct)(nil)
var _ FailurePostProcessor = (*PrePostTestStruct)(nil)

type PrePostTestStruct struct {
	ValImplTest
	BeforeCalled  bool
	SuccessCalled bool
	FailureCalled bool
	FailureCause  error
}

func (p *PrePostTestStruct) Before() error {
	p.BeforeCalled = true
	return nil
}

func (p *PrePostTestStruct) Success() error {
	p.SuccessCalled = true
	return nil
}

func (p *PrePostTestStruct) Failure(cause error) error {
	p.FailureCalled = true
	p.FailureCause = cause
	return nil
}

var _ PreProcessor = (*FailingPreProcessor)(nil)

type FailingPreProcessor struct {
	ValImplTest
}

func (f *FailingPreProcessor) Before() error {
	return errors.New("preprocessor failed")
}

var _ SuccessPostProcessor = (*FailingPostProcessor)(nil)

type FailingPostProcessor struct {
	ValImplTest
}

func (f *FailingPostProcessor) Success() error {
	return errors.New("postprocessor failed")
}

var _ FailurePostProcessor = (*FailingFailurePostProcessor)(nil)

type FailingFailurePostProcessor struct {
	ValImplTest
}

func (f *FailingFailurePostProcessor) Failure(cause error) error {
	return errors.New("failure postprocessor failed")
}

func TestNewTag(t *testing.T) {
	tag := NewTag(valTagKey)
	if tag.Key != valTagKey {
		t.Errorf("expected key %q, got %q", valTagKey, tag.Key)
	}
}

func TestSetAndGetDirective(t *testing.T) {
	tag := NewTag(valTagKey)
	MustRegisterDirective(tag, &RangeDirective{})

	expect := "range"
	got, ok := tag.directive(expect)
	if !ok {
		t.Fatalf("expected directive %s to be registered", expect)
	}
	if got == nil {
		t.Fatal("got nil directive")
	}
}

func TestRegisterDirective_EmptyName(t *testing.T) {
	tag := NewTag(valTagKey)
	for _, name := range []string{"", "   "} {
		err := RegisterDirective(tag, &dummyDirective{name: name})
		var empty *EmptyDirectiveNameError
		if !errors.As(err, &empty) {
			t.Fatalf("name %q: expected EmptyDirectiveNameError, got %v", name, err)
		}
	}
}

func TestRegisterDirective_Duplicate(t *testing.T) {
	tag := NewTag(valTagKey)
	if err := RegisterDirective(tag, &RangeDirective{}); err != nil {
		t.Fatalf("first registration: %v", err)
	}
	err := RegisterDirective(tag, &RangeDirective{})
	var dup *DuplicateDirectiveError
	if !errors.As(err, &dup) {
		t.Fatalf("expected DuplicateDirectiveError, got %v", err)
	}
	if dup.Name != "range" {
		t.Errorf("expected duplicate name %q, got %q", "range", dup.Name)
	}
}

func TestMustRegisterDirective_PanicsOnDuplicate(t *testing.T) {
	tag := NewTag(valTagKey)
	MustRegisterDirective(tag, &RangeDirective{})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()
	MustRegisterDirective(tag, &RangeDirective{})
}

func TestProcessStruct_InvalidInput(t *testing.T) {
	tag := NewTag(valTagKey)

	err := tag.ProcessStruct("not-a-struct")
	if err == nil {
		t.Fatal("expected error for non-struct input")
	}
	var procErr *ProcessError
	if !errors.As(err, &procErr) {
		t.Fatalf("expected *ProcessError, got %v", err)
	}
	var target *InvalidTargetError
	if !errors.As(err, &target) {
		t.Fatalf("expected *InvalidTargetError cause, got %v", err)
	}
}

func TestProcessStruct_Success(t *testing.T) {
	tag := NewTag(valTagKey)
	MustRegisterDirective(tag, &RangeDirective{})
	MustRegisterDirective(tag, &LengthDirective{})

	ts := ValImplTest{
		Number: 2,
		Word:   "tagex",
	}
	err := tag.ProcessStruct(&ts)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestProcessStruct_MultipleTags(t *testing.T) {
	valTag := NewTag(valTagKey)
	MustRegisterDirective(valTag, &RangeDirective{})

	mulTag := NewTag("mul")
	MustRegisterDirective(mulTag, &MultiplyDirective{})

	type multiTagged struct {
		Number int `val:"range, min=0, max=5" mul:"mul, factor=2"`
	}

	ts := multiTagged{Number: 2}
	err := ProcessStruct(&ts, valTag, mulTag)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ts.Number != 4 {
		t.Fatalf("expected Number to be 4, got %d", ts.Number)
	}
}

func TestProcessStruct_MultipleTags_NilTag(t *testing.T) {
	valTag := NewTag(valTagKey)
	MustRegisterDirective(valTag, &RangeDirective{})

	type multiTagged struct {
		Number int `val:"range, min=0, max=5"`
	}

	ts := multiTagged{Number: 2}
	err := ProcessStruct(&ts, valTag, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var procErr *ProcessError
	if !errors.As(err, &procErr) {
		t.Fatalf("expected *ProcessError, got %v", err)
	}
	var nilTag *NilTagError
	if !errors.As(err, &nilTag) {
		t.Fatalf("expected *NilTagError cause, got %v", err)
	}
}

func TestProcessStruct_ParamConverter_Success(t *testing.T) {
	tag := NewTag("sum")
	MustRegisterDirective(tag, &SumDirective{})

	type target struct {
		Count int `sum:"sum, addends=1|2|3"`
	}

	ts := target{Count: 10}
	err := tag.ProcessStruct(&ts)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ts.Count != 16 {
		t.Fatalf("expected Count to be 16, got %d", ts.Count)
	}
}

func TestProcessStruct_ParamConverter_Error(t *testing.T) {
	tag := NewTag("sum")
	MustRegisterDirective(tag, &SumDirective{})

	type target struct {
		Count int `sum:"sum, addends=1|bad|3"`
	}

	ts := target{Count: 10}
	err := tag.ProcessStruct(&ts)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var procErr *ProcessError
	if !errors.As(err, &procErr) {
		t.Fatalf("expected ProcessError, got: %v", err)
	}
	if procErr.Param != "addends" {
		t.Fatalf("expected ProcessError.Param to be \"addends\", got %q", procErr.Param)
	}
}

func TestProcessStruct_MultipleTags_Order(t *testing.T) {
	addTag := NewTag("add")
	MustRegisterDirective(addTag, &AddDirective{})

	mulTag := NewTag("mul")
	MustRegisterDirective(mulTag, &MultiplyDirective{})

	type multiTagged struct {
		Number int `add:"add, addend=3" mul:"mul, factor=2"`
	}

	ts := multiTagged{Number: 2}
	err := ProcessStruct(&ts, addTag, mulTag)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if ts.Number != 10 {
		t.Fatalf("expected Number to be 10, got %d", ts.Number)
	}
}

func TestProcessStruct_MultipleTags_ErrorWrap(t *testing.T) {
	valTag := NewTag(valTagKey)
	MustRegisterDirective(valTag, &RangeDirective{})

	badTag := NewTag("bad")

	type multiTagged struct {
		Number int `val:"range, min=0, max=3" bad:"missing"`
	}

	ts := multiTagged{Number: 2}
	err := ProcessStruct(&ts, valTag, badTag)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var tagErr *TagError
	if !errors.As(err, &tagErr) {
		t.Fatalf("expected TagError, got: %v", err)
	}
	if tagErr.TagKey != "bad" {
		t.Fatalf("expected TagError tag key to be \"bad\", got %q", tagErr.TagKey)
	}
	var procErr *ProcessError
	if !errors.As(err, &procErr) {
		t.Fatalf("expected ProcessError, got: %v", err)
	}
}

func TestProcessStruct_Failure(t *testing.T) {
	tag := NewTag(valTagKey)

	ts := ValImplTest{
		Number: 2,
	}
	err := tag.ProcessStruct(&ts)
	if err == nil {
		t.Fatal("expected error due to missing directive")
	}
	var unknownErr *UnknownDirectiveError
	if !errors.As(err, &unknownErr) {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestProcessStruct_ParamContext_Missing(t *testing.T) {
	tag := NewTag(valTagKey)
	MustRegisterDirective(tag, &RangeDirective{})

	type target struct {
		Number int `val:"range, max=3"`
	}

	ts := target{Number: 2}
	err := tag.ProcessStruct(&ts)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var procErr *ProcessError
	if !errors.As(err, &procErr) {
		t.Fatalf("expected ProcessError, got: %v", err)
	}
	if procErr.Param != "min" {
		t.Fatalf("expected ProcessError.Param to be \"min\", got %q", procErr.Param)
	}
}

func TestProcessStruct_ParamContext_Conversion(t *testing.T) {
	tag := NewTag(valTagKey)
	MustRegisterDirective(tag, &RangeDirective{})

	type target struct {
		Number int `val:"range, min=bad, max=3"`
	}

	ts := target{Number: 2}
	err := tag.ProcessStruct(&ts)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var procErr *ProcessError
	if !errors.As(err, &procErr) {
		t.Fatalf("expected ProcessError, got: %v", err)
	}
	if procErr.Param != "min" {
		t.Fatalf("expected ProcessError.Param to be \"min\", got %q", procErr.Param)
	}
}

func TestProcessStruct_PreProcessor(t *testing.T) {
	tag := NewTag(valTagKey)
	MustRegisterDirective(tag, &RangeDirective{})
	MustRegisterDirective(tag, &LengthDirective{})
	ts := PrePostTestStruct{
		ValImplTest: ValImplTest{Number: 2, Word: "test"},
	}
	err := tag.ProcessStruct(&ts)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !ts.BeforeCalled {
		t.Fatal("expected Before() to be called")
	}
}

func TestProcessStruct_PostProcessor(t *testing.T) {
	tag := NewTag(valTagKey)
	MustRegisterDirective(tag, &RangeDirective{})
	MustRegisterDirective(tag, &LengthDirective{})
	ts := PrePostTestStruct{
		ValImplTest: ValImplTest{Number: 2, Word: "test"},
	}
	err := tag.ProcessStruct(&ts)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !ts.SuccessCalled {
		t.Fatal("expected Success() to be called")
	}
}

func TestProcessStruct_PreProcessor_Failure(t *testing.T) {
	tag := NewTag(valTagKey)
	ts := FailingPreProcessor{
		ValImplTest: ValImplTest{Number: 2, Word: "test"},
	}

	err := tag.ProcessStruct(&ts)
	if err == nil {
		t.Fatal("expected error from Before()")
	}
	var hookErr *HookError
	if !errors.As(err, &hookErr) {
		t.Fatalf("expected HookError, got: %v", err)
	}
	if hookErr.Err == nil || hookErr.Err.Error() != "preprocessor failed" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestProcessStruct_PostProcessor_Failure(t *testing.T) {
	tag := NewTag(valTagKey)
	MustRegisterDirective(tag, &RangeDirective{})
	MustRegisterDirective(tag, &LengthDirective{})
	ts := FailingPostProcessor{
		ValImplTest: ValImplTest{Number: 2, Word: "test"},
	}

	err := tag.ProcessStruct(&ts)
	if err == nil {
		t.Fatal("expected error from Success()")
	}
	var hookErr *HookError
	if !errors.As(err, &hookErr) {
		t.Fatalf("expected HookError, got: %v", err)
	}
	if hookErr.Err == nil || hookErr.Err.Error() != "postprocessor failed" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestProcessStruct_FailurePostProcessor_Called(t *testing.T) {
	tag := NewTag(valTagKey)
	ts := PrePostTestStruct{
		ValImplTest: ValImplTest{Number: 2, Word: "failure"},
	}

	err := tag.ProcessStruct(&ts)
	if err == nil {
		t.Fatal("expected error from directive processing")
	}
	if !ts.FailureCalled {
		t.Fatal("expected Failure() to be called")
	}
	if ts.FailureCause == nil {
		t.Fatal("expected Failure() to receive cause error")
	}
}

func TestProcessStruct_FailurePostProcessor_Error(t *testing.T) {
	tag := NewTag(valTagKey)
	MustRegisterDirective(tag, &RangeDirective{})
	MustRegisterDirective(tag, &LengthDirective{})
	ts := FailingFailurePostProcessor{
		ValImplTest: ValImplTest{Number: 5, Word: "failure"},
	}

	err := tag.ProcessStruct(&ts)
	if err == nil {
		t.Fatal("expected error from Failure()")
	}
	var hookErr *HookError
	if !errors.As(err, &hookErr) {
		t.Fatalf("expected HookError, got: %v", err)
	}
	if hookErr.Err == nil || hookErr.Err.Error() != "failure postprocessor failed" {
		t.Fatalf("unexpected error message: %v", err)
	}
	if hookErr.Cause == nil {
		t.Fatalf("expected Failure() hook error to include cause, got: %v", err)
	}
	var tagErr *TagError
	if !errors.As(hookErr.Cause, &tagErr) {
		t.Fatalf("expected HookError.Cause to include TagError, got: %v", hookErr.Cause)
	}
}

func TestProcessStruct_Recursion_EmbeddedAndNamed(t *testing.T) {
	tag := NewTag(valTagKey)
	MustRegisterDirective(tag, &RangeDirective{})
	MustRegisterDirective(tag, &LengthDirective{})

	type embedded struct {
		Number int `val:"range, min=0, max=3"`
	}
	type named struct {
		Word string `val:"length, min=2, max=5"`
	}
	type outer struct {
		embedded
		Named named
	}

	ts := outer{
		embedded: embedded{Number: 2},
		Named:    named{Word: "test"},
	}
	err := tag.ProcessStruct(&ts)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestProcessStruct_Recursion_NilPointerSkipped(t *testing.T) {
	tag := NewTag(valTagKey)
	MustRegisterDirective(tag, &RangeDirective{})

	type inner struct {
		Number int `val:"range, min=0, max=3"`
	}
	type outer struct {
		Ptr *inner
	}

	ts := outer{}
	err := tag.ProcessStruct(&ts)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestProcessStruct_Recursion_UnexportedSkipped(t *testing.T) {
	tag := NewTag(valTagKey)
	MustRegisterDirective(tag, &RangeDirective{})

	type outer struct {
		inner struct {
			Number int `val:"range, min=0, max=3"`
		}
	}

	ts := outer{
		inner: struct {
			Number int `val:"range, min=0, max=3"`
		}{Number: 10},
	}
	err := tag.ProcessStruct(&ts)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestProcessStruct_Recursion_SliceOfStructs_Error(t *testing.T) {
	tag := NewTag(valTagKey)
	MustRegisterDirective(tag, &RangeDirective{})

	type item struct {
		N int `val:"range, min=0, max=5"`
	}
	type outer struct {
		Items []item
	}

	ts := outer{Items: []item{{N: 2}, {N: 9}}}
	err := tag.ProcessStruct(&ts)
	if err == nil {
		t.Fatal("expected error from out-of-range element")
	}
	var procErr *ProcessError
	if !errors.As(err, &procErr) {
		t.Fatalf("expected ProcessError, got: %v", err)
	}
	if procErr.FieldPath != "Items[1].N" {
		t.Fatalf("expected field path %q, got %q", "Items[1].N", procErr.FieldPath)
	}
}

func TestProcessStruct_Recursion_SliceOfStructs_Mutates(t *testing.T) {
	tag := NewTag("mul")
	MustRegisterDirective(tag, &MultiplyDirective{})

	type item struct {
		N int `mul:"mul, factor=2"`
	}
	type outer struct {
		Items []item
	}

	ts := outer{Items: []item{{N: 5}, {N: 7}}}
	if err := tag.ProcessStruct(&ts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts.Items[0].N != 10 || ts.Items[1].N != 14 {
		t.Fatalf("expected [10 14], got [%d %d]", ts.Items[0].N, ts.Items[1].N)
	}
}

func TestProcessStruct_Recursion_ArrayAndPointers_Mutates(t *testing.T) {
	tag := NewTag("mul")
	MustRegisterDirective(tag, &MultiplyDirective{})

	type item struct {
		N int `mul:"mul, factor=2"`
	}
	type outer struct {
		Arr [2]item
		Ptr []*item
	}

	ts := outer{Arr: [2]item{{N: 3}, {N: 4}}, Ptr: []*item{{N: 5}}}
	if err := tag.ProcessStruct(&ts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ts.Arr[0].N != 6 || ts.Arr[1].N != 8 {
		t.Fatalf("array: expected [6 8], got [%d %d]", ts.Arr[0].N, ts.Arr[1].N)
	}
	if ts.Ptr[0].N != 10 {
		t.Fatalf("ptr slice: expected 10, got %d", ts.Ptr[0].N)
	}
}

func TestProcessStruct_Recursion_MapOfStructs_Mutates(t *testing.T) {
	tag := NewTag("mul")
	MustRegisterDirective(tag, &MultiplyDirective{})

	type item struct {
		N int `mul:"mul, factor=2"`
	}
	type outer struct {
		ByID map[string]item
	}

	ts := outer{ByID: map[string]item{"a": {N: 5}}}
	if err := tag.ProcessStruct(&ts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The map value is not addressable; the copy-back is what makes this work.
	if got := ts.ByID["a"].N; got != 10 {
		t.Fatalf("expected map value mutated to 10, got %d", got)
	}
}

func TestProcessStruct_Recursion_CycleReturnsError(t *testing.T) {
	tag := NewTag(valTagKey)
	MustRegisterDirective(tag, &RangeDirective{})

	type Node struct {
		N    int `val:"range, min=0, max=10"`
		Next *Node
	}
	n := &Node{N: 5}
	n.Next = n // self-cycle

	err := tag.ProcessStruct(n)

	// Same handhold as every other processing failure: the ProcessError
	// envelope carries context, and the *MaxDepthError is the cause.
	var procErr *ProcessError
	if !errors.As(err, &procErr) {
		t.Fatalf("expected *ProcessError envelope on cyclic data, got %v", err)
	}
	if procErr.FieldPath == "" {
		t.Error("expected a FieldPath locating the cycle")
	}
	if len(procErr.FieldPath) > 200 {
		t.Errorf("FieldPath should be truncated, got %d chars", len(procErr.FieldPath))
	}
	var depthErr *MaxDepthError
	if !errors.As(err, &depthErr) {
		t.Fatalf("expected *MaxDepthError cause, got %v", err)
	}
}

func TestProcessStruct_Recursion_DeepAcyclicOK(t *testing.T) {
	tag := NewTag(valTagKey)
	MustRegisterDirective(tag, &RangeDirective{})

	type Node struct {
		N    int `val:"range, min=0, max=10"`
		Next *Node
	}
	// A 100-deep acyclic chain — well under maxDepth, must not false-trigger.
	head := &Node{N: 1}
	cur := head
	for i := 0; i < 100; i++ {
		cur.Next = &Node{N: 1}
		cur = cur.Next
	}
	if err := tag.ProcessStruct(head); err != nil {
		t.Fatalf("deep acyclic chain should pass, got %v", err)
	}
}

func TestProcessStruct_Recursion_MapOfStructs_ErrorPath(t *testing.T) {
	tag := NewTag(valTagKey)
	MustRegisterDirective(tag, &RangeDirective{})

	type item struct {
		N int `val:"range, min=0, max=5"`
	}
	type outer struct {
		ByID map[string]item
	}

	ts := outer{ByID: map[string]item{"bad": {N: 9}}}
	err := tag.ProcessStruct(&ts)
	if err == nil {
		t.Fatal("expected error from out-of-range map value")
	}
	var procErr *ProcessError
	if !errors.As(err, &procErr) {
		t.Fatalf("expected ProcessError, got: %v", err)
	}
	if procErr.FieldPath != "ByID[bad].N" {
		t.Fatalf("expected field path %q, got %q", "ByID[bad].N", procErr.FieldPath)
	}
}
