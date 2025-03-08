package tagex

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func processParams(data any, args map[string]string) (bool, error) {

	val := reflect.ValueOf(data)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return false, fmt.Errorf("expected a pointer to a struct but got %T", data)
	}
	val = val.Elem() // struct

	for n := 0; n < val.NumField(); n++ {
		field := val.Type().Field(n)
		if tagValue, ok := field.Tag.Lookup("param"); ok {
			key := strings.TrimSpace(tagValue)
			raw, ok := args[key]
			if !ok {
				return false, fmt.Errorf("%q parameter not set", key)
			}

			fieldValue := val.FieldByName(field.Name)
			err := setVal(fieldValue, raw, field.Name)
			if err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

func setVal(fieldVal reflect.Value, rawVal string, fieldName string) error {
	if !fieldVal.CanSet() {
		return fmt.Errorf("cannot set field %q", fieldName)
	}
	if conv, ok := converters[fieldVal.Kind()]; ok {
		return conv(fieldVal, rawVal)
	}
	return fmt.Errorf("%q of type %s is unsupported", fieldName, fieldVal.Kind())
}

type Converter func(reflect.Value, string) error

const msg = "unable to convert value %q to %s"

var converters = map[reflect.Kind]Converter{
	reflect.String: func(v reflect.Value, s string) error {
		v.SetString(s)
		return nil
	},
	reflect.Int: func(v reflect.Value, s string) error {
		i, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf(msg, s, "int")
		}
		v.SetInt(int64(i))
		return nil
	},
	reflect.Float64: func(v reflect.Value, s string) error {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf(msg, s, "float64")
		}
		v.SetFloat(f)
		return nil
	},
	reflect.Bool: func(v reflect.Value, s string) error {
		b, err := strconv.ParseBool(s)
		if err != nil {
			return fmt.Errorf(msg, s, "bool")
		}
		v.SetBool(b)
		return nil
	},
}

func RegisterConverter(kind reflect.Kind, converter Converter) {
	converters[kind] = converter
}
