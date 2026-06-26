# mutate

A `MutMode` `clamp` directive that constrains an `int` field to `[min, max]`. In
`MutMode`, `Handle`'s return value is written back to the field, so an
out-of-range value is replaced in place.

```bash
go run ./examples/mutate
```

`Volume` starts at 250 and is clamped to 100. See
[docs/directives.md](../../docs/directives.md#evalmode-vs-mutmode).
