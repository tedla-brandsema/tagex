package tagex

import (
	"fmt"
	"reflect"
)

type Stage string

const (
	StagePre       Stage = "pre"
	StageDirective Stage = "directive"
	StageParam     Stage = "param"
	StagePost      Stage = "post"
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
	case StagePre:
		prefix = "pre-processing"
	case StageDirective:
		prefix = "directive processing"
	case StageParam:
		prefix = "parameter processing"
	case StagePost:
		prefix = "post-processing"
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

type UnknownDirectiveError struct {
	Name string
}

func (e *UnknownDirectiveError) Error() string {
	return fmt.Sprintf("unknown directive %q", e.Name)
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
		Param:  field.Tag.Get(ParamKey),
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
