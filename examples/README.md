# Tagex examples

Each example is a self-contained `main` program. Run one with:

```bash
go run ./examples/<name>
```

| Example | What it shows |
| --- | --- |
| [validate](validate/) | An `EvalMode` directive that checks a field without changing it. |
| [mutate](mutate/) | A `MutMode` directive whose return value is written back to the field. |
| [custom-converter](custom-converter/) | A directive implementing `ParamConverter` to accept a `[]int` parameter. |

For API reference and concept guides, see [docs/](../docs/). Go testable examples
that render on pkg.go.dev live in `example_test.go`.
