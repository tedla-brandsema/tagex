# Errors

Tagex returns structured, typed errors so callers can inspect *where* and *why*
processing failed rather than parsing strings. Use `errors.As` to reach a
concrete type and `errors.Is`/`Unwrap` to follow the chain.

## ProcessError

`*ProcessError` is the top-level wrapper for a failure during processing. Its
fields locate the failure:

| Field       | Meaning                                              |
| ----------- | ---------------------------------------------------- |
| `Stage`     | `pre`, `directive`, `param`, or `post`               |
| `FieldPath` | dotted path to the field (e.g. `Engine.Cylinders`)   |
| `Directive` | directive name involved, if any                      |
| `Param`     | parameter name involved, if any                      |
| `Cause`     | the underlying error (`Unwrap` returns it)           |

```go
ok, err := checkTag.ProcessStruct(&car)
if !ok {
	var pe *tagex.ProcessError
	if errors.As(err, &pe) {
		log.Printf("stage=%s field=%s directive=%s: %v",
			pe.Stage, pe.FieldPath, pe.Directive, pe.Cause)
	}
}
```

When processing multiple tags, the error is additionally wrapped in a
`*TagError` carrying the offending `TagKey`.

## Error types

| Type                         | Returned when                                              |
| ---------------------------- | --------------------------------------------------------- |
| `*ProcessError`              | any failure during processing (wraps the cause)           |
| `*TagError`                  | a tag's processing failed (carries the tag key)           |
| `*HookError`                 | a `Before`/`Success`/`Failure` hook returned an error     |
| `*UnknownDirectiveError`     | a tag value names a directive that isn't registered       |
| `*DirectiveParseError`       | a tag value has no directive name                          |
| `*ParamParseError`           | a tag arg isn't a `key=value` pair                         |
| `*MissingParamError`         | a required parameter was not provided                     |
| `*ParamConflictError`        | a `param` sets both `required` and `default`              |
| `*ConversionError`           | a parameter value couldn't be converted to the field type |
| `*UnsupportedParamTypeError` | a `param` field has an unsupported type                   |
| `*TypeMismatchError`         | a directive was applied to a field of the wrong type      |
| `*FieldAccessError`          | a field value could not be read                           |
| `*FieldSetError`             | a `MutMode` result could not be written back              |

Each type can be reached with `errors.As`:

```go
var missing *tagex.MissingParamError
if errors.As(err, &missing) {
	log.Printf("missing parameter %q", missing.Param)
}
```
