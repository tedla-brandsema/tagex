package tagex

import "fmt"

const valTagKey = "val"

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
