package tagex

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type DirectiveFieldError struct {
	Msg string
}

func (e DirectiveFieldError) Error() string {
	return e.Msg
}

func processParams(data any, args map[string]string) (bool, error) {

	val, err := pointerStruct(data)
	if err != nil {
		return false, DirectiveFieldError{Msg: err.Error()}
	}

	for n := 0; n < val.NumField(); n++ {
		field := val.Type().Field(n)
		if tagValue, ok := field.Tag.Lookup("param"); ok {
			key := strings.TrimSpace(tagValue)
			raw, ok := args[key]
			if !ok {
				return false, ParamError{Msg: fmt.Sprintf("%q parameter not set", key)}
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
		return DirectiveFieldError{Msg: fmt.Sprintf("cannot set field %q", fieldName)}
	}
	if conv, ok := converters[fieldVal.Kind()]; ok {
		return conv(fieldVal, rawVal)
	}
	return DirectiveFieldError{Msg: fmt.Sprintf("%q of type %s is unsupported", fieldName, fieldVal.Kind())}
}

type ConversionError struct {
	Msg string
}

func (e ConversionError) Error() string {
	return e.Msg
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
			return ConversionError{Msg: fmt.Sprintf(msg, s, "int")}
		}
		v.SetInt(int64(i))
		return nil
	},
	reflect.Float64: func(v reflect.Value, s string) error {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return ConversionError{Msg: fmt.Sprintf(msg, s, "float64")}
		}
		v.SetFloat(f)
		return nil
	},
	reflect.Bool: func(v reflect.Value, s string) error {
		b, err := strconv.ParseBool(s)
		if err != nil {
			return ConversionError{Msg: fmt.Sprintf(msg, s, "bool")}
		}
		v.SetBool(b)
		return nil
	},
}

func RegisterConverter(kind reflect.Kind, converter Converter) {
	converters[kind] = converter
}
