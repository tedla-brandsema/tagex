package tagex

import (
	"errors"
	"fmt"
	"reflect"
)

// ParamConverter allows a directive to control how its parameters
// are parsed from raw tag values.
type ParamConverter interface {
	ConvertParam(
		field reflect.StructField,
		fieldValue reflect.Value,
		raw string,
	) error
}

// DirectiveMode defines how a directive handles its value.
type DirectiveMode int

const (
	// EvalMode evaluates the field and does not mutate its value.
	EvalMode DirectiveMode = iota
	// MutMode evaluates the field and writes the returned value back.
	MutMode
)

// Directive defines a semantic operation that can be applied to a
// struct field of type T.
type Directive[T any] interface {
	Name() string
	Mode() DirectiveMode
	Handle(val T) (T, error)
}

type anyDirective interface {
	HandleAny(val reflect.Value) error
	Unwrap() any
	clone() anyDirective
}

type directiveWrapper[T any] struct {
	Directive[T]
}

func (dw directiveWrapper[T]) Unwrap() any {
	return dw.Directive
}

// clone returns a fresh copy of the wrapped directive. The registered directive
// is only a template; per-call parameter state is written to the copy so that
// concurrent ProcessStruct calls on a shared Tag never race on its fields.
func (dw directiveWrapper[T]) clone() anyDirective {
	src := reflect.ValueOf(dw.Directive)
	if src.Kind() != reflect.Ptr || src.IsNil() {
		return dw // value directives carry no settable param state to copy
	}
	dup := reflect.New(src.Elem().Type())
	dup.Elem().Set(src.Elem())
	return directiveWrapper[T]{Directive: dup.Interface().(Directive[T])}
}

func (dw directiveWrapper[T]) HandleAny(val reflect.Value) (err error) {
	t, err := valParse[T](val)
	if err != nil {
		return err
	}

	t, err = dw.Handle(t)
	if err != nil {
		return &HandleError{Nested: err}
	}

	if dw.Mode() == MutMode {
		return valSet(val, t)
	}

	return nil
}

func valSet[T any](val reflect.Value, t T) (err error) {
	if !val.CanSet() {
		return &FieldSetError{Msg: "unable to set field value"}
	}

	defer func() {
		if r := recover(); r != nil {
			err = &FieldSetError{Msg: fmt.Sprintf("failed to set field value: %v", r)}
		}
	}()
	val.Set(reflect.ValueOf(t))

	return nil
}

func valParse[T any](val reflect.Value) (T, error) {
	var zero T
	if !val.CanInterface() {
		return zero, &FieldAccessError{Msg: "cannot access field value"}
	}

	if err := valTypeAssert[T](val); err != nil {
		return zero, err
	}

	t, ok := val.Interface().(T)
	if !ok {
		return zero, &TypeMismatchError{Expected: val.Type(), Got: reflect.TypeFor[T]()}
	}
	return t, nil
}

func valTypeAssert[T any](val reflect.Value) error {
	t := reflect.TypeFor[T]()
	if t.AssignableTo(val.Type()) {
		return nil
	}
	return &TypeMismatchError{Expected: val.Type(), Got: t}
}

// processDirective applies every directive in tagValue to fieldValue. Directives
// are chained with ';' and run left-to-right; each MutMode segment's written-back
// value is what the next segment reads, so order is significant
// ("trim;length, min=3" differs from "length, min=3;trim"). Processing stops at
// the first failing segment and returns its error. Note that under
// ProcessStructAll a MutMode segment that already ran has still mutated the field
// even when a later segment in the same chain fails.
func processDirective(tag *Tag, tagValue string, fieldValue reflect.Value) error {
	for _, seg := range splitChain(tagValue) {
		if err := processSegment(tag, seg, fieldValue); err != nil {
			return err
		}
	}
	return nil
}

// processSegment applies a single directive segment ("name, k=v, ...") to
// fieldValue: it parses the directive name and args, runs the directive on a
// per-call copy, and (in MutMode) writes the result back to fieldValue.
func processSegment(tag *Tag, tagValue string, fieldValue reflect.Value) error {
	var err error

	directiveName, args, err := splitTagValue(tagValue)
	if err != nil {
		stage := StageDirective
		var paramErr *ParamParseError
		if errors.As(err, &paramErr) {
			stage = StageParam
		}
		return &ProcessError{
			Stage:     stage,
			Directive: directiveName,
			Cause:     err,
		}
	}
	template, ok := tag.directive(directiveName)
	if !ok {
		return &ProcessError{
			Stage:     StageDirective,
			Directive: directiveName,
			Cause:     &UnknownDirectiveError{Name: directiveName},
		}
	}
	directive := template.clone() // per-call copy; never mutate the shared template
	err = ProcessParams(directive.Unwrap(), args)
	if err != nil {
		param := ""
		var missingErr *MissingParamError
		if errors.As(err, &missingErr) {
			param = missingErr.Param
		}
		var convErr *ConversionError
		if errors.As(err, &convErr) && param == "" {
			param = convErr.Param
		}
		return &ProcessError{
			Stage:     StageParam,
			Directive: directiveName,
			Param:     param,
			Cause:     err,
		}
	}
	err = directive.HandleAny(fieldValue)
	if err != nil {
		return &ProcessError{
			Stage:     StageDirective,
			Directive: directiveName,
			Cause:     err,
		}
	}
	return nil
}
