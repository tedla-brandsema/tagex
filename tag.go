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
		if field.PkgPath != "" { // unexported
			continue
		}

		fieldValue := val.Field(n)
		fieldPath := joinPath(path, field.Name)

		for _, tag := range tags {
			if tag == nil {
				continue
			}
			if tagValue, ok := field.Tag.Lookup(tag.Key); ok {
				if err := processDirective(tag, tagValue, fieldValue); err != nil {
					return &TagError{
						TagKey: tag.Key,
						Err:    wrapFieldError(fieldPath, err),
					}
				}
			}
		}

		if err := processValue(tags, fieldValue, fieldPath); err != nil {
			return err
		}
	}

	return nil
}

// processValue descends into val to reach any nested struct fields, recursing
// through pointers, slices, arrays, and maps. Paths gain "[i]" for indexed
// elements and "[key]" for map entries (e.g. Items[2].SKU).
func processValue(tags []*Tag, val reflect.Value, path string) error {
	switch val.Kind() {
	case reflect.Struct:
		return processStructFields(tags, val, path)
	case reflect.Ptr:
		if val.IsNil() {
			return nil
		}
		return processValue(tags, val.Elem(), path)
	case reflect.Slice, reflect.Array:
		for i := 0; i < val.Len(); i++ {
			if err := processValue(tags, val.Index(i), fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	case reflect.Map:
		for _, key := range val.MapKeys() {
			elem := val.MapIndex(key)
			// Map values are not addressable, so MutMode directives can't write
			// to them in place. Process an addressable copy and store it back.
			// ponytail: copies every value even for EvalMode; revisit only if
			// map-of-struct validation lands on a hot path.
			c := reflect.New(elem.Type()).Elem()
			c.Set(elem)
			if err := processValue(tags, c, fmt.Sprintf("%s[%v]", path, key.Interface())); err != nil {
				return err
			}
			val.SetMapIndex(key, c)
		}
	}

	return nil
}

func joinPath(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
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
