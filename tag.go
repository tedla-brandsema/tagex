package taggart

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

type Tag struct {
	Key      string
	mut      sync.RWMutex
	registry map[string]AnyDirectiveHandler
}

func (t *Tag) initRegistry() {
	if t.registry == nil {
		t.registry = make(map[string]AnyDirectiveHandler)
	}
}

func (t *Tag) Register(d Directive) {
	t.mut.Lock()
	defer t.mut.Unlock()

	t.initRegistry()
	t.registry[d.Name] = d.Handler
}

func (t *Tag) Get(directiveName string) AnyDirectiveHandler {
	t.mut.RLock()
	defer t.mut.RUnlock()

	t.initRegistry()
	return t.registry[directiveName]
}

type AnyDirectiveHandler interface {
	HandleReflect(val reflect.Value, args []string) error
}

type TypeErasedHandler[T any] struct {
	handler func(val T, args []string) error
}

func (h *TypeErasedHandler[T]) HandleReflect(val reflect.Value, args []string) error {
	if !val.CanInterface() {
		return fmt.Errorf("cannot access field value")
	}

	if !val.Type().AssignableTo(reflect.TypeFor[T]()) { // type assertion
		return fmt.Errorf("type mismatch: expected %v, got %v",
			reflect.TypeFor[T](), val.Type())
	}

	typedVal, ok := val.Interface().(T) // convert val to T
	if !ok {
		return fmt.Errorf("type assertion failed")
	}

	return h.handler(typedVal, args)
}

type Directive struct {
	Name    string
	Handler AnyDirectiveHandler
}

func NewDirective[T any](name string, handler func(val T, args []string) error) Directive {
	return Directive{
		Name: name,
		Handler: &TypeErasedHandler[T]{
			handler: handler,
		},
	}
}

func splitTagValue(tagVal string) (id string, args []string) {
	parts := strings.Split(tagVal, ",")
	if len(parts) == 0 {
		return "", nil
	}
	id = strings.TrimSpace(parts[0])
	if len(parts) > 1 {
		args = parts[1:]
		// TODO process args into k,v pairs
	}
	return id, args
}

func processTag(tag *Tag, tagValue string, fieldValue reflect.Value) error {
	directive, args := splitTagValue(tagValue)
	handler := tag.Get(directive)
	if handler == nil {
		return fmt.Errorf("unknown directive: %s", directive)
	}
	if err := handler.HandleReflect(fieldValue, args); err != nil {
		return fmt.Errorf("directive %q failed: %w", directive, err)
	}
	return nil
}

func ProcessStruct(tag *Tag, data interface{}) (bool, error) {
	var err error

	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return false, fmt.Errorf("expected a struct but got %T", data)
	}

	for n := 0; n < val.NumField(); n++ {
		field := val.Type().Field(n)
		if tagValue, ok := field.Tag.Lookup(tag.Key); ok {
			fieldValue := val.FieldByName(field.Name)

			err = processTag(tag, tagValue, fieldValue)
			if err != nil {
				return false, fmt.Errorf("error validating field %q: %v", field.Name, err)
			}
		}
	}
	return true, nil
}
