package tagex

import (
	"errors"
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
	RegisterDirective(&tag, &RangeDirective{})

	expect := "range"
	got, ok := tag.directive(expect)
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
	if !strings.Contains(err.Error(), "expected a pointer to a struct") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestProcessStruct_Success(t *testing.T) {
	tag := NewTag(valTagKey)
	RegisterDirective(&tag, &RangeDirective{})
	RegisterDirective(&tag, &LengthDirective{})

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

func TestProcessStruct_MultipleTags(t *testing.T) {
	valTag := NewTag(valTagKey)
	RegisterDirective(&valTag, &RangeDirective{})

	mulTag := NewTag("mul")
	RegisterDirective(&mulTag, &MultiplyDirective{})

	type multiTagged struct {
		Number int `val:"range, min=0, max=5" mul:"mul, factor=2"`
	}

	ts := multiTagged{Number: 2}
	ok, err := ProcessStruct(&ts, &valTag, &mulTag)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !ok {
		t.Fatal("expected ok to be true")
	}
	if ts.Number != 4 {
		t.Fatalf("expected Number to be 4, got %d", ts.Number)
	}
}

func TestProcessStruct_Failure(t *testing.T) {
	tag := NewTag(valTagKey)

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
	var unknownErr *UnknownDirectiveError
	if !errors.As(err, &unknownErr) {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestProcessStruct_PreProcessor(t *testing.T) {
	tag := NewTag(valTagKey)
	RegisterDirective(&tag, &RangeDirective{})
	RegisterDirective(&tag, &LengthDirective{})
	ts := PrePostTestStruct{
		ValImplTest: ValImplTest{Number: 2, Word: "test"},
	}
	ok, err := tag.ProcessStruct(&ts)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !ok {
		t.Fatal("expected ok to be true")
	}
	if !ts.BeforeCalled {
		t.Fatal("expected Before() to be called")
	}
}

func TestProcessStruct_PostProcessor(t *testing.T) {
	tag := NewTag(valTagKey)
	RegisterDirective(&tag, &RangeDirective{})
	RegisterDirective(&tag, &LengthDirective{})
	ts := PrePostTestStruct{
		ValImplTest: ValImplTest{Number: 2, Word: "test"},
	}
	ok, err := tag.ProcessStruct(&ts)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !ok {
		t.Fatal("expected ok to be true")
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

	ok, err := tag.ProcessStruct(&ts)
	if err == nil {
		t.Fatal("expected error from Before()")
	}
	if ok {
		t.Fatal("expected ok to be false")
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
	RegisterDirective(&tag, &RangeDirective{})
	RegisterDirective(&tag, &LengthDirective{})
	ts := FailingPostProcessor{
		ValImplTest: ValImplTest{Number: 2, Word: "test"},
	}

	ok, err := tag.ProcessStruct(&ts)
	if err == nil {
		t.Fatal("expected error from Success()")
	}
	if ok {
		t.Fatal("expected ok to be false")
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

	ok, err := tag.ProcessStruct(&ts)
	if err == nil {
		t.Fatal("expected error from directive processing")
	}
	if ok {
		t.Fatal("expected ok to be false")
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
	ts := FailingFailurePostProcessor{
		ValImplTest: ValImplTest{Number: 5, Word: "failure"},
	}

	ok, err := tag.ProcessStruct(&ts)
	if err == nil {
		t.Fatal("expected error from Failure()")
	}
	if ok {
		t.Fatal("expected ok to be false")
	}
	var hookErr *HookError
	if !errors.As(err, &hookErr) {
		t.Fatalf("expected HookError, got: %v", err)
	}
	if hookErr.Err == nil || hookErr.Err.Error() != "failure postprocessor failed" {
		t.Fatalf("unexpected error message: %v", err)
	}
}
