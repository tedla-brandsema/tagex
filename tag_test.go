package taggart

import (
	"fmt"
	"testing"
)

var tag = Tag{
	Key: "val",
}

type RangeDirective struct {
	Min int `param:"min"`
	Max int `param:"max"`
}

func (d *RangeDirective) Name() string {
	return "range"
}

func (d *RangeDirective) Handle(val int) error {
	if val < d.Min || val > d.Max {
		return fmt.Errorf("value %d out of range [%d, %d]", val, d.Min, d.Max)
	}
	return nil
}

type LengthDirective struct {
	Min int `param:"min"`
	Max int `param:"max"`
}

func (d *LengthDirective) Name() string {
	return "length"
}

func (d *LengthDirective) Handle(val string) error {
	if len(val) < d.Min || len(val) > d.Max {
		return fmt.Errorf("value %s with length %d out of range [%d, %d]", val, len(val), d.Min, d.Max)
	}
	return nil
}

type MyStruct struct {
	Number int    `val:"range, min=0, max=3"`
	Word   string `val:"length, min=2, max=5"`
}

func TestProcessStruct(t *testing.T) {

	RegisterDirective[int](&tag, &RangeDirective{})
	RegisterDirective[string](&tag, &LengthDirective{})

	instance := &MyStruct{
		Number: 2,
		Word:   "Pluk",
	}

	if ok, err := tag.ProcessStruct(instance); !ok {
		t.Fatal(err)
	}
	t.Log("success!")

}
