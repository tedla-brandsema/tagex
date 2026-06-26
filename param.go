package tagex

// Param tag matrix (options vs provided arg):
//
// Tag options                      | Arg provided? | Result
// required=true                    | yes           | uses arg value
// required=true                    | no            | error: MissingParamError
// required=false                   | yes           | uses arg value
// required=false                   | no            | skipped; field unchanged
// default=...                      | yes           | uses arg value (default ignored)
// default=...                      | no            | uses default value
// required=true + default=...      | yes/no        | error: ParamConflictError
// required=false + default=...     | yes/no        | error: ParamConflictError
//
// Any chosen value still goes through ParamConverter/DefaultConvert and can fail.
import (
	"reflect"
	"strconv"
)

const paramKey = "param"

type paramSpec struct {
	name         string
	required     bool
	defaultValue *string
}

func parseParamTag(tagValue string) (paramSpec, error) {
	name, args, err := splitTagValue(tagValue)
	if err != nil {
		return paramSpec{}, err
	}

	spec := paramSpec{
		name:     name,
		required: true,
	}
	requiredSet := false
	if raw, ok := args["required"]; ok {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return paramSpec{}, &ConversionError{Param: name, Raw: raw, Target: "bool"}
		}
		spec.required = parsed
		requiredSet = true
	}
	if raw, ok := args["default"]; ok {
		if requiredSet {
			return paramSpec{}, &ParamConflictError{Param: name}
		}
		spec.defaultValue = &raw
		spec.required = false
	}

	return spec, nil
}

func processParams(data any, args map[string]string) (bool, error) {

	val, err := pointerStruct(data)
	if err != nil {
		return false, &FieldAccessError{Msg: err.Error()}
	}

	for n := 0; n < val.NumField(); n++ {
		field := val.Type().Field(n)
		if tagValue, ok := field.Tag.Lookup(paramKey); ok {
			spec, err := parseParamTag(tagValue)
			if err != nil {
				return false, err
			}

			raw, ok := args[spec.name]
			if !ok {
				if spec.defaultValue != nil {
					raw = *spec.defaultValue
					ok = true
				} else if spec.required {
					return false, &MissingParamError{Param: spec.name}
				} else {
					continue
				}
			}

			fieldValue := val.FieldByName(field.Name)
			// Directive-owned conversion
			if pc, ok := data.(ParamConverter); ok {
				if err := pc.ConvertParam(field, fieldValue, raw); err != nil {
					return false, err
				}
				continue
			}

			// Default conversion
			if err := DefaultConvert(fieldValue, raw, spec.name); err != nil {
				return false, err
			}

		}
	}
	return true, nil
}

// ProcessParams applies tag args to param-tagged fields and returns any error.
func ProcessParams(data any, args map[string]string) error {
	_, err := processParams(data, args)
	return err
}


const msg = "unable to convert value %q to %s"

// DefaultConvert parses raw into fieldVal using the built-in conversions for
// string, int, int64, float64, and bool. param names the parameter for error
// messages. It returns a *ConversionError if raw cannot be parsed, or a
// *UnsupportedParamTypeError if the field type is not supported.
//
// A ParamConverter implementation can call DefaultConvert to handle the
// primitive fields it does not convert itself.
func DefaultConvert(fieldVal reflect.Value, raw string, param string) error {
	switch fieldVal.Kind() {
	case reflect.String:
		fieldVal.SetString(raw)
		return nil

	case reflect.Int:
		i, err := strconv.Atoi(raw)
		if err != nil {
			return &ConversionError{Param: param, Raw: raw, Target: "int"}
		}
		fieldVal.SetInt(int64(i))
		return nil

	case reflect.Int64:
		i, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return &ConversionError{Param: param, Raw: raw, Target: "int64"}
		}
		fieldVal.SetInt(i)
		return nil

	case reflect.Float64:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return &ConversionError{Param: param, Raw: raw, Target: "float64"}
		}
		fieldVal.SetFloat(f)
		return nil

	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return &ConversionError{Param: param, Raw: raw, Target: "bool"}
		}
		fieldVal.SetBool(b)
		return nil
	}

	return &UnsupportedParamTypeError{Type: fieldVal.Kind()}
}
