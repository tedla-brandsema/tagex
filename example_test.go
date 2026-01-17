package tagex

import (
	"fmt"
	"reflect"
	"strconv"
)

type rangeDirectiveExample struct {
	Min int `param:"min"`
	Max int `param:"max"`
}

func (d *rangeDirectiveExample) Name() string { return "range" }
func (d *rangeDirectiveExample) Mode() DirectiveMode { return EvalMode }
func (d *rangeDirectiveExample) Handle(val int) (int, error) {
	if val < d.Min || val > d.Max {
		return val, fmt.Errorf("value %d out of range", val)
	}
	return val, nil
}

type auditDirectiveExample struct{}

func (d *auditDirectiveExample) Name() string { return "audit" }
func (d *auditDirectiveExample) Mode() DirectiveMode { return EvalMode }
func (d *auditDirectiveExample) Handle(val string) (string, error) { return val, nil }

type recordExample struct {
	Name       string `check:"audit"`
	PreCalled  bool
	PostCalled bool
}

func (r *recordExample) Before() error {
	r.PreCalled = true
	return nil
}

func (r *recordExample) After() error {
	r.PostCalled = true
	return nil
}

type multiplyDirectiveExample struct {
	Factor int64 `param:"factor"`
}

func (d *multiplyDirectiveExample) Name() string { return "mul" }
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
	fmt.Println(rec.PreCalled, rec.PostCalled)
	// Output: true true
}

func Example_customConverter() {
	calcTag := NewTag("calc")
	calcTag.SetConverter(reflect.Int64, func(v reflect.Value, raw string) error {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return ConversionError{Msg: fmt.Sprintf("unable to convert value %q to int64", raw)}
		}
		v.SetInt(n)
		return nil
	})
	RegisterDirective(&calcTag, &multiplyDirectiveExample{})

	type Payload struct {
		Count int64 `calc:"mul, factor=3"`
	}

	p := Payload{Count: 5}
	_, _ = calcTag.ProcessStruct(&p)
	fmt.Println(p.Count)
	// Output: 15
}
