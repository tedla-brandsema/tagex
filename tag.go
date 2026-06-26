package tagex

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
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

func (t *Tag) setDirective(name string, d anyDirective) error {
	t.mut.Lock()
	defer t.mut.Unlock()

	t.initDirectiveRegistry()
	if _, exists := t.directiveRegistry[name]; exists {
		return &DuplicateDirectiveError{Name: name}
	}
	t.directiveRegistry[name] = d
	return nil
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
		return val, &InvalidTargetError{Got: fmt.Sprintf("%T", v)}
	}
	return val.Elem(), nil
}

// truncatePath shortens a very long field path (produced by a deep cycle) so
// error messages stay readable, cutting on a field boundary.
func truncatePath(p string) string {
	const max = 120
	if len(p) <= max {
		return p
	}
	if i := strings.LastIndexByte(p[:max], '.'); i > 0 {
		return p[:i] + ".…(truncated)"
	}
	return p[:max] + "…(truncated)"
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

// maxDepth bounds recursion so that cyclic data (a struct that reaches itself
// through a pointer, slice, or map) returns a *MaxDepthError instead of
// overflowing the stack. It is far deeper than any real struct nests, so legit
// acyclic data never hits it.
const maxDepth = 1000

func processStructFields(tags []*Tag, val reflect.Value, path string, depth int) error {
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

		if err := processValue(tags, fieldValue, fieldPath, depth+1); err != nil {
			return err
		}
	}

	return nil
}

// processValue descends into val to reach any nested struct fields, recursing
// through pointers, slices, arrays, and maps. Paths gain "[i]" for indexed
// elements and "[key]" for map entries (e.g. Items[2].SKU). depth bounds the
// recursion against cyclic data (see maxDepth).
func processValue(tags []*Tag, val reflect.Value, path string, depth int) error {
	if depth > maxDepth {
		// Wrap like every other processing failure so errors.As(&ProcessError)
		// works uniformly; the *MaxDepthError is the Cause.
		return &ProcessError{
			Stage:     StageStruct,
			FieldPath: truncatePath(path),
			Cause:     &MaxDepthError{Limit: maxDepth},
		}
	}
	switch val.Kind() {
	case reflect.Struct:
		return processStructFields(tags, val, path, depth)
	case reflect.Ptr:
		if val.IsNil() {
			return nil
		}
		return processValue(tags, val.Elem(), path, depth+1)
	case reflect.Slice, reflect.Array:
		for i := 0; i < val.Len(); i++ {
			if err := processValue(tags, val.Index(i), fmt.Sprintf("%s[%d]", path, i), depth+1); err != nil {
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
			if err := processValue(tags, c, fmt.Sprintf("%s[%v]", path, key.Interface()), depth+1); err != nil {
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
		return &ProcessError{Stage: StageInput, Cause: err}
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
			return &ProcessError{Stage: StageInput, Cause: &NilTagError{}}
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
	if err = processStructFields(tags, val, "", 0); err != nil {
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

// RegisterDirective registers d with t under d.Name(); directives are looked up
// by that name when processing struct fields. It returns an *EmptyDirectiveNameError
// if the name is blank, or a *DuplicateDirectiveError if the name is already
// registered on t. Use MustRegisterDirective to panic instead — appropriate for
// registration done once at program startup.
func RegisterDirective[T any](t *Tag, d Directive[T]) error {
	name := d.Name()
	if strings.TrimSpace(name) == "" {
		return &EmptyDirectiveNameError{}
	}
	return t.setDirective(name, directiveWrapper[T]{Directive: d})
}

// MustRegisterDirective is like RegisterDirective but panics if registration
// fails. It is intended for setup-time registration, where a blank or duplicate
// directive name is a programming error that should fail fast at startup.
func MustRegisterDirective[T any](t *Tag, d Directive[T]) {
	if err := RegisterDirective(t, d); err != nil {
		panic(err)
	}
}
