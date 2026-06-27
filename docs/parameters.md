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

## Empty values

An arg with an empty value — `check:"greet, sep="` — is **rejected** at parse
time with a `*ParamParseError` ("malformed key value pair"). This is deliberate:
a stray `sep=` is far more often a typo than an intentional empty string, so the
library fails loud rather than silently accepting it.

If you genuinely want an empty string, make it explicit one of two ways:

- Mark the param optional and omit the arg — a missing `required=false` param
  leaves the field at its zero value, which for a `string` is `""`:

  ```go
  type Greeter struct {
      Sep string `param:"sep, required=false"`
  }
  // check:"greet"  ->  Sep == ""
  ```

- Or implement `ParamConverter` and map a sentinel of your choosing (for
  example `sep=none`) to `""` in your own logic.

A value **may** contain `=` — only the first `=` in a pair splits key from
value, so `pattern=a=b` parses as key `pattern`, value `a=b`. A value may **not**
contain `,` or `;`, which separate pairs and chain directives respectively (see
[Directives](directives.md#chaining-directives)). A free-form value that needs
one of those — a regular expression such as `\d{1,3}`, for instance — is **not**
inline-expressible today and is not "safe" in a tag value. For structured values,
pick an internal delimiter that avoids `,` and `;` (e.g. `|`) and split it in a
`ParamConverter`. Note a `ParamConverter` cannot rescue a literal `,`/`;` in the
value: the split that breaks on them happens *before* the converter runs.

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

## Using parameters without directives

The parameter layer stands on its own. If all you need is to map a set of string
args onto a struct's `param`-tagged fields, call `ProcessParams` directly — no
`Tag`, directive, or `ProcessStruct` required:

```go
type Config struct {
	Port int    `param:"port"`
	Host string `param:"host, default=localhost"`
}

var cfg Config
err := tagex.ProcessParams(&cfg, map[string]string{"port": "8080"})
// cfg.Port == 8080, cfg.Host == "localhost"
```

`ProcessParams`, `DefaultConvert`, and the `ParamConverter` hook are usable
independently of the directive machinery — import Tagex and use only the
parameter part. Because this is a lower layer, its failures are the
parameter-specific typed errors (`*MissingParamError`, `*ConversionError`,
`*ParamConflictError`, `*UnsupportedParamTypeError`) returned **directly**, not
wrapped in `*ProcessError`. The `*ProcessError` envelope belongs to the
directive-processing layer (`ProcessStruct`); branch on the specific types with
`errors.As` here.
