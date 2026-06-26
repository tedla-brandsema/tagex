// Mutate runs a MutMode directive whose return value is written back to the
// field, clamping an int into a range.
package main

import (
	"fmt"

	"github.com/tedla-brandsema/tagex"
)

// ClampDirective clamps an int field into [Min, Max] and writes it back.
type ClampDirective struct {
	Min int `param:"min"`
	Max int `param:"max"`
}

func (d *ClampDirective) Name() string              { return "clamp" }
func (d *ClampDirective) Mode() tagex.DirectiveMode { return tagex.MutMode }

func (d *ClampDirective) Handle(val int) (int, error) {
	return min(max(val, d.Min), d.Max), nil
}

func main() {
	settingsTag := tagex.NewTag("settings")
	tagex.RegisterDirective(settingsTag, &ClampDirective{})

	type Config struct {
		Volume int `settings:"clamp, min=0, max=100"`
	}

	cfg := Config{Volume: 250}
	if err := settingsTag.ProcessStruct(&cfg); err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("clamped volume:", cfg.Volume)
	// Output:
	// clamped volume: 100
}
