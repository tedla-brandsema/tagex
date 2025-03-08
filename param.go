package taggart

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
		return fmt.Errorf("cannot set fieldVal %q", fieldName)
	}

	switch fieldVal.Kind() {
	case reflect.String:
		fieldVal.SetString(rawVal)
	case reflect.Int:
		parsed, err := strconv.Atoi(rawVal)
		if err != nil {
			return err
		}
		fieldVal.SetInt(int64(parsed))
	case reflect.Float64:
		parsed, err := strconv.ParseFloat(rawVal, 64)
		if err != nil {
			return err
		}
		fieldVal.SetFloat(parsed)
	case reflect.Bool:
		parsed, err := strconv.ParseBool(rawVal)
		if err != nil {
			return err
		}
		fieldVal.SetBool(parsed)
	default:
		return fmt.Errorf("fieldVal %q of type %s is unsupported", fieldName, fieldVal.Kind())
	}
	return nil
}
