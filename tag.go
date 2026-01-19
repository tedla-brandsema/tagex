package tagex

import (
	"fmt"
	"reflect"
	"sync"
)

// ================ Pre Processing ===================

type PreProcessingError struct {
	Err error
}

// Error returns the wrapped error message, or a default if the wrapped error is nil.
func (e PreProcessingError) Error() string {
	if e.Err == nil {
		return "pre-processing error"
	}
	return e.Err.Error()
}

// Unwrap exposes the underlying error for errors.Is/errors.As.
func (e PreProcessingError) Unwrap() error {
	return e.Err
}

// PreProcessor runs before Tag.ProcessStruct executes directives.
type PreProcessor interface {
	Before() error
}

// InvokePreProcessor validates v and invokes PreProcessor.Before if implemented.
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

// ================ Post Processing ===================

type PostProcessingError struct {
	Err error
}

// Error returns the wrapped error message, or a default if the wrapped error is nil.
func (e PostProcessingError) Error() string {
	if e.Err == nil {
		return "post-processing error"
	}
	return e.Err.Error()
}

// Unwrap exposes the underlying error for errors.Is/errors.As.
func (e PostProcessingError) Unwrap() error {
	return e.Err
}

// ================ Success Post Processing ===================

// SuccessPostProcessor runs after Tag.ProcessStruct executes directives.
type SuccessPostProcessor interface {
	Success() error
}

// InvokeSuccessPostProcessor validates v and invokes PostProcessor.After if implemented.
func InvokeSuccessPostProcessor(v any) error {
	_, err := pointerStruct(v)
	if err != nil {
		return err
	}

	if err = invokeSuccessPostProcessor(v); err != nil {
		return PostProcessingError{Err: err}
	}

	return nil
}

func invokeSuccessPostProcessor(v any) error {
	p, ok := v.(SuccessPostProcessor)
	if ok {
		return p.Success()
	}
	return nil
}

// ================ Failure Post Processing ===================

// SuccessPostProcessor runs after Tag.ProcessStruct executes directives.
type FailurePostProcessor interface {
	Failure(cause error) error
}

// InvokeSuccessPostProcessor validates v and invokes PostProcessor.After if implemented.
func InvokeFailurePostProcessor(v any, cause error) error {
	_, err := pointerStruct(v)
	if err != nil {
		return err
	}

	if err = invokeFailurePostProcessor(v, cause); err != nil {
		return PostProcessingError{Err: err}
	}

	return nil
}

func invokeFailurePostProcessor(v any, cause error) error {
	p, ok := v.(FailurePostProcessor)
	if ok {
		return p.Failure(cause)
	}
	return nil
}

// Tag represents a processing context for a specific struct tag key.
// It owns the set of directives and converters used when processing
// tagged struct fields.
type Tag struct {
	Key               string
	mut               sync.RWMutex
	directiveRegistry map[string]anyDirective
}

// NewTag creates a new Tag for the given struct tag key.
// The returned Tag is fully initialized with default converters.
func NewTag(key string) Tag {
	return Tag{
		Key: key,
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

func pointerStruct(v any) (reflect.Value, error) {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return val, fmt.Errorf("expected a pointer to a struct but got %T", v)
	}
	return val.Elem(), nil
}

// ProcessStruct applies all directives associated with the Tag
// to the provided struct pointer.
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
				cause := fmt.Errorf("error processing field %q: %w", field.Name, err)
				if err = invokeFailurePostProcessor(data, cause); err != nil {
					return false, PostProcessingError{Err: err} 
				}
				return false, cause
			}
		}
	}

	// Post-processing
	if err = invokeSuccessPostProcessor(data); err != nil {
		return false, PostProcessingError{Err: err}
	}

	return true, nil
}

// RegisterDirective registers a Directive with a Tag.
// Directives are looked up by name when processing struct fields.
func RegisterDirective[T any](t *Tag, d Directive[T]) {
	t.setDirective(d.Name(), directiveWrapper[T]{Directive: d})
}
