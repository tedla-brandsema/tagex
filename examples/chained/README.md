# chained

Several directives on one field, separated by `;`. The field is tagged
`clean:"trim;lower;maxlen, n=12"`: the directives run left to right, and each
`MutMode` result feeds the next — `trim` then `lower` normalize the value, then
`maxlen` validates it.

```bash
go run ./examples/chained
```

`"  Ada  "` normalizes to `"ada"` and passes; `"  TheLongestUsername  "` is
trimmed and lowercased to `"thelongestusername"`, which then fails `maxlen`.
Because a chain stops at the first failing segment, order matters — see
[docs/directives.md](../../docs/directives.md#chaining-directives).
