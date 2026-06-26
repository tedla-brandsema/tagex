# Directives

A directive is a named, typed operation applied to a struct field. It implements
the generic `Directive[T]` interface, where `T` is the field type it handles:

```go
type Directive[T any] interface {
	Name() string
	Mode() DirectiveMode
	Handle(val T) (T, error)
}
```

- `Name()` is the identifier used in a tag value (`check:"range, ..."` selects the
  directive named `range`).
- `Mode()` decides whether `Handle`'s return value is written back (see below).
- `Handle(val T)` does the work and returns the (possibly new) value and an error.

Register a directive against a tag key, then process a struct pointer:

```go
checkTag := tagex.NewTag("check")
tagex.RegisterDirective(&checkTag, &RangeDirective{})
ok, err := checkTag.ProcessStruct(&car)
```

`RegisterDirective` infers `T` from the directive, so the call site needs no
explicit type argument.

## EvalMode vs MutMode

`Mode()` returns one of two constants:

| Mode       | `Handle` return value | Use for                            |
| ---------- | --------------------- | ---------------------------------- |
| `EvalMode` | ignored               | validation / inspection            |
| `MutMode`  | written back to field | normalization / transformation     |

In `MutMode` the field must be settable (it is, when you pass a pointer to the
struct). A `MutMode` directive that returns a different value replaces the field:

```go
func (d *ClampDirective) Mode() tagex.DirectiveMode { return tagex.MutMode }
func (d *ClampDirective) Handle(val int) (int, error) {
	return min(max(val, d.Min), d.Max), nil // value is clamped in place
}
```

## Field type and `T`

`Handle(val T)` fixes the field type the directive accepts. Applying a directive
to a field of a different type fails with a `*TypeMismatchError`. A directive for
`int` fields declares `Handle(val int) (int, error)`; one for `string` declares
`Handle(val string) (string, error)`.

## Multiple tags in one pass

Register directives under different keys and process them together:

```go
ok, err := tagex.ProcessStruct(&data, &checkTag, &normalizeTag)
```

`Tag.ProcessStruct(&data)` is shorthand for the single-tag case.

## Nested structs

Processing recurses into exported struct fields and non-nil pointer-to-struct
fields. Field paths in errors use dotted notation (e.g. `Engine.Cylinders`), so a
failure deep in a nested struct is reported with its full path.
