package tagex

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type rangeDirectiveExample struct {
	Min int `param:"min"`
	Max int `param:"max"`
}

func (d *rangeDirectiveExample) Name() string        { return "range" }
func (d *rangeDirectiveExample) Mode() DirectiveMode { return EvalMode }
func (d *rangeDirectiveExample) Handle(val int) (int, error) {
	if val < d.Min || val > d.Max {
		return val, fmt.Errorf("value %d out of range", val)
	}
	return val, nil
}

type auditDirectiveExample struct{}

func (d *auditDirectiveExample) Name() string                      { return "audit" }
func (d *auditDirectiveExample) Mode() DirectiveMode               { return EvalMode }
func (d *auditDirectiveExample) Handle(val string) (string, error) { return val, nil }

type recordExample struct {
	Name          string `check:"audit"`
	BeforeCalled  bool
	SuccessCalled bool
}

func (r *recordExample) Before() error {
	r.BeforeCalled = true
	return nil
}

func (r *recordExample) Success() error {
	r.SuccessCalled = true
	return nil
}

type multiplyDirectiveExample struct {
	Factor int64 `param:"factor"`
}

func (d *multiplyDirectiveExample) Name() string        { return "mul" }
func (d *multiplyDirectiveExample) Mode() DirectiveMode { return MutMode }
func (d *multiplyDirectiveExample) Handle(val int64) (int64, error) {
	return val * d.Factor, nil
}

func Example_basic() {
	checkTag := NewTag("check")
	RegisterDirective(&checkTag, &rangeDirectiveExample{})

	type Car struct {
		Doors int `check:"range, min=2, max=4"`
	}

	car := Car{Doors: 4}
	ok, err := checkTag.ProcessStruct(&car)
	fmt.Println(ok, err == nil)
	// Output: true true
}

func Example_prePostProcessing() {
	checkTag := NewTag("check")
	RegisterDirective(&checkTag, &auditDirectiveExample{})

	rec := recordExample{Name: "ok"}
	_, _ = checkTag.ProcessStruct(&rec)
	fmt.Println(rec.BeforeCalled, rec.SuccessCalled)
	// Output: true true
}

type sumDirectiveExample struct {
	Addends []int `param:"addends"`
}

func (d *sumDirectiveExample) Name() string        { return "sum" }
func (d *sumDirectiveExample) Mode() DirectiveMode { return MutMode }
func (d *sumDirectiveExample) Handle(val int) (int, error) {
	total := val
	for _, addend := range d.Addends {
		total += addend
	}
	return total, nil
}

func (d *sumDirectiveExample) ConvertParam(field reflect.StructField, fieldValue reflect.Value, raw string) error {
	if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.Int {
		parts := strings.Split(raw, "|")
		addends := make([]int, 0, len(parts))
		for _, part := range parts {
			value := strings.TrimSpace(part)
			if value == "" {
				return NewConversionError(field, raw, "[]int")
			}
			num, err := strconv.Atoi(value)
			if err != nil {
				return NewConversionError(field, raw, "[]int")
			}
			addends = append(addends, num)
		}
		fieldValue.Set(reflect.ValueOf(addends))
		return nil
	}

	return defaultConvert(fieldValue, raw, field.Tag.Get(paramKey))
}

func Example_paramConverter() {
	checkTag := NewTag("check")
	RegisterDirective(&checkTag, &sumDirectiveExample{})

	type Item struct {
		Count int `check:"sum, addends=1|2|3"`
	}

	item := Item{Count: 10}
	_, _ = checkTag.ProcessStruct(&item)
	fmt.Println(item.Count)
	// Output: 16
}
