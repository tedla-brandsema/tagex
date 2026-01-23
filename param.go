package tagex

import (
	"reflect"
	"strconv"
	"strings"
)

const paramKey = "param"

func processParams(data any, args map[string]string) (bool, error) {

	val, err := pointerStruct(data)
	if err != nil {
		return false, &FieldAccessError{Msg: err.Error()}
	}

	for n := 0; n < val.NumField(); n++ {
		field := val.Type().Field(n)
		if tagValue, ok := field.Tag.Lookup(paramKey); ok {
			key := strings.TrimSpace(tagValue)
			raw, ok := args[key]
			if !ok {
				return false, &MissingParamError{Param: key}
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
			if err := defaultConvert(fieldValue, raw, key); err != nil {
				return false, err
			}

		}
	}
	return true, nil
}

// Converter converts a raw string into a typed value for a reflect.Value.
const msg = "unable to convert value %q to %s"

func defaultConvert(fieldVal reflect.Value, raw string, param string) error {
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
