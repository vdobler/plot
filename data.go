package plot

import (
	"fmt"
	"reflect"
)

type DataFrame struct {
	Data   interface{}
	N      int
	Fields []Field
	SOM    bool // true for slice-of-measurements, false for collection-of-slices
}

// NewDataFrame construct a data frame from data. All fields wich can be
// used in plot are set up.
func NewDataFrame(data interface{}) (DataFrame, error) {
	t := reflect.TypeOf(data)
	switch t.Kind() {
	case reflect.Slice:
		return newSOMDataFrame(data)
	case reflect.Struct:
		panic("COS data frame not implemented")
	}
	return DataFrame{}, fmt.Errorf("cannot convert %T to data frame", t.String())
}

func newSOMDataFrame(data interface{}) (DataFrame, error) {
	t := reflect.TypeOf(data).Elem()
	v := reflect.ValueOf(data)
	n := v.Len()
	df := DataFrame{
		Data:   data,
		N:      n,
		Fields: []Field{},
		SOM:    true,
	}

	// Fields first.
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		field := Field{
			Name: f.Name,
		}
		println("Field: Name =", f.Name, "   type =", f.Type.String())
		switch f.Type.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			field.Type = Int
			field.Value = func(i int) interface{} {
				return v.Index(i).Int()
			}
		case reflect.String:
			field.Type = String
			field.Value = func(i int) interface{} {
				return v.Index(i).String()
			}
		case reflect.Float32, reflect.Float64:
			field.Type = Float
			field.Value = func(i int) interface{} {
				return v.Index(i).Float()
			}
		case reflect.Struct:
			if f.Name == "Time" && f.PkgPath == "time" {
				field.Type = Time
				field.Value = func(i int) interface{} {
					return v.Index(i).Interface()
				}
			} else {
				continue
			}
		default:
			continue
		}
		df.Fields = append(df.Fields, field)
	}

	// The same for methods
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		field := Field{
			Name: m.Name,
		}

		println("Method: Name =", m.Name, "   i/o =", m.Type.NumIn(), m.Type.NumOut())

		// Look for methods with signatures like "func(elemtype) [int,string,float,time]"
		mt := m.Type
		if mt.NumIn() != 1 || mt.NumOut() != 1 {
			continue
		}
		switch mt.Out(0).Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			field.Type = Int
			field.Value = func(i int) interface{} {
				return m.Func.Call([]reflect.Value{v})[0].Int()
			}
		case reflect.String:
			field.Type = String
			field.Value = func(i int) interface{} {
				return m.Func.Call([]reflect.Value{v})[0].String()
			}
		case reflect.Float32, reflect.Float64:
			field.Type = Float
			field.Value = func(i int) interface{} {
				return m.Func.Call([]reflect.Value{v})[0].Float()
			}
		case reflect.Struct:
			if mt.Out(0).Name() == "Time" && mt.Out(0).PkgPath() == "time" {
				field.Type = Time
				field.Value = func(i int) interface{} {
					return m.Func.Call([]reflect.Value{v})[0].Interface()
				}
			} else {
				continue
			}
		default:
			continue
		}
		df.Fields = append(df.Fields, field)
	}

	return df, nil
}

type Field struct {
	// Name of the field or method.
	Name string

	// Type of the field or return type of method.
	Type FieldType

	// Value returns the value of the filed for the i'th element in
	// the data frame.
	Value func(i int) interface{}
}

// FieldType represents the basisc type of a field.
type FieldType uint

const (
	Int FieldType = iota
	Float
	String
	Time
)

// Filter extracts all rows from df where field==value.
// Value may be an integer or a string.  TODO: allow range function
func (df DataFrame) Filter(field string, value interface{}) DataFrame {
	var ft FieldType
	// Make sure value has proper type.
	switch reflect.TypeOf(value).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		ft = Int
	case reflect.String:
		ft = String
	default:
		panic("Bad type of value" + reflect.TypeOf(value).String())
	}

	// Make sure field exists and has same type as value.
	var valFunc func(int) interface{}
	for _, f := range df.Fields {
		if f.Name == field {
			if f.Type != ft {
				panic("Incompatible filter types")
			}
			valFunc = f.Value
			break
		}
	}
	if valFunc == nil {
		panic("No such field " + field + " in " + reflect.TypeOf(df.Data).String())
	}

	if df.SOM {
		return df.filterSOM(valFunc, value, ft)
	} else {
		panic("COS not implemented")
	}
}

func (df DataFrame) filterSOM(valFunc func(int) interface{}, value interface{}, ft FieldType) DataFrame {
	v := reflect.ValueOf(df.Data)
	result := reflect.MakeSlice(reflect.TypeOf(df.Data), 0, 10)
	n := 0
	for i := 0; i < df.N; i++ {
		val := valFunc(i)

		switch ft {
		case Int:
			if val.(int64) != value.(int64) {
				continue
			}
		case String:
			if val.(string) != value.(string) {
				continue
			}
		default:
			panic("Ooops")
		}
		result = reflect.Append(result, v.Index(i))
		n++
	}

	rdf := DataFrame{
		Data:   result.Interface(),
		N:      n,
		SOM:    true,
		Fields: make([]Field, len(df.Fields)),
	}
	copy(rdf.Fields, df.Fields)
	return rdf
}

// dfValueSOM extracts field from v.
func dfValueSOM(v reflect.Value, field string) reflect.Value {
	t := v.Type()
	if _, ok := t.FieldByName(field); ok {
		return v.FieldByName(field)
	}

	if meth, ok := t.MethodByName(field); ok {
		r := meth.Func.Call([]reflect.Value{v})
		return r[0] // TODO: error handling?
	}

	panic("Bad field")
}

// MinMax determines minium and maximum value of field in df.
// For integer and string types the unique values are returned too.
func MinMax(df DataFrame, field string) (min, max interface{}, uniques []interface{}) {
	t := reflect.TypeOf(df)
	println(t.String())
	switch t.Kind() {
	case reflect.Slice:
		return minMaxSOM(df, field)
	case reflect.Struct:
		panic("COS not implemented")
	default:
		panic("Not a data frame.")
	}
}

func minMaxSOM(df DataFrame, field string) (min, max interface{}, uniques []interface{}) {
	v := reflect.ValueOf(df)
	ev := v.Index(0)
	et := ev.Type()
	et = dfValueSOM(ev, field).Type()
	switch et.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return minMaxSOMInt(v, field, et)
	case reflect.String:
		return minMaxSOMString(v, field, et)
	case reflect.Float32, reflect.Float64:
		return minMaxSOMFloat(v, field, et)
	case reflect.Struct:
		if et.Name() == "Time" && et.PkgPath() == "time" {
			return minMaxSOMTime(v, field, et)
		}
	}
	panic("Bad data frame " + et.String())
}

// extract min, max (and unique values) from data frame df for field. t is the element type.
func minMaxSOMInt(df reflect.Value, field string, t reflect.Type) (min, max int64, uniques []interface{}) {
	allVals := make(map[int64]struct{})
	n := df.Len()
	for i := 0; i < n; i++ {
		elem := df.Index(i)
		fieldVal := dfValueSOM(elem, field)
		val := fieldVal.Int()
		allVals[val] = struct{}{}

		// Determine min and max.
		if i == 0 {
			min = val
			max = val
		} else {
			if val < min {
				min = val
			} else if val > max {
				max = val
			}
		}
	}

	un := make([]interface{}, len(allVals))
	i := 0
	for v, _ := range allVals {
		un[i] = v
		i++
	}

	// TODO: Sort un

	return min, max, un
}

func minMaxSOMString(df reflect.Value, field string, t reflect.Type) (min, max string, uniques []interface{}) {
	allVals := make(map[string]struct{})
	n := df.Len()
	for i := 0; i < n; i++ {
		elem := df.Index(i)
		fieldVal := dfValueSOM(elem, field)
		val := fieldVal.String()
		allVals[val] = struct{}{}

		// Determine min and max.
		if i == 0 {
			min = val
			max = val
		} else {
			if val < min {
				min = val
			} else if val > max {
				max = val
			}
		}
	}

	un := make([]interface{}, len(allVals))
	i := 0
	for v, _ := range allVals {
		un[i] = v
		i++
	}

	return min, max, un
}

func minMaxSOMFloat(df reflect.Value, field string, t reflect.Type) (min, max float64, uniq []interface{}) {
	n := df.Len()
	for i := 0; i < n; i++ {
		elem := df.Index(i)
		fieldVal := dfValueSOM(elem, field)
		val := fieldVal.Float()

		// Determine min and max.
		if i == 0 {
			min = val
			max = val
		} else {
			if val < min {
				min = val
			} else if val > max {
				max = val
			}
		}
	}

	return min, max, nil
}

func minMaxSOMTime(df reflect.Value, field string, t reflect.Type) (min, max interface{}, uniques []interface{}) {
	panic("Not implemented")
}
