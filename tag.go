package tagex

import (
	"fmt"
	"reflect"
	"sync"
)

type PreProcessingError struct {
	Err error
}

func (e PreProcessingError) Error() string {
	if e.Err == nil {
		return "pre-processing error"
	}
	return e.Err.Error()
}

func (e PreProcessingError) Unwrap() error {
	return e.Err
}

type PreProcessor interface {
	Before() error
}

type PostProcessingError struct {
	Err error
}

func (e PostProcessingError) Error() string {
	if e.Err == nil {
		return "post-processing error"
	}
	return e.Err.Error()
}

func (e PostProcessingError) Unwrap() error {
	return e.Err
}

type PostProcessor interface {
	After() error
}

func InvokePreProcessor(v any) error {
	_, err := pointerStruct(v)
	if err != nil {
		return err
	}

	if err = invokePreProcessor(v); err != nil {
		return PreProcessingError{Err: err}
	}

	return nil
}

func invokePreProcessor(v any) error {
	if p, ok := v.(PreProcessor); ok {
		return p.Before()
	}
	return nil
}

func InvokePostProcessor(v any) error {
	_, err := pointerStruct(v)
	if err != nil {
		return err
	}

	if err = invokePostProcessor(v); err != nil {
		return PostProcessingError{Err: err}
	}

	return nil
}

func invokePostProcessor(v any) error {
	p, ok := v.(PostProcessor)
	if ok {
		return p.After()
	}
	return nil
}

type Tag struct {
	Key               string
	mut               sync.RWMutex
	directiveRegistry map[string]anyDirective
	converterRegistry map[reflect.Kind]Converter
}

func NewTag(key string) Tag {
	return Tag{
		Key:               key,
		converterRegistry: defaultConverters(),
	}
}

func (t *Tag) initDirectiveRegistry() {
	if t.directiveRegistry == nil {
		t.directiveRegistry = make(map[string]anyDirective)
	}
}

func (t *Tag) setDirective(name string, d anyDirective) {
	t.mut.Lock()
	defer t.mut.Unlock()

	t.initDirectiveRegistry()
	t.directiveRegistry[name] = d
}

func (t *Tag) directive(name string) (anyDirective, bool) {
	t.mut.RLock()
	defer t.mut.RUnlock()

	t.initDirectiveRegistry()
	d, ok := t.directiveRegistry[name]
	return d, ok
}

func (t *Tag) SetConverter(kind reflect.Kind, converter Converter) {
	t.mut.Lock()
	defer t.mut.Unlock()

	t.converterRegistry[kind] = converter
}

func (t *Tag) converter(kind reflect.Kind) (Converter, bool) {
	t.mut.RLock()
	defer t.mut.RUnlock()

	c, ok := t.converterRegistry[kind]
	return c, ok
}

func pointerStruct(v any) (reflect.Value, error) {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return val, fmt.Errorf("expected a pointer to a struct but got %T", v)
	}
	return val.Elem(), nil
}

func (t *Tag) ProcessStruct(data any) (bool, error) {
	t.mut.RLock()
	defer t.mut.RUnlock()

	var err error

	val, err := pointerStruct(data)
	if err != nil {
		return false, err
	}

	// Pre-processing
	if err = invokePreProcessor(data); err != nil {
		return false, PreProcessingError{Err: err}
	}

	// Process directives
	for n := 0; n < val.NumField(); n++ {
		field := val.Type().Field(n)
		if tagValue, ok := field.Tag.Lookup(t.Key); ok {
			fieldValue := val.FieldByName(field.Name)

			err = processDirective(t, tagValue, fieldValue)
			if err != nil {
				return false, fmt.Errorf("error processing field %q: %w", field.Name, err)
			}
		}
	}

	// Post-processing
	if err = invokePostProcessor(data); err != nil {
		return false, PostProcessingError{Err: err}
	}

	return true, nil
}

func RegisterDirective[T any](t *Tag, d Directive[T]) {
	t.setDirective(d.Name(), directiveWrapper[T]{Directive: d})
}
