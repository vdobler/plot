package plot

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	// "strconv"
	"math"
	"text/tabwriter"
	"time"
)

type DataFrame struct {
	Name string
	N    int
	Data map[string][]float64
	Type map[string]Field
}

func (df *DataFrame) FieldNames() (names []string) {
	for name, _ := range df.Data {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (df *DataFrame) Copy() *DataFrame {
	result := NewDataFrame("Copy of: " + df.Name)
	names := df.FieldNames()
	for _, name := range names {
		result.Type[name] = df.Type[name]
		result.Data[name] = make([]float64, df.N)
		copy(result.Data[name], df.Data[name])
	}
	result.N = df.N
	return result
}

func NewDataFrame(name string) *DataFrame {
	return &DataFrame{
		Name: name,
		N:    0,
		Data: make(map[string][]float64),
		Type: make(map[string]Field),
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

		values := make([]float64, n)
		field := Field{}

		switch f.Type.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			field.Type = Int
			for j := 0; j < n; j++ {
				values[j] = float64(v.Index(j).FieldByName(f.Name).Int())
			}
		case reflect.String:
			field.Type = String
			for j := 0; j < n; j++ {
				s := v.Index(j).FieldByName(f.Name).String()
				values[j] = float64(field.AddStr(s))
			}
			df.Data[f.Name] = values
		case reflect.Float32, reflect.Float64:
			field.Type = Float
			for j := 0; j < n; j++ {
				values[j] = v.Index(j).FieldByName(f.Name).Float()
			}
			df.Data[f.Name] = values
		case reflect.Struct: // Checked above for beeing time.Time
			field.Type = Time
			if n > 0 {
				field.T0 = v.Index(0).FieldByName(f.Name).Interface().(time.Time)
			}
			for j := 0; j < n; j++ {
				delta := v.Index(j).FieldByName(f.Name).Interface().(time.Time).Sub(field.T0)
				values[j] = float64(delta)
			}
		default:
			continue
		}
		df.Data[f.Name] = values
		df.Type[f.Name] = field

		// println("newSOMDataFrame: added field Name =", f.Name, "   type =", f.Type.String())

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

		values := make([]float64, n)
		field := Field{}

		switch mt.Out(0).Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			field.Type = Int
			for j := 0; j < n; j++ {
				values[j] = float64(m.Func.Call([]reflect.Value{v.Index(j)})[0].Int())
			}
		case reflect.String:
			field.Type = String
			for j := 0; j < n; j++ {
				s := m.Func.Call([]reflect.Value{v.Index(j)})[0].String()
				values[j] = float64(field.AddStr(s))
			}
		case reflect.Float32, reflect.Float64:
			field.Type = Float
			for j := 0; j < n; j++ {
				values[j] = m.Func.Call([]reflect.Value{v.Index(j)})[0].Float()
			}
		case reflect.Struct: // checked above for beeing time.Time
			field.Type = Float
			if n > 0 {
				field.T0 = m.Func.Call([]reflect.Value{v.Index(0)})[0].Interface().(time.Time)
			}
			for j := 0; j < n; j++ {
				t1 := m.Func.Call([]reflect.Value{v.Index(j)})[0].Interface().(time.Time)
				values[j] = float64(t1.Sub(field.T0))
			}
		default:
			panic("Oooops")
		}

		df.Data[m.Name] = values

		// println("newSOMDataFrame: added method Name =", m.Name, "   type =", df.Type[m.Name].String())
	}

	// TODO: Maybe pointer methods too?
	// v.Addr().MethodByName()

	return df, nil
}

type Field struct {
	Type FieldType
	Str  []string // contains the string values
	T0   time.Time
}

func (f Field) Discrete() bool { return f.Type.Discrete() }

func (f *Field) AddStr(s string) int {
	if i := f.StrIdx(s); i != -1 {
		return i
	}
	f.Str = append(f.Str, s)
	return len(f.Str) - 1
}

func (f Field) StrIdx(s string) int {
	for i, t := range f.Str {
		if s == t {
			return i
		}
	}
	return -1
}

func (f Field) Int(x float64) int64 {
	switch f.Type {
	case Int, Float, String:
		return int64(x)
	}
	panic("Ooops")
}

func (f Field) String(x float64) string {
	switch f.Type {
	case Float:
		return fmt.Sprintf("%f", x)
	case Int:
		return fmt.Sprintf("%d", math.Floor(x))
	case Time:
		t := f.Time(x)
		return t.Format("2006-01-02 15:04:05")
	case String:
		return f.Str[int(x)]
	}
	panic("Oooops")
}

func (f Field) Time(x float64) time.Time {
	switch f.Type {
	case Time:
		delta := time.Duration(int64(x))
		return f.T0.Add(delta)
	}
	panic("Oooops")
}

// FieldType represents the basisc type of a field.
type FieldType uint

const (
	Int FieldType = iota
	Float
	String
	Time
	Vector
)

// Discrete returns true if ft is a descrete type.
func (ft FieldType) Discrete() bool {
	return ft == Int || ft == String
}

// Strings representation of ft.
func (ft FieldType) String() string {
	return []string{"Int", "Float", "String", "Time", "Vector"}[ft]
}

func isTime(x reflect.Type) bool {
	return x.PkgPath() == "time" && x.Kind() == reflect.Struct && x.Name() == "Time"
}

// Filter extracts all rows from df where field==value.
// Value may be an integer or a string.  TODO: allow range function
func Filter(df *DataFrame, field string, value interface{}) *DataFrame {
	if df == nil {
		return nil
	}

	var floatVal float64

	dfft, ok := df.Type[field]
	if !ok {
		return df.Copy()
	}
	if dfft.Type != Int && dfft.Type != String {
		panic(fmt.Sprintf("Cannot filter by %q in data frame %q", field, df.Name))
	}

	// Make sure value has proper type.
	switch reflect.TypeOf(value).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		floatVal = float64(reflect.ValueOf(value).Int())
	case reflect.String:
		sidx := dfft.StrIdx(value.(string))
		if sidx == -1 {
			return nil
		}
		floatVal = float64(sidx)
	default:
		panic("Bad type of value" + reflect.TypeOf(value).String())
	}

	result := NewDataFrame(fmt.Sprintf("%s|%s=%v", df.Name, field, value))
	for n, t := range df.Type {
		result.Type[n] = t
	}
	for i := 0; i < df.N; i++ {
		if df.Data[field][i] != floatVal {
			continue
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
func Levels(df *DataFrame, field string) []interface{} {
	if df == nil {
		return nil
	}
	t, ok := df.Type[field]
	if !ok {
		panic(fmt.Sprintf("No such field %q in data frame %q.", field, df.Name))
	}
	if !t.Discrete() {
		panic(fmt.Sprintf("Field %q (%s) in data frame %q is not discrete.", field, t, df.Name))
	}

	uniques := make(map[float64]struct{})
	column := df.Data[field]
	for _, v := range column {
		uniques[v] = struct{}{}
	}
	levels := make([]float64, len(uniques))
	i := 0
	for v, _ := range uniques {
		levels[i] = v
		i++
	}
	sort.Float64s(levels)

	result := make([]interface{}, len(levels))
	for i, v := range levels {
		result[i] = v
	}
	return result
}

// MinMax returns the minimum and maximum element and their indixes.
func MinMax(df *DataFrame, field string) (minval, maxval float64, minidx, maxidx int) {
	if df == nil {
		return 0, 0, -1, -1
	}
	_, ok := df.Type[field]
	if !ok {
		panic(fmt.Sprintf("No such field %q in data frame %q.", field, df.Name))
	}

	if df.N == 0 {
		return 0, 0, -1, -1
	}

	column := df.Data[field]
	minval, maxval = column[0], column[0]
	minidx, maxidx = 0, 0
	for i := 1; i < df.N; i++ {
		v := column[i]
		if v < minval {
			minval, minidx = v, i
		} else if v > maxval {
			maxval, maxidx = v, i
		}
	}

	return minval, maxval, minidx, maxidx
}

func (df *DataFrame) Print(out io.Writer) {
	names := df.FieldNames()

	fmt.Fprintf(out, "Data Frame %q:\n", df.Name)

	w := new(tabwriter.Writer)
	w.Init(out, 0, 8, 2, ' ', 0)
	for _, name := range names {
		fmt.Fprintf(w, "\t%s", name)
	}
	fmt.Fprintln(w)
	for i := 0; i < df.N; i++ {
		fmt.Fprintf(w, "%d", i)
		for _, name := range names {
			fmt.Fprintf(w, "\t%v", df.Data[name][i])
		}
		fmt.Fprintln(w)
	}
	w.Flush()

}
