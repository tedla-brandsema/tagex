# custom-converter

A `sum` directive that takes a `[]int` parameter — a type the default converter
doesn't support. The directive implements `ParamConverter` to parse the raw arg
(`addends=1|2|3`) into a slice.

```bash
go run ./examples/custom-converter
```

`Count` starts at 10 and becomes 16 after adding 1, 2, and 3. Because
`ConvertParam` replaces the default converter for *every* parameter on the
directive, it handles the `[]int` field itself. See
[docs/parameters.md](../../docs/parameters.md#custom-conversion-with-paramconverter).
