# Tagex documentation

Tagex is a library for processing custom struct tags. You define *directives* —
typed operations that evaluate or mutate a field — register them against a tag
key, and call `ProcessStruct` to walk a struct and apply them.

- [Quick start](quick-start.md) — write, register, and run your first directive.
- [Directives](directives.md) — the `Directive[T]` interface, `EvalMode` vs `MutMode`, multiple tags, nested structs.
- [Parameters](parameters.md) — `param` tags, `required`/`default` semantics, default conversion, and `ParamConverter`.
- [Lifecycle hooks](hooks.md) — `Before`, `Success`, and `Failure` callbacks around processing.
- [Errors](errors.md) — the typed error model and how to inspect it with `errors.As`.

Runnable programs live in [examples/](../examples/). Go testable examples that
render on [pkg.go.dev](https://pkg.go.dev/github.com/tedla-brandsema/tagex) live
in `example_test.go`.
