# Quick start

This guide builds a `range` directive that checks an `int` field falls within
bounds, then applies it to a struct.

## Before you begin

- Go 1.22 or later.

```bash
go get github.com/tedla-brandsema/tagex@latest
```

## 1. Implement a directive

A directive implements `Directive[T]` for the field type `T` it handles:

```go
type RangeDirective struct {
	Min int `param:"min"`
	Max int `param:"max"`
}

func (d *RangeDirective) Name() string                { return "range" }
func (d *RangeDirective) Mode() tagex.DirectiveMode   { return tagex.EvalMode }

func (d *RangeDirective) Handle(val int) (int, error) {
	if val < d.Min || val > d.Max {
		return val, fmt.Errorf("value %d out of range [%d, %d]", val, d.Min, d.Max)
	}
	return val, nil
}
```

`Min` and `Max` are `param`-tagged, so the library fills them from the tag args
before `Handle` runs.

## 2. Create a tag and register the directive

```go
checkTag := tagex.NewTag("check")
tagex.RegisterDirective(checkTag, &RangeDirective{})
```

## 3. Annotate and process a struct

```go
type Car struct {
	Doors int `check:"range, min=2, max=4"`
}

car := Car{Doors: 4}
err := checkTag.ProcessStruct(&car)
// err == nil
```

`ProcessStruct` takes a pointer to a struct. It returns `nil` when every
directive passes, or a typed [error](errors.md) describing the first failure.

## Next steps

- Mutate fields instead of only checking them — [directives](directives.md#evalmode-vs-mutmode).
- Add typed parameters with defaults — [parameters](parameters.md).
- Run code before/after processing — [lifecycle hooks](hooks.md).
