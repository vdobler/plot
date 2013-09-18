package plot

import (
	"reflect"
)

// Len computes the length of df.
func Len(df DataFrame) int {
	t := reflect.TypeOf(df)
	v := reflect.ValueOf(df)
	switch t.Kind() {
	case reflect.Slice:
		return v.Len()
	case reflect.Struct:
		v = v.Field(0)
		if v.Kind() != reflect.Slice {
			panic("Not a data frame.")
		}
		return v.Len()
	default:
		panic("Not a data frame.")
	}
}

// Filter extracts all rows from df where field==value.
func Filter(df DataFrame, field string, value int64) DataFrame {
	n := Len(df)
	t := reflect.TypeOf(df)
	switch t.Kind() {
	case reflect.Slice:
		return filterSOM(df, field, value, n)
	case reflect.Struct:
		// return filterCOS(df, n)
	default:
		panic("Not a data frame.")
	}
	return nil
}

func filterSOM(df DataFrame, field string, value int64, n int) DataFrame {
	v := reflect.ValueOf(df)
	ev := v.Index(0)
	et := ev.Type()
	result := reflect.MakeSlice(reflect.TypeOf(df), 0, 10)
	for i:=0; i<n; i++ {
		elem := v.Index(i)
		fieldVal := dfValueSOM(elem, et, field)
		if fieldVal.Int() != value { // TODO handle uints
			continue
		}
		result = reflect.Append(result, elem)
	}
	return result.Interface()
}

// dfValueSOM extracts field from v which must be of type t.
func dfValueSOM(v reflect.Value, t reflect.Type, field string) reflect.Value {
	if _, ok := t.FieldByName(field); ok {
		return v.FieldByName(field)
	}

	if meth, ok := t.MethodByName(field); ok {
		r := meth.Func.Call([]reflect.Value{v})
		return r[0] // TODO: error handling?
	}

	panic("Bad field")
}
