package tagex

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type DirectiveHandler[T any] interface {
	Handle(val T) error
}

type Directive[T any] interface {
	Name() string
	DirectiveHandler[T]
}

type AnyDirective interface {
	HandleAny(val reflect.Value) error
	Unwrap() any
}

type DirectiveWrapper[T any] struct {
	Directive[T]
}

func (dw DirectiveWrapper[T]) Unwrap() any {
	return dw.Directive
}

func (dw DirectiveWrapper[T]) HandleAny(val reflect.Value) error {
	v, err := valParse[T](val)
	if err != nil {
		return err
	}
	return dw.Handle(v)
}

func valParse[T any](val reflect.Value) (T, error) {
	var zero T
	if !val.CanInterface() {
		return zero, fmt.Errorf("cannot access field value")
	}

	if !val.Type().AssignableTo(reflect.TypeFor[T]()) { // type assertion
		return zero, fmt.Errorf("type mismatch: expected %v, got %v", reflect.TypeFor[T](), val.Type())
	}

	typedVal, ok := val.Interface().(T) // convert val to T
	if !ok {
		return zero, fmt.Errorf("type assertion failed")
	}
	return typedVal, nil
}

func processDirective(tag *Tag, tagValue string, fieldValue reflect.Value) error {
	var err error

	directiveName, args, err := splitTagValue(tagValue)
	if err != nil {
		return err
	}
	directive, ok := tag.get(directiveName)
	if !ok {
		return fmt.Errorf("unknown directive %q", directiveName)
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
	return "", "", fmt.Errorf("malformed key value pair %q, expected format is \"key=value\"", strings.TrimSpace(pair))
}

func splitTagValue(tagVal string) (id string, args map[string]string, err error) {
	parts := strings.Split(tagVal, ",")
	if len(parts) == 0 || parts[0] == "" {
		return "", nil, errors.New("no directive set")
	}
	id = strings.TrimSpace(parts[0])
	args, err = extractPairs(parts[1:])
	return id, args, err
}
