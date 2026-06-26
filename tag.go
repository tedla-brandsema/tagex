package tagex

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// ================ Pre Processing ===================

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
		return err
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
		return err
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
		return err
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

	// Reading a nil map is safe; the registry is created lazily by
	// setDirective under the write lock, so no init is needed here.
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

func wrapFieldError(fieldName string, err error) error {
	var pe *ProcessError
	if errors.As(err, &pe) {
		if pe.FieldPath == "" {
			pe.FieldPath = fieldName
		}
		return pe
	}
	return &ProcessError{
		Stage:     StageDirective,
		FieldPath: fieldName,
		Cause:     err,
	}
}

func processStructFields(tags []*Tag, val reflect.Value, path string) error {
	for n := 0; n < val.NumField(); n++ {
		field := val.Type().Field(n)
		if field.PkgPath != "" {
			continue
		}

		fieldValue := val.FieldByName(field.Name)
		for _, tag := range tags {
			if tag == nil {
				continue
			}
			if tagValue, ok := field.Tag.Lookup(tag.Key); ok {
				if err := processDirective(tag, tagValue, fieldValue); err != nil {
					fieldName := field.Name
					if path != "" {
						fieldName = path + "." + field.Name
					}
					return &TagError{
						TagKey: tag.Key,
						Err:    wrapFieldError(fieldName, err),
					}
				}
			}
		}

		switch fieldValue.Kind() {
		case reflect.Struct:
			nextPath := field.Name
			if path != "" {
				nextPath = path + "." + field.Name
			}
			if err := processStructFields(tags, fieldValue, nextPath); err != nil {
				return err
			}
		case reflect.Ptr:
			if fieldValue.IsNil() {
				continue
			}
			elem := fieldValue.Elem()
			if elem.Kind() != reflect.Struct {
				continue
			}
			nextPath := field.Name
			if path != "" {
				nextPath = path + "." + field.Name
			}
			if err := processStructFields(tags, elem, nextPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// ProcessStruct applies all directives associated with the Tag
// to the provided struct pointer.
func (t *Tag) ProcessStruct(data any) (bool, error) {
	return ProcessStruct(data, t)
}

// ProcessStruct applies directives for multiple tags in a single pass.
func ProcessStruct(data any, tags ...*Tag) (bool, error) {
	var err error

	val, err := pointerStruct(data)
	if err != nil {
		return false, err
	}

	for _, tag := range tags {
		if tag == nil {
			return false, fmt.Errorf("nil tag provided")
		}
		tag.mut.RLock()
		defer tag.mut.RUnlock()
	}

	// Pre-processing
	if err = invokePreProcessor(data); err != nil {
		return false, &ProcessError{
			Stage: StagePre,
			Cause: &HookError{Hook: "Before", Err: err},
		}
	}

	// Process directives
	if err = processStructFields(tags, val, ""); err != nil {
		cause := err
		if err = invokeFailurePostProcessor(data, cause); err != nil {
			return false, &ProcessError{
				Stage: StagePost,
				Cause: &HookError{Hook: "Failure", Err: err, Cause: cause},
			}
		}
		return false, cause
	}

	// Post-processing
	if err = invokeSuccessPostProcessor(data); err != nil {
		return false, &ProcessError{
			Stage: StagePost,
			Cause: &HookError{Hook: "Success", Err: err},
		}
	}

	return true, nil
}

// RegisterDirective registers a Directive with a Tag.
// Directives are looked up by name when processing struct fields.
func RegisterDirective[T any](t *Tag, d Directive[T]) {
	t.setDirective(d.Name(), directiveWrapper[T]{Directive: d})
}
