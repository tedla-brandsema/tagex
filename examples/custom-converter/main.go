// Custom-converter shows a directive that implements ParamConverter to accept a
// parameter type the default converter doesn't handle (here, a []int).
package main

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/tedla-brandsema/tagex"
)

// SumDirective adds a list of addends (parsed from "a|b|c") to an int field.
type SumDirective struct {
	Addends []int `param:"addends"`
}

func (d *SumDirective) Name() string              { return "sum" }
func (d *SumDirective) Mode() tagex.DirectiveMode { return tagex.MutMode }

func (d *SumDirective) Handle(val int) (int, error) {
	total := val
	for _, addend := range d.Addends {
		total += addend
	}
	return total, nil
}

// ConvertParam replaces the default converter for every param on this directive.
// It handles the []int field itself and falls back to tagex.DefaultConvert for
// any primitive param.
func (d *SumDirective) ConvertParam(field reflect.StructField, fieldValue reflect.Value, raw string) error {
	if field.Type.Kind() != reflect.Slice || field.Type.Elem().Kind() != reflect.Int {
		return tagex.DefaultConvert(fieldValue, raw, field.Tag.Get("param"))
	}
	parts := strings.Split(raw, "|")
	addends := make([]int, 0, len(parts))
	for _, part := range parts {
		num, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return tagex.NewConversionError(field, raw, "[]int")
		}
		addends = append(addends, num)
	}
	fieldValue.Set(reflect.ValueOf(addends))
	return nil
}

func main() {
	checkTag := tagex.NewTag("check")
	tagex.RegisterDirective(checkTag, &SumDirective{})

	type Item struct {
		Count int `check:"sum, addends=1|2|3"`
	}

	item := Item{Count: 10}
	if err := checkTag.ProcessStruct(&item); err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("count:", item.Count)
	// Output:
	// count: 16
}
