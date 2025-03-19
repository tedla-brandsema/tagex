package tagex

import (
	"fmt"
	"reflect"
	"sync"
)

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
	t.mut.Lock()
	defer t.mut.Unlock()

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
		if tagValue, ok := field.Tag.Lookup(t.Key); ok {
			fieldValue := val.FieldByName(field.Name)

			err = processDirective(t, tagValue, fieldValue)
			if err != nil {
				return false, fmt.Errorf("error processing field %q: %v", field.Name, err)
			}
		}
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
