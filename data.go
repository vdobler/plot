package plot

import (
	"fmt"
	"reflect"
	"sort"
)

// DataFrames are collections of tabular data.
type DataFrame struct {
	// Data is the data structure. Either a slice-of-mesurements or
	// a collection-of-slices.
	Data interface{}

	// N is the number of observations in Data
	N int

	// Fields are the variables available in each observation.
	Fields []Field

	// SOM indicates whether Data is in SOM format.
	SOM bool
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
				println("Int-Value " + v.String() + " " + v.Index(i).Type().String())
				return v.Index(i).FieldByName(f.Name).Int()
			}
		case reflect.String:
			field.Type = String
			field.Value = func(i int) interface{} {
				return v.Index(i).FieldByName(f.Name).String()
			}
		case reflect.Float32, reflect.Float64:
			field.Type = Float
			field.Value = func(i int) interface{} {
				return v.Index(i).FieldByName(f.Name).Float()
			}
		case reflect.Struct:
			if f.Name == "Time" && f.PkgPath == "time" {
				field.Type = Time
				field.Value = func(i int) interface{} {
					return v.Index(i).FieldByName(f.Name).Interface()
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
				return m.Func.Call([]reflect.Value{v.Index(i)})[0].Int()
			}
		case reflect.String:
			field.Type = String
			field.Value = func(i int) interface{} {
				return m.Func.Call([]reflect.Value{v.Index(i)})[0].String()
			}
		case reflect.Float32, reflect.Float64:
			field.Type = Float
			field.Value = func(i int) interface{} {
				return m.Func.Call([]reflect.Value{v.Index(i)})[0].Float()
			}
		case reflect.Struct:
			if mt.Out(0).Name() == "Time" && mt.Out(0).PkgPath() == "time" {
				field.Type = Time
				field.Value = func(i int) interface{} {
					return m.Func.Call([]reflect.Value{v.Index(i)})[0].Interface()
				}
			} else {
				continue
			}
		default:
			continue
		}
		df.Fields = append(df.Fields, field)
	}

	// TODO: Maybe pointer methods too?
	// v.Addr().MethodByName()

	return df, nil
}

// Field returns the field with name fn.
func (df DataFrame) Field(fn string) (Field, error) {
	for _, f := range df.Fields {
		if f.Name == fn {
			return f, nil
		}
	}
	return Field{}, fmt.Errorf("No such field %q", fn)
}

// Field represent a variable in one observation.
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

// Discrete returns true if ft is a descrete type.
func (ft FieldType) Discrete() bool {
	return ft == Int || ft == String
}

// Strings representation of ft.
func (ft FieldType) String() string {
	return []string{"Int", "Float", "String", "Time"}[ft]
}

// Filter extracts all rows from df where field==value.
// Value may be an integer or a string.  TODO: allow range function
func (df DataFrame) Filter(field string, value interface{}) DataFrame {
	var ft FieldType
	// Make sure value has proper type.
	switch reflect.TypeOf(value).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value = reflect.ValueOf(value).Int() // Make sure value is int64.
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

// Sorting of int64 slices.
type IntSlice []int64

func (p IntSlice) Len() int           { return len(p) }
func (p IntSlice) Less(i, j int) bool { return p[i] < p[j] }
func (p IntSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func SortInts(a []int64)              { sort.Sort(IntSlice(a)) }

// Levels returns the levels of field.
func (df DataFrame) Levels(field string) []interface{} {
	f, err := df.Field(field)
	if err != nil || !f.Type.Discrete() {
		panic("Field " + field + " not existent or continuous.")
	}

	switch f.Type {
	case Int:
		uniques := make(map[int64]struct{})
		for i := 0; i < df.N; i++ {
			v := f.Value(i).(int64)
			uniques[v] = struct{}{}
		}
		levels := make([]int64, len(uniques))
		i := 0
		for v, _ := range uniques {
			levels[i] = v
			i++
		}
		SortInts(levels)
		result := make([]interface{}, len(levels))
		for i, v := range levels {
			result[i] = v
		}
		return result
	case String:
		uniques := make(map[string]struct{})
		for i := 0; i < df.N; i++ {
			v := f.Value(i).(string)
			uniques[v] = struct{}{}
		}
		levels := make([]string, len(uniques))
		i := 0
		for v, _ := range uniques {
			levels[i] = v
			i++
		}
		sort.Strings(levels)
		result := make([]interface{}, len(levels))
		for i, v := range levels {
			result[i] = v
		}
		return result
	default:
		panic("Bad field for levels")
	}

}
