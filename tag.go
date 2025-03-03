package taggart

import (
	"fmt"
	"reflect"
	"strings"
)

type Tag struct {
	Key string
	//Directives []Directive
}

//func (t *Tag) Add(directive Directive) error {
//	t.Directives = append(t.Directives, directive)
//	return nil
//}

type AnyDirectiveHandler interface {
	HandleReflect(val reflect.Value) error
	//Type() reflect.Type
}

type TypeErasedHandler[T any] struct {
	handler func(val T) error
}

func (h *TypeErasedHandler[T]) HandleReflect(val reflect.Value) error {
	if !val.CanInterface() {
		return fmt.Errorf("cannot access field value")
	}

	//if !val.Type().AssignableTo(reflect.TypeOf((*T)(nil)).Elem()) {
	if !val.Type().AssignableTo(reflect.TypeFor[T]()) { // type assertion
		return fmt.Errorf("type mismatch: expected %v, got %v",
			//reflect.TypeOf((*T)(nil)).Elem(), val.Type())
			reflect.TypeFor[T](), val.Type())
	}

	typedVal, ok := val.Interface().(T) // convert val to T
	if !ok {
		return fmt.Errorf("type assertion failed")
	}

	return h.handler(typedVal)
}

//func (h *TypeErasedHandler[T]) Type() reflect.Type {
//	return reflect.TypeOf((*T)(nil)).Elem()
//	//return reflect.TypeFor[T]()
//}

type Directive struct {
	Name    string
	Handler AnyDirectiveHandler
}

func NewDirective[T any](name string, handler func(val T) error) Directive {
	return Directive{
		Name: name,
		Handler: &TypeErasedHandler[T]{
			handler: handler,
		},
	}
}

var directiveRegistry = map[string]AnyDirectiveHandler{}

func RegisterDirective(d Directive) {
	directiveRegistry[d.Name] = d.Handler
}

//type DirectiveHandleFunc[T any] func(val T) error
//
//func (h DirectiveHandleFunc[T]) Handle(val T) error {
//	return h(val)
//}
//func (h DirectiveHandleFunc[T]) Type() reflect.Type {
//	return reflect.TypeFor[T]()
//}
//
//type DirectiveHandler[T any] interface {
//	Handle(val T) error
//	Type() reflect.Type
//}
//
//type Directive[T any] struct {
//	Name    string
//	Handler DirectiveHandler[T]
//}
//
//var directiveRegistry = map[string]DirectiveHandler{}
//
//func RegisterDirective(d Directive) {
//	directiveRegistry[d.Name] = d.Handler
//}

func IsOfType(val reflect.Value, expectedType reflect.Type) bool {
	if !val.IsValid() {
		return false
	}
	return val.Type() == expectedType
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

func processTag(tagValue string, fieldValue reflect.Value) error {
	directive, _ := splitTagValue(tagValue) // TODO: handle args
	handler := directiveRegistry[directive]
	if handler == nil {
		return fmt.Errorf("unknown directive: %s", directive)
	}
	//if !IsOfType(fieldValue, handler.Type()) {
	//	return fmt.Errorf("directive %q handels cannot handle field type %q", directive, fieldValue.Type())
	//}
	if err := handler.HandleReflect(fieldValue); err != nil {
		return fmt.Errorf("directive %q failed: %w", directive, err)
	}
	return nil
}

func ProcessStruct(tag Tag, data interface{}) (bool, error) {
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
			err = processTag(tagValue, fieldValue)
			if err != nil {
				return false, fmt.Errorf("error validating field %q: %v", field.Name, err)
			}
		}
	}
	return true, nil
}
