# validate

An `EvalMode` `range` directive that checks an `int` field falls within
`[min, max]`. `EvalMode` ignores `Handle`'s return value, so the field is never
changed — only validated.

```bash
go run ./examples/validate
```

The second car has 5 doors and fails its `range` check; the error carries the
tag key, field path, and directive name. See [docs/directives.md](../../docs/directives.md)
and [docs/errors.md](../../docs/errors.md).
