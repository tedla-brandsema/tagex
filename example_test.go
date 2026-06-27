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
	MustRegisterDirective(checkTag, &rangeDirectiveExample{})

	type Car struct {
		Doors int `check:"range, min=2, max=4"`
	}

	car := Car{Doors: 4}
	err := checkTag.ProcessStruct(&car)
	fmt.Println(err == nil)
	// Output: true
}

func Example_prePostProcessing() {
	checkTag := NewTag("check")
	MustRegisterDirective(checkTag, &auditDirectiveExample{})

	rec := recordExample{Name: "ok"}
	_ = checkTag.ProcessStruct(&rec)
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

	return DefaultConvert(fieldValue, raw, field.Tag.Get(paramKey))
}

func Example_paramConverter() {
	checkTag := NewTag("check")
	MustRegisterDirective(checkTag, &sumDirectiveExample{})

	type Item struct {
		Count int `check:"sum, addends=1|2|3"`
	}

	item := Item{Count: 10}
	_ = checkTag.ProcessStruct(&item)
	fmt.Println(item.Count)
	// Output: 16
}

type trimExample struct{}

func (d *trimExample) Name() string        { return "trim" }
func (d *trimExample) Mode() DirectiveMode { return MutMode }
func (d *trimExample) Handle(val string) (string, error) {
	return strings.TrimSpace(val), nil
}

type upperExample struct{}

func (d *upperExample) Name() string        { return "upper" }
func (d *upperExample) Mode() DirectiveMode { return MutMode }
func (d *upperExample) Handle(val string) (string, error) {
	return strings.ToUpper(val), nil
}

// Example_chainedDirectives applies two directives to one field by separating
// them with ';'. They run left to right — trim first, then upper — each MutMode
// result feeding the next.
func Example_chainedDirectives() {
	cleanTag := NewTag("clean")
	MustRegisterDirective(cleanTag, &trimExample{})
	MustRegisterDirective(cleanTag, &upperExample{})

	type User struct {
		Name string `clean:"trim;upper"`
	}

	user := User{Name: "  ada  "}
	_ = cleanTag.ProcessStruct(&user)
	fmt.Printf("%q\n", user.Name)
	// Output: "ADA"
}

type prefixExample struct {
	With string `param:"with"`
}

func (d *prefixExample) Name() string        { return "prefix" }
func (d *prefixExample) Mode() DirectiveMode { return MutMode }
func (d *prefixExample) Handle(val string) (string, error) {
	return d.With + val, nil
}

// Example_quotedValue wraps a parameter value in single quotes so it can hold the
// reserved characters ',' and ';' (and trailing whitespace) literally, instead
// of them being read as separators.
func Example_quotedValue() {
	renderTag := NewTag("render")
	MustRegisterDirective(renderTag, &prefixExample{})

	type Line struct {
		Text string `render:"prefix, with='[a, b; c] '"`
	}

	line := Line{Text: "hello"}
	_ = renderTag.ProcessStruct(&line)
	fmt.Println(line.Text)
	// Output: [a, b; c] hello
}
