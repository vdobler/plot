package plot

import (
	"fmt"
	"reflect"
	"sort"
)

type DataFrame struct {
	Name string
	N    int
	Data map[string][]interface{}
	Type map[string]FieldType
}

func (df *DataFrame) FieldNames() (names []string) {
	for name, _ := range df.Data {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func NewDataFrame(name string) *DataFrame {
	return &DataFrame{
		Name: name,
		N:    0,
		Data: make(map[string][]interface{}),
		Type: make(map[string]FieldType),
	}
}

func (df *DataFrame) Add(name string, data []interface{}, ft FieldType) {
	df.Data[name] = data
	df.Type[name] = ft

	// Assert data has proper type if not empty.
	if len(data) > 0 {
		dt := reflect.TypeOf(data[0])
		if (ft == Int && dt.Kind() != reflect.Int64) ||
			(ft == String && dt.Kind() != reflect.String) ||
			(ft == Float && dt.Kind() != reflect.Float64) ||
			(ft == Time && !isTime(dt)) {
			panic(fmt.Sprintf("Cannot add %s: data has type []%s, expecting %s.",
				name, dt.String(), ft.String()))
		}
	}
}

// NewDataFrameFrom construct a data frame from data. All fields wich can be
// used in plot are set up.
func NewDataFrameFrom(data interface{}) (*DataFrame, error) {
	t := reflect.TypeOf(data)
	switch t.Kind() {
	case reflect.Slice:
		return newSOMDataFrame(data)
	case reflect.Struct:
		panic("COS data frame not implemented")
	}
	return &DataFrame{}, fmt.Errorf("cannot convert %T to data frame", t.String())
}

func newSOMDataFrame(data interface{}) (*DataFrame, error) {
	t := reflect.TypeOf(data).Elem()
	v := reflect.ValueOf(data)
	df := NewDataFrame(reflect.TypeOf(data).String())
	n := v.Len()
	df.N = n

	// Fields first.
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		switch f.Type.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		case reflect.String:
		case reflect.Float32, reflect.Float64:
		case reflect.Struct:
			if !isTime(f.Type) {
				continue
			}
		default:
			continue
		}

		values := make([]interface{}, n)

		switch f.Type.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			df.Type[f.Name] = Int
			for j := 0; j < n; j++ {
				values[j] = v.Index(j).FieldByName(f.Name).Int()
			}
		case reflect.String:
			df.Type[f.Name] = String
			for j := 0; j < n; j++ {
				values[j] = v.Index(j).FieldByName(f.Name).String()
			}
			df.Data[f.Name] = values
		case reflect.Float32, reflect.Float64:
			df.Type[f.Name] = Float
			for j := 0; j < n; j++ {
				values[j] = v.Index(j).FieldByName(f.Name).Float()
			}
			df.Data[f.Name] = values
		case reflect.Struct: // Checked above for beeing time.Time
			df.Type[f.Name] = Time
			for j := 0; j < n; j++ {
				values[j] = v.Index(j).FieldByName(f.Name).Interface()
			}
		default:
			continue
		}
		df.Data[f.Name] = values

		println("newSOMDataFrame: added field Name =", f.Name, "   type =", f.Type.String())

	}

	// The same for methods.
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)

		// Look for methods with signatures like "func(elemtype) [int,string,float,time]"
		mt := m.Type
		if mt.NumIn() != 1 || mt.NumOut() != 1 {
			continue
		}
		switch mt.Out(0).Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		case reflect.String:
		case reflect.Float32, reflect.Float64:
		case reflect.Struct:
			if !isTime(mt.Out(0)) {
				continue
			}
		default:
			continue
		}

		values := make([]interface{}, n)

		switch mt.Out(0).Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			df.Type[m.Name] = Int
			for j := 0; j < n; j++ {
				values[j] = m.Func.Call([]reflect.Value{v.Index(j)})[0].Int()
			}
		case reflect.String:
			df.Type[m.Name] = String
			for j := 0; j < n; j++ {
				values[j] = m.Func.Call([]reflect.Value{v.Index(j)})[0].String()
			}
		case reflect.Float32, reflect.Float64:
			df.Type[m.Name] = Float
			for j := 0; j < n; j++ {
				values[j] = m.Func.Call([]reflect.Value{v.Index(j)})[0].Float()
			}
		case reflect.Struct: // checked above for beeing time.Time
			df.Type[m.Name] = Float
			for j := 0; j < n; j++ {
				values[j] = m.Func.Call([]reflect.Value{v.Index(j)})[0].Interface
			}
		default:
			panic("Oooops")
		}

		df.Data[m.Name] = values

		println("newSOMDataFrame: added method Name =", m.Name, "   type =", df.Type[m.Name].String())
	}

	// TODO: Maybe pointer methods too?
	// v.Addr().MethodByName()

	return df, nil
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

func isTime(x reflect.Type) bool {
	return x.PkgPath() == "time" && x.Kind() == reflect.Struct && x.Name() == "Time"
}

// Filter extracts all rows from df where field==value.
// Value may be an integer or a string.  TODO: allow range function
func (df *DataFrame) Filter(field string, value interface{}) *DataFrame {
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

	// Make sure field exists and has same type as value.  TODO: 1st is bad for facetting...
	dfft, ok := df.Type[field]
	if !ok {
		panic(fmt.Sprintf("No such field %q in data frame %q", field, df.Name))
	}
	if dfft != ft {
		panic(fmt.Sprintf("No such field %q in data frame %q as type %s. Cannot filter by %s",
			field, df.Name, dfft.String, ft.String))
	}

	result := NewDataFrame(fmt.Sprintf("%s|%s=%v", df.Name, field, value))
	for n, t := range df.Type {
		result.Type[n] = t
	}
	for i := 0; i < df.N; i++ {
		switch ft {
		case Int:
			if df.Data[field][i].(int64) != value.(int64) {
				continue
			}
		case String:
			if df.Data[field][i].(string) != value.(string) {
				continue
			}
		default:
			panic(fmt.Sprintf("Oooops: %s on data frame %s", ft, df.Name))
		}
		for n, _ := range df.Type {
			result.Data[n] = append(result.Data[n], df.Data[n][i])
		}
		result.N++
	}
	return result
}

// Sorting of int64 slices.
type IntSlice []int64

func (p IntSlice) Len() int           { return len(p) }
func (p IntSlice) Less(i, j int) bool { return p[i] < p[j] }
func (p IntSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func SortInts(a []int64)              { sort.Sort(IntSlice(a)) }

// Levels returns the levels of field.
func (df *DataFrame) Levels(field string) []interface{} {
	t, ok := df.Type[field]
	if !ok {
		panic(fmt.Sprintf("No such field %q in data frame %q.", field, df.Name))
	}
	if !t.Discrete() {
		panic(fmt.Sprintf("Field %q (%s) in data frame %q is not discrete.", field, t, df.Name))
	}

	column := df.Data[field]
	switch t {
	case Int:
		uniques := make(map[int64]struct{})
		for _, v := range column {
			uniques[v.(int64)] = struct{}{}
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
		for _, v := range column {
			uniques[v.(string)] = struct{}{}
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
