*Tagex* is a extensible library to easily add struct tags to your code.

![Tests](https://github.com/tedla-brandsema/tagex/actions/workflows/test.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/tedla-brandsema/tagex)](https://goreportcard.com/report/github.com/tedla-brandsema/tagex)

# Installing

Use `go get` to install the latest version
of the library.
```
go get -u github.com/tedla-brandsema/tagex@latest
```

Then, import Tagex in your application:

```go
import "github.com/tedla-brandsema/tagex"
```
# Example

There are many reasons why you might want to create a custom tag, one of which might be to validate a struct field.

Let's say we have the following `Car` struct:
```go
type Car struct {
	Doors int
	Wheels int
}
```

We can add a check to both fields to see if the value of the field falls within a certain range. 
Translating that into a tag would look something like this:
```
`check:"range, min=<int>, max=<int>"`
```

Where:
 * `check` is the *key* of our tag;
 * `range` is the *directive* which we invoke;
 *  and both `min` and `max` are *parameters* for the `range` directive.

We can leverage *Tagex* to implement our range check by implementing the `Directive` interface as follows:
```go
// RangeDirective implements the "tagex.Directive[T any]" interface by defining
// both the "Name() string", "Mode() tagex.DirectiveMode" and "Handle(val T) (T, error)" methods.
//
// It also marks two fields (Min and Max) as parameters.
type RangeDirective struct {
	Min int `param:"min"`
	Max int `param:"max"`
}

// Name returns the name of the directive to be used as the directive identifier.
func (d *RangeDirective) Name() string {
	return "range"
}

// Mode returns either `tagex.EvalMode` or `tagex.MutMode`, which indicates whether the directive
// only evaluates the field value or mutates its contents.
func (d *RangeDirective) Mode() tagex.DirectiveMode {
	return tagex.EvalMode
}

// Handle is where the actual work of the directive is performed. Depending on the `tagex.DirectiveMode` 
// returned by the Mode() method, it either sets the return value as the field value (i.e., tagex.MutMode) 
// or ignores the return value (i.e., tagex.EvalMode).
//
// Even though tagex.Directive[T any] is generic, your implementation of it can be explicit.
// Here Handle takes a val of type "int", therefore "RangeDirective" is of type "int".
// This means we can only apply our RangeDirective to fields of type "int".
func (d *RangeDirective) Handle(val int) (int, error) {
	if val < d.Min || val > d.Max {
		return val, fmt.Errorf("value %d out of range [%d, %d]", val, d.Min, d.Max)
	}
	return val, nil
}
```

All directives must implement two functions:
* `Name() string` which returns the name of the *directive*;
* and `Handle(val T) error` where `T` is the *type* of the field the directive handles.

Also, notice that our `RangeDirective` has tag annotated fields of its own. Both the `Min` and `Max` field are annotated
with a `param:"<name>"` tags. This is how we add *parameters* to a *directive*.
By default, the `param` annotation can only be set on fields of type *string*, *int*, *float64* and *bool*. 
But, just like everything else in *Tagex*, this too is extensible.

We can now create a *tag* and register our directive with it as follows:
```go
checkTag := tagex.NewTag("check")
tagex.RegisterDirective(&checkTag, &RangeDirective{})
```

We are now ready to annotate our `Car` struct with our custom *tag* and start checking if instances of our struct comply
with our `RangeDirective`. Here is a complete working example:
```go
package main

import (
	"fmt"
	"github.com/tedla-brandsema/tagex"
)

// RangeDirective implements the "tagex.Directive[T any]" interface by defining
// both the "Name() string", "Mode() tagex.DirectiveMode" and "Handle(val T) (T, error)" methods.
//
// It also marks two fields (Min and Max) as parameters.
type RangeDirective struct {
	Min int `param:"min"`
	Max int `param:"max"`
}

// Name returns the name of the directive to be used as the directive identifier.
func (d *RangeDirective) Name() string {
	return "range"
}

// Mode returns either `tagex.EvalMode` or `tagex.MutMode`, which indicates whether the directive
// only evaluates the field value or mutates its contents.
func (d *RangeDirective) Mode() tagex.DirectiveMode {
	return tagex.EvalMode
}

// Handle is where the actual work of the directive is performed. Depending on the `tagex.DirectiveMode` 
// returned by the Mode() method, it either sets the return value as the field value (i.e., tagex.MutMode) 
// or ignores the return value (i.e., tagex.EvalMode).
//
// Even though tagex.Directive[T any] is generic, your implementation of it can be explicit.
// Here Handle takes a val of type "int", therefore "RangeDirective" is of type "int".
// This means we can only apply our RangeDirective to fields of type "int".
func (d *RangeDirective) Handle(val int) (int, error) {
	if val < d.Min || val > d.Max {
		return val, fmt.Errorf("value %d out of range [%d, %d]", val, d.Min, d.Max)
	}
	return val, nil
}

func main() {
	// Create our "check" tag
	checkTag := tagex.NewTag("check")

	// Register our "range" directive with our check tag
	tagex.RegisterDirective(&checkTag, &RangeDirective{})

	// Now we can use our "range" directive on "int" fields of our Car struct
	type Car struct {
		Name   string
		Doors  int `check:"range, min=2, max=4"`
		Wheels int `check:"range, min=3, max=4"`
	}

	// Create an array of "Car" instances
	cars := [...]Car{
		{
			Name:   "Citroën Deux Chevaux",
			Doors:  4,
			Wheels: 4,
		},
		{
			Name:   "Reliant Robin",
			Doors:  3,
			Wheels: 3,
		},
		{
			Name:   "VW Golf",
			Doors:  5,
			Wheels: 4,
		},
	}

	// Invoke the range directive on each car by calling "ProcessStruct" on "checkTag"
	for _, car := range cars {
		if ok, err := checkTag.ProcessStruct(&car); !ok {
			fmt.Printf("The %s did not pass our checks: %v\n", car.Name, err)
			continue
		}
		fmt.Printf("The %s passed our checks!\n", car.Name)
	}
}
```

Running this code will yield the following output:
```
The Citroën Deux Chevaux passed our checks!
The Reliant Robin passed our checks!
The VW Golf did not pass our checks: error processing field "Doors": directive "range" failed: value 5 out of range [2, 4]
```
It seems we did not take into account that hatchbacks are considered 5-door cars.  We can easily accommodate hatchbacks 
by modifying the value of the `max` parameter for the `Door` field, should we wish to do so.