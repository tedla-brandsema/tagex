// Chained applies several directives to one field by separating them with ';'.
// They run left to right and each MutMode result feeds the next, so this trims a
// username, lowercases it, then enforces a maximum length — in that order.
package main

import (
	"fmt"
	"strings"

	"github.com/tedla-brandsema/tagex"
)

// TrimDirective removes surrounding whitespace (MutMode).
type TrimDirective struct{}

func (d *TrimDirective) Name() string              { return "trim" }
func (d *TrimDirective) Mode() tagex.DirectiveMode { return tagex.MutMode }
func (d *TrimDirective) Handle(val string) (string, error) {
	return strings.TrimSpace(val), nil
}

// LowerDirective lowercases the field (MutMode).
type LowerDirective struct{}

func (d *LowerDirective) Name() string              { return "lower" }
func (d *LowerDirective) Mode() tagex.DirectiveMode { return tagex.MutMode }
func (d *LowerDirective) Handle(val string) (string, error) {
	return strings.ToLower(val), nil
}

// MaxLenDirective rejects a value longer than N (EvalMode).
type MaxLenDirective struct {
	N int `param:"n"`
}

func (d *MaxLenDirective) Name() string              { return "maxlen" }
func (d *MaxLenDirective) Mode() tagex.DirectiveMode { return tagex.EvalMode }
func (d *MaxLenDirective) Handle(val string) (string, error) {
	if len(val) > d.N {
		return val, fmt.Errorf("%q is longer than %d characters", val, d.N)
	}
	return val, nil
}

func main() {
	cleanTag := tagex.NewTag("clean")
	tagex.MustRegisterDirective(cleanTag, &TrimDirective{})
	tagex.MustRegisterDirective(cleanTag, &LowerDirective{})
	tagex.MustRegisterDirective(cleanTag, &MaxLenDirective{})

	type Account struct {
		Username string `clean:"trim;lower;maxlen, n=12"`
	}

	accounts := []Account{
		{Username: "  Ada  "},
		{Username: "  TheLongestUsername  "},
	}

	for i := range accounts {
		if err := cleanTag.ProcessStruct(&accounts[i]); err != nil {
			fmt.Printf("rejected: %v\n", err)
			continue
		}
		fmt.Printf("normalized to %q\n", accounts[i].Username)
	}
	// Output:
	// normalized to "ada"
	// rejected: tag "clean" error: directive processing field "Username" directive "maxlen": "thelongestusername" is longer than 12 characters
}
