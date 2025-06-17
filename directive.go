package tagex

import (
	"fmt"
	"reflect"
	"strings"
)

type DirectiveError struct {
	Msg string
}

func (e DirectiveError) Error() string {
	return e.Msg
}

type ParamError struct {
	Msg string
}

func (e ParamError) Error() string {
	return e.Msg
}

type FieldError struct {
	Msg string
}

func (e FieldError) Error() string {
	return e.Msg
}

type DirectiveMode int

const (
	EvalMode DirectiveMode = iota
	MutMode
)

type HandleError struct {
	Nested error
}

func (e HandleError) Error() string {
	if e.Nested == nil {
		return ""
	}
	return e.Nested.Error()
}

func (e HandleError) Unwrap() error {
	return e.Nested
}

type Directive[T any] interface {
	Name() string
	Mode() DirectiveMode
	Handle(val T) (T, error)
}

type anyDirective interface {
	HandleAny(val reflect.Value) error
	Unwrap() any
}

type directiveWrapper[T any] struct {
	Directive[T]
}

func (dw directiveWrapper[T]) Unwrap() any {
	return dw.Directive
}

func (dw directiveWrapper[T]) HandleAny(val reflect.Value) (err error) {
	t, err := valParse[T](val)
	if err != nil {
		return err
	}

	t, err = dw.Handle(t)
	if err != nil {
		return HandleError{Nested: err}
	}

	if dw.Mode() == MutMode {
		return valSet(val, t)
	}

	return nil
}

func valSet[T any](val reflect.Value, t T) (err error) {
	if !val.CanSet() {
		return FieldError{Msg: "unable to set field value"}
	}

	defer func() {
		if r := recover(); r != nil {
			err = FieldError{Msg: fmt.Sprintf("failed to set field value: %v", r)}
		}
	}()
	val.Set(reflect.ValueOf(t))

	return nil
}

func valParse[T any](val reflect.Value) (T, error) {
	var zero T
	if !val.CanInterface() {
		return zero, FieldError{Msg: "cannot access field value"}
	}

	if ok, err := valTypeAssert[T](val); !ok {
		return zero, err
	}

	t, ok := val.Interface().(T)
	if !ok {
		return zero, DirectiveError{Msg: "type conversion failed"}
	}
	return t, nil
}

func valTypeAssert[T any](val reflect.Value) (bool, error) {
	t := reflect.TypeFor[T]()
	if t.AssignableTo(val.Type()) {
		return true, nil
	}
	return false, DirectiveError{Msg: fmt.Sprintf("type mismatch: expected %v, got %v", val.Type(), t)}
}

func processDirective(tag *Tag, tagValue string, fieldValue reflect.Value) error {
	var err error

	directiveName, args, err := splitTagValue(tagValue)
	if err != nil {
		return err
	}
	directive, ok := tag.get(directiveName)
	if !ok {
		return DirectiveError{Msg: fmt.Sprintf("unknown directive %q", directiveName)}
	}
	_, err = processParams(directive.Unwrap(), args)
	if err != nil {
		return err
	}
	err = directive.HandleAny(fieldValue)
	if err != nil {
		return fmt.Errorf("directive %q failed: %w", directiveName, err)
	}
	return nil
}

func extractPairs(args []string) (map[string]string, error) {
	pairs := make(map[string]string)
	for _, pair := range args {
		k, v, err := kv(pair)
		if err != nil {
			return nil, err
		}
		pairs[k] = v
	}
	return pairs, nil
}

func kv(pair string) (k string, v string, err error) {
	split := strings.Split(pair, "=")
	if len(split) == 2 {
		k = strings.TrimSpace(split[0])
		v = strings.TrimSpace(split[1])
		if k != "" && v != "" {
			return k, v, nil
		}
	}
	return "", "", ParamError{Msg: fmt.Sprintf("malformed key value pair %q, expected format is \"key=value\"", strings.TrimSpace(pair))}
}

func splitTagValue(tagVal string) (id string, args map[string]string, err error) {
	parts := strings.Split(tagVal, ",")
	if len(parts) == 0 || parts[0] == "" {
		return "", nil, DirectiveError{Msg: "no directive set"}
	}
	id = strings.TrimSpace(parts[0])
	args, err = extractPairs(parts[1:])
	return id, args, err
}
