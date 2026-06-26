# Parameters

A directive can declare parameters by tagging its own fields with `param`. The
library reads the tag args and fills those fields before `Handle` runs.

```go
type RangeDirective struct {
	Min int `param:"min"`
	Max int `param:"max"`
}
```

Applied as `check:"range, min=2, max=4"`, the args `min=2` and `max=4` are
assigned to `Min` and `Max`.

## required and default

A `param` tag accepts two options that control what happens when the arg is
absent:

| Tag                              | Arg provided | Result                                   |
| -------------------------------- | ------------ | ---------------------------------------- |
| `param:"min"`                    | yes          | uses the arg value                       |
| `param:"min"`                    | no           | error: `*MissingParamError`              |
| `param:"min, required=false"`    | yes          | uses the arg value                       |
| `param:"min, required=false"`    | no           | skipped; field left unchanged            |
| `param:"min, default=2"`         | yes          | uses the arg value (default ignored)     |
| `param:"min, default=2"`         | no           | uses the default value                   |
| `param:"min, required=true, default=2"` | any   | error: `*ParamConflictError`             |

Parameters are **required by default**. Setting both `required` and `default` is
a conflict — `default` already implies the parameter is optional.

Whichever value is chosen still goes through conversion and can fail with a
`*ConversionError`.

## Default conversion

Out of the box, `param` fields may be `string`, `int`, `int64`, `float64`, or
`bool`. Values are parsed with `strconv`; an unparseable value yields a
`*ConversionError`, and an unsupported field type yields a
`*UnsupportedParamTypeError`.

## Custom conversion with ParamConverter

To accept richer types (slices, custom formats), implement `ParamConverter` on
the directive:

```go
type ParamConverter interface {
	ConvertParam(field reflect.StructField, fieldValue reflect.Value, raw string) error
}
```

When a directive implements it, `ConvertParam` is called for **every** parameter
on that directive instead of the default converter. Handle the types you care
about, and call `tagex.DefaultConvert` to fall back to the built-in conversions
for the rest:

```go
func (d *SumDirective) ConvertParam(field reflect.StructField, fieldValue reflect.Value, raw string) error {
	if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.Int {
		// parse raw ("1|2|3") into []int and set fieldValue
		return nil
	}
	return tagex.DefaultConvert(fieldValue, raw, field.Tag.Get("param"))
}
```

`DefaultConvert` is also usable on its own — if you only need to map primitive
tag args onto a struct's fields, you can call it directly instead of registering
a directive.

See the [custom-converter example](../examples/custom-converter/) for a complete
program.
