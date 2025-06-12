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

// InvokePreProcessor returns true if v implements PreProcessor
// and false if it does not.
func InvokePreProcessor(v any) (bool, error) {
	if p, ok := v.(PreProcessor); ok {
		return true, p.Before()
	}
	return false, nil
}

// InvokePostProcessor returns true if v implements PostProcessor
// and false if it does not.
func InvokePostProcessor(v any) (bool, error) {
	if p, ok := v.(PostProcessor); ok {
		return true, p.After()
	}
	return false, nil
}

type Tag struct {
	Key      string
	mut      sync.RWMutex
	registry map[string]anyDirective
}

func (t *Tag) initRegistry() {
	if t.registry == nil {
		t.registry = make(map[string]anyDirective)
	}
}

func (t *Tag) setDirective(name string, d anyDirective) {
	t.mut.Lock()
	defer t.mut.Unlock()

	t.initRegistry()
	t.registry[name] = d
}

func (t *Tag) get(name string) (anyDirective, bool) {
	t.mut.RLock()
	defer t.mut.RUnlock()

	t.initRegistry()
	d, ok := t.registry[name]
	return d, ok
}

func (t *Tag) ProcessStruct(data any) (bool, error) {
	t.mut.RLock()
	defer t.mut.RUnlock()

	var err error

	val := reflect.ValueOf(data)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return false, fmt.Errorf("expected a pointer to a struct but got %T", data)
	}
	val = val.Elem()

	// Pre-processing
	if _, err = InvokePreProcessor(data); err != nil {
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
	if _, err = InvokePostProcessor(data); err != nil {
		return false, PostProcessingError{Err: err}
	}

	return true, nil
}

func NewTag(key string) Tag {
	return Tag{
		Key: key,
	}
}

func RegisterDirective[T any](t *Tag, d Directive[T]) {
	t.setDirective(d.Name(), directiveWrapper[T]{Directive: d})
}
