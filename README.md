# Tagex

*Tagex* is an extensible library for processing custom struct tags.

[![Go.Dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/tedla-brandsema/tagex)
![Tests](https://github.com/tedla-brandsema/tagex/actions/workflows/test.yml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/tedla-brandsema/tagex)](https://goreportcard.com/report/github.com/tedla-brandsema/tagex)
[![License:MIT](https://img.shields.io/badge/License-MIT-brightgreen.svg)](https://opensource.org/licenses/MIT)

## What Tagex is

Struct tags are just metadata strings — Go gives you the tag, but acting on it
means writing reflection by hand for every field, type, and parameter. Tagex
fills that gap. You implement a typed *directive* once, register it against a tag
key, and Tagex walks your struct (including nested/embedded structs and slices,
arrays, and maps of structs) to apply it:

- **Directives are typed** — `Handle(val int)` only ever sees `int` fields; a
  mismatch is a typed error, not a panic.
- **Parameters are declared, not parsed** — tag a directive's own fields with
  `param` and Tagex fills them, with `required`/`default` semantics and pluggable
  conversion. Single-quote a value to embed `,`, `;`, or `=` (`pattern='\d{1,3}'`).
- **Two modes** — `EvalMode` validates a field; `MutMode` writes a transformed
  value back.
- **Chained directives** — apply several to one field with `;`
  (`val:"trim;length, min=3"`); they run left to right, each `MutMode` result
  feeding the next.
- **Lifecycle hooks** — run `Before`, `Success`, and `Failure` callbacks around
  processing.
- **Structured errors** — failures carry the stage, field path, directive, and
  parameter so callers can inspect them with `errors.As`.
- **Safe to share** — once its directives are registered, a `Tag` can be reused
  across goroutines; `ProcessStruct` keeps per-call state off the shared directives.

## Installing

Requires **Go 1.22** or later.

```bash
go get -u github.com/tedla-brandsema/tagex@latest
```

```go
import "github.com/tedla-brandsema/tagex"
```

## Quick start

Implement a directive, register it under a tag key, and process a struct pointer:

```go
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
		Doors int `check:"range, min=2, max=4"`
	}

	car := Car{Doors: 4}
	err := checkTag.ProcessStruct(&car) // err == nil
	fmt.Println(err)
}
```

The full step-by-step is in the [quick start](docs/quick-start.md).

## Core concepts

- **[Directives](docs/directives.md)** — the `Directive[T]` interface, `EvalMode`
  vs `MutMode`, multiple tags in one pass, and nested structs/collections.
- **[Parameters](docs/parameters.md)** — `param` tags, the `required`/`default`
  matrix, default conversion, and custom conversion via `ParamConverter`.
- **[Lifecycle hooks](docs/hooks.md)** — `Before`, `Success`, and `Failure`.
- **[Errors](docs/errors.md)** — the typed error model and how to inspect it.

## Examples

Runnable programs in [examples/](examples/) — run one with `go run ./examples/<name>`:

- [validate](examples/validate/) — an `EvalMode` directive that checks a field without changing it.
- [mutate](examples/mutate/) — a `MutMode` directive whose result is written back to the field.
- [chained](examples/chained/) — several directives on one field with `;`, run left to right.
- [custom-converter](examples/custom-converter/) — a directive implementing `ParamConverter` for a `[]int` parameter.

## Documentation

Full documentation is in [docs/](docs/index.md). Package reference and Go testable
examples render on [pkg.go.dev](https://pkg.go.dev/github.com/tedla-brandsema/tagex).

## Status

Tagex is **pre-1.0 (0.x)**: the API is still settling, and breaking changes bump
the minor version. See the [changelog](CHANGELOG.md) for what changed and the
full stability policy, and pin a version.

## License

Tagex is licensed under the terms in [LICENSE](LICENSE).
