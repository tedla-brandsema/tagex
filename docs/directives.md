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
tagex.MustRegisterDirective(checkTag, &RangeDirective{})
err := checkTag.ProcessStruct(&car)
```

Both register functions infer `T` from the directive, so the call site needs no
explicit type argument.

`RegisterDirective` returns an error if the directive's `Name()` is blank
(`*EmptyDirectiveNameError`) or already registered on the tag
(`*DuplicateDirectiveError`) — both are setup-time programming mistakes.
`MustRegisterDirective` is the same call but panics on that error, which is what
you want for registration done once at startup (it fails fast at boot rather
than silently). Use `RegisterDirective` and handle the error only if you
register dynamically at runtime.

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
err := tagex.ProcessStruct(&data, checkTag, normalizeTag)
```

`Tag.ProcessStruct(&data)` is shorthand for the single-tag case.

## Nested structs and collections

Processing recurses into exported fields, descending through:

- nested structs and non-nil pointers (`Engine`, `*Engine`);
- slices and arrays of structs/pointers (`Wheels []Wheel`);
- maps with struct/pointer values (`ByVIN map[string]Car`).

Field paths in errors use dotted notation with indices for elements and keys, so
a failure deep in a collection is reported with its full path — `Wheels[2].PSI`,
`ByVIN[1HGCM].Doors`. `MutMode` directives write back through all of these,
including map values (each is processed as an addressable copy and stored back).

Not recursed: interface-typed fields, and map *keys*. Self-referential pointer
graphs (a struct that reaches itself through a pointer, slice, or map) will
recurse without bound — don't process cyclic data.

## Concurrency

A `Tag` is safe to share across goroutines once its directives are registered.
Registering directives is the only operation that mutates a `Tag`, so register
all directives during setup; after that, any number of goroutines may call
`ProcessStruct` on the same `Tag` concurrently. Per-call parameter state is kept
on a per-invocation copy of the directive, never on the shared registered
instance, so concurrent calls don't interfere.
