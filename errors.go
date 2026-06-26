package tagex

import (
	"fmt"
	"reflect"
)

type Stage string

const (
	StageInput     Stage = "input"
	StagePre       Stage = "pre"
	StageDirective Stage = "directive"
	StageParam     Stage = "param"
	StagePost      Stage = "post"
	StageStruct    Stage = "struct"
)

type ProcessError struct {
	Stage     Stage
	FieldPath string
	Directive string
	Param     string
	Cause     error
}

func (e *ProcessError) Error() string {
	if e == nil {
		return "<nil>"
	}

	prefix := "processing"
	switch e.Stage {
	case StageInput:
		prefix = "input validation"
	case StagePre:
		prefix = "pre-processing"
	case StageDirective:
		prefix = "directive processing"
	case StageParam:
		prefix = "parameter processing"
	case StagePost:
		prefix = "post-processing"
	case StageStruct:
		prefix = "struct processing"
	}

	msg := prefix
	if e.FieldPath != "" {
		msg += fmt.Sprintf(" field %q", e.FieldPath)
	}
	if e.Directive != "" {
		msg += fmt.Sprintf(" directive %q", e.Directive)
	}
	if e.Param != "" {
		msg += fmt.Sprintf(" param %q", e.Param)
	}
	if e.Cause != nil {
		msg += ": " + e.Cause.Error()
	}
	return msg
}

func (e *ProcessError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

type TagError struct {
	TagKey string
	Err    error
}

func (e *TagError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Err == nil {
		return fmt.Sprintf("tag %q error", e.TagKey)
	}
	return fmt.Sprintf("tag %q error: %v", e.TagKey, e.Err)
}

func (e *TagError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type HookError struct {
	Hook  string
	Cause error
	Err   error
}

func (e *HookError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Cause != nil {
		return fmt.Sprintf("%s hook failed: %v (cause: %v)", e.Hook, e.Err, e.Cause)
	}
	return fmt.Sprintf("%s hook failed: %v", e.Hook, e.Err)
}

func (e *HookError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// HandleError marks an error returned by a directive's Handle method — a value
// the directive rejected — as opposed to a framework failure such as a type
// mismatch or an unsettable field. Both occur at StageDirective, so this type is
// how callers tell a domain/validation failure apart from a wiring bug, via
// errors.As. The rejected value's error is available through Unwrap.
type HandleError struct {
	Nested error
}

func (e *HandleError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Nested == nil {
		return "directive handle failed"
	}
	return e.Nested.Error()
}

func (e *HandleError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Nested
}

type UnknownDirectiveError struct {
	Name string
}

func (e *UnknownDirectiveError) Error() string {
	return fmt.Sprintf("unknown directive %q", e.Name)
}

type EmptyDirectiveNameError struct{}

func (e *EmptyDirectiveNameError) Error() string {
	return "directive name must not be empty"
}

type DuplicateDirectiveError struct {
	Name string
}

func (e *DuplicateDirectiveError) Error() string {
	return fmt.Sprintf("directive %q is already registered", e.Name)
}

// MaxDepthError reports that processing recursed past the nesting limit, which
// usually means the data is cyclic (a value that reaches itself through a
// pointer, slice, or map). Like other processing failures it is wrapped in a
// *ProcessError, whose FieldPath locates where the limit was hit.
type MaxDepthError struct {
	Limit int
}

func (e *MaxDepthError) Error() string {
	return fmt.Sprintf("maximum nesting depth %d exceeded (possible cycle)", e.Limit)
}

// InvalidTargetError reports that the value passed to ProcessStruct was not a
// pointer to a struct. Got is its concrete type.
type InvalidTargetError struct {
	Got string
}

func (e *InvalidTargetError) Error() string {
	return fmt.Sprintf("expected a pointer to a struct but got %s", e.Got)
}

// NilTagError reports that a nil *Tag was passed to ProcessStruct.
type NilTagError struct{}

func (e *NilTagError) Error() string {
	return "nil tag provided"
}

type DirectiveParseError struct {
	TagValue string
}

func (e *DirectiveParseError) Error() string {
	return "directive name is required"
}

type ParamParseError struct {
	Pair string
}

func (e *ParamParseError) Error() string {
	return fmt.Sprintf("malformed key value pair %q, expected format is \"key=value\"", e.Pair)
}

type MissingParamError struct {
	Param string
}

func (e *MissingParamError) Error() string {
	return fmt.Sprintf("%q parameter not set", e.Param)
}

type TypeMismatchError struct {
	Expected reflect.Type
	Got      reflect.Type
}

func (e *TypeMismatchError) Error() string {
	return fmt.Sprintf("type mismatch: expected %v, got %v", e.Expected, e.Got)
}

type ConversionError struct {
	Param  string
	Raw    string
	Target string
}

func (e *ConversionError) Error() string {
	return fmt.Sprintf(msg, e.Raw, e.Target)
}

func NewConversionError(field reflect.StructField, raw string, target string) *ConversionError {
	return &ConversionError{
		Param:  field.Tag.Get(paramKey),
		Raw:    raw,
		Target: target,
	}
}

type FieldAccessError struct {
	Msg string
}

func (e *FieldAccessError) Error() string {
	return e.Msg
}

type FieldSetError struct {
	Msg string
}

func (e *FieldSetError) Error() string {
	return e.Msg
}

type UnsupportedParamTypeError struct {
	Type reflect.Kind
}

func (e *UnsupportedParamTypeError) Error() string {
	return fmt.Sprintf("unsupported param type %s", e.Type)
}

type ParamConflictError struct {
	Param string
}

func (e *ParamConflictError) Error() string {
	return fmt.Sprintf("%q param cannot set both required and default", e.Param)
}
