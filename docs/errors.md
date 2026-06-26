# Errors

Tagex returns structured, typed errors so callers can inspect *where* and *why*
processing failed rather than parsing strings. Use `errors.As` to reach a
concrete type and `errors.Is`/`Unwrap` to follow the chain.

## ProcessError

`*ProcessError` is the top-level wrapper for **every** failure from
`ProcessStruct` — including structural ones like exceeding the nesting limit — so
`errors.As(err, &pe)` is the one handhold that always works; the specific kind is
the `Cause`. (The lower-level [`ProcessParams`](parameters.md#using-parameters-without-directives)
layer returns its parameter-typed errors directly, not wrapped in
`*ProcessError`.) Its fields locate the failure:

| Field       | Meaning                                              |
| ----------- | ---------------------------------------------------- |
| `Stage`     | `input`, `pre`, `directive`, `param`, `post`, or `struct` |
| `FieldPath` | dotted path to the field (e.g. `Engine.Cylinders`)   |
| `Directive` | directive name involved, if any                      |
| `Param`     | parameter name involved, if any                      |
| `Cause`     | the underlying error (`Unwrap` returns it)           |

```go
if err := checkTag.ProcessStruct(&car); err != nil {
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
| `*InvalidTargetError`        | `ProcessStruct` got a value that isn't a pointer to a struct |
| `*NilTagError`               | `ProcessStruct` got a nil `*Tag`                          |
| `*HookError`                 | a `Before`/`Success`/`Failure` hook returned an error     |
| `*HandleError`               | a directive's `Handle` rejected the value (see below)     |
| `*UnknownDirectiveError`     | a tag value names a directive that isn't registered       |
| `*EmptyDirectiveNameError`   | `RegisterDirective` got a directive with a blank `Name()` |
| `*DuplicateDirectiveError`   | `RegisterDirective` got a name already registered on the tag |
| `*DirectiveParseError`       | a tag value has no directive name                          |
| `*ParamParseError`           | a tag arg isn't a `key=value` pair                         |
| `*MissingParamError`         | a required parameter was not provided                     |
| `*ParamConflictError`        | a `param` sets both `required` and `default`              |
| `*ConversionError`           | a parameter value couldn't be converted to the field type |
| `*UnsupportedParamTypeError` | a `param` field has an unsupported type                   |
| `*TypeMismatchError`         | a directive was applied to a field of the wrong type      |
| `*FieldAccessError`          | a field value could not be read                           |
| `*FieldSetError`             | a `MutMode` result could not be written back              |
| `*MaxDepthError`             | recursion hit the nesting limit (usually cyclic data)     |

Each type can be reached with `errors.As`:

```go
var missing *tagex.MissingParamError
if errors.As(err, &missing) {
	log.Printf("missing parameter %q", missing.Param)
}
```

## Validation failures vs. framework errors

Three different failures all surface at `StageDirective`: a directive's `Handle`
rejecting the value, a `*TypeMismatchError` (directive applied to the wrong field
type), and a `*FieldSetError` (a `MutMode` result couldn't be written). `Stage`
alone can't tell them apart. `*HandleError` is the positive marker for the first
case — a *domain* failure (the value is invalid) rather than a *wiring* bug — so
you can branch on it:

```go
if err := tag.ProcessStruct(&form); err != nil {
	var he *tagex.HandleError
	if errors.As(err, &he) {
		return badRequest(he)   // a rule fired: surface it to the user
	}
	return internalError(err)   // type mismatch / unsettable field: a bug to fix
}
```

The value's own error (whatever your `Handle` returned) is available via
`errors.As` for that type, or through `HandleError`'s `Unwrap`.
