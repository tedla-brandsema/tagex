// Validate runs an EvalMode directive that checks an int field falls within a
// range, without changing the field.
package main

import (
	"fmt"

	"github.com/tedla-brandsema/tagex"
)

// RangeDirective checks that an int field is within [Min, Max].
// Min and Max are filled from the tag args before Handle runs.
type RangeDirective struct {
	Min int `param:"min"`
	Max int `param:"max"`
}

func (d *RangeDirective) Name() string              { return "range" }
func (d *RangeDirective) Mode() tagex.DirectiveMode { return tagex.EvalMode }

func (d *RangeDirective) Handle(val int) (int, error) {
	if val < d.Min || val > d.Max {
		return val, fmt.Errorf("value %d out of range [%d, %d]", val, d.Min, d.Max)
	}
	return val, nil
}

func main() {
	checkTag := tagex.NewTag("check")
	tagex.MustRegisterDirective(checkTag, &RangeDirective{})

	type Car struct {
		Name  string
		Doors int `check:"range, min=2, max=4"`
	}

	cars := []Car{
		{Name: "Citroën 2CV", Doors: 4},
		{Name: "VW Golf", Doors: 5},
	}

	for _, car := range cars {
		if err := checkTag.ProcessStruct(&car); err != nil {
			fmt.Printf("%s failed: %v\n", car.Name, err)
			continue
		}
		fmt.Printf("%s passed\n", car.Name)
	}
	// Output:
	// Citroën 2CV passed
	// VW Golf failed: tag "check" error: directive processing field "Doors" directive "range": value 5 out of range [2, 4]
}
