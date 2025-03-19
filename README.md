*Tagex* is a extensible library to easily add struct tags to your code.

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

We can add a check to both fields to see if the value of the field falls within a range. 
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
// RangeDirective implements the tagex.Directive[T any] interface by defining
// both the "Name() string" and "Handle(val T) error" methods.
//
// It also annotates two fields (Min and Max) as parameters.
type RangeDirective struct {
	Min int `param:"min"`
	Max int `param:"max"`
}

func (d *RangeDirective) Name() string {
	return "range"
}

// Even though tagex.Directive[T any] is generic, your implementation of it can be explicit. 
// Here Handle() explicitly is of type "int", which makes our "RangeDirective" explicitly of type "int".
// This means we can use our RangeDirective only on fields of type "int".
func (d *RangeDirective) Handle(val int) error {
	if val < d.Min || val > d.Max {
		return fmt.Errorf("value %d out of range [%d, %d]", val, d.Min, d.Max)
	}
	return nil
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
```
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

// RangeDirective implements the tagex.Directive[T any] interface by defining
// both the "Name() string" and "Handle(val T) error" methods.
//
// It also tags two fields (Min and Max) as parameters.
type RangeDirective struct {
	Min int `param:"min"`
	Max int `param:"max"`
}

func (RangeDirective) Name() string {
	return "range"
}

// Even though tagex.Directive[T any] is generic, your implementation of it can be explicit. 
// Here Handle() explicitly takes a val of type "int", which makes our "RangeDirective" explicitly of type "int".
// This means we can use our RangeDirective only on fields of type "int".
func (d *RangeDirective) Handle(val int) error {
	if val < d.Min || val > d.Max {
		return fmt.Errorf("value %d out of range [%d, %d]", val, d.Min, d.Max)
	}
	return nil
}

func main() {
	// Create our "check" tag
	checkTag := tagex.NewTag("check")

	// Register our "range" directive with our check tag
	tagex.RegisterDirective(&checkTag, &RangeDirective{})

	// Now we can use our "range" directive on "int" fields of our "Car" struct
	type Car struct {
		Name   string
		Doors  int `check:"range, min=2, max=4"`
		Wheels int `check:"range, min=3, max=4"`
	}

	// Create instances of our "Car" struct
	cars := []Car{
		{
			Name:   "Deux Chevaux",
			Doors:  4,
			Wheels: 4,
		},
		{
			Name:   "Reliant Robin",
			Doors:  3,
			Wheels: 3,
		},
		{
			Name:   "Eliica",
			Doors:  4,
			Wheels: 8,
		},
	}

	// Check our cars by calling "ProcessStruct" on our tag
	for _, car := range cars {
		if ok, err := checkTag.ProcessStruct(car); !ok {
			fmt.Printf("The %s did not pass our checks: %v\n", car.Name, err)
			continue
		}
		fmt.Printf("The %s passed our checks!\n", car.Name)
	}
}
```

Running this code will yield the following output:
```
The Deux Chevaux passed our checks!
The Reliant Robin passed our checks!
The Eliica did not pass our checks: error processing field "Wheels": directive "range" failed: value 8 out of range [3, 4]
```

The Eliica didn't pass our check because it has 8 wheels. This is an oversight on our part. I leave it to you to fix this bug.