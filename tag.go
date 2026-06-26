package tagex

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

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
func NewTag(key string) *Tag {
	return &Tag{Key: key}
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

// distinctTags returns tags with duplicate pointers removed, preserving order.
func distinctTags(tags []*Tag) []*Tag {
	out := make([]*Tag, 0, len(tags))
	for _, tag := range tags {
		seen := false
		for _, kept := range out {
			if kept == tag {
				seen = true
				break
			}
		}
		if !seen {
			out = append(out, tag)
		}
	}
	return out
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
// to the provided struct pointer. It returns nil on success.
func (t *Tag) ProcessStruct(data any) error {
	return ProcessStruct(data, t)
}

// ProcessStruct applies directives for multiple tags in a single pass.
// It returns nil on success.
func ProcessStruct(data any, tags ...*Tag) error {
	val, err := pointerStruct(data)
	if err != nil {
		return err
	}

	// Lock and process each distinct Tag once. The same *Tag passed twice
	// would otherwise take a recursive read lock (which Go prohibits and can
	// deadlock) and run its directives twice. Guarded by len > 1 to keep the
	// common single-tag path allocation-free.
	if len(tags) > 1 {
		tags = distinctTags(tags)
	}

	for _, tag := range tags {
		if tag == nil {
			return fmt.Errorf("nil tag provided")
		}
		tag.mut.RLock()
		defer tag.mut.RUnlock()
	}

	// Pre-processing
	if err = InvokePreProcessor(data); err != nil {
		return &ProcessError{
			Stage: StagePre,
			Cause: &HookError{Hook: "Before", Err: err},
		}
	}

	// Process directives
	if err = processStructFields(tags, val, ""); err != nil {
		cause := err
		if err = InvokeFailurePostProcessor(data, cause); err != nil {
			return &ProcessError{
				Stage: StagePost,
				Cause: &HookError{Hook: "Failure", Err: err, Cause: cause},
			}
		}
		return cause
	}

	// Post-processing
	if err = InvokeSuccessPostProcessor(data); err != nil {
		return &ProcessError{
			Stage: StagePost,
			Cause: &HookError{Hook: "Success", Err: err},
		}
	}

	return nil
}

// RegisterDirective registers a Directive with a Tag.
// Directives are looked up by name when processing struct fields.
func RegisterDirective[T any](t *Tag, d Directive[T]) {
	t.setDirective(d.Name(), directiveWrapper[T]{Directive: d})
}
