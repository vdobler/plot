package plot

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"math"
	"text/tabwriter"
	"time"
)

var m = math.Floor

// -------------------------------------------------------------------------
// Field Types

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

// -------------------------------------------------------------------------
// Field

// Field represents a column in a data frame.
type Field struct {
	Type   FieldType
	Data   []float64
	Pool   *StringPool
	Origin int64
}

func NewField(n int, t FieldType, pool *StringPool) Field {
	f := Field{
		Type:   t,
		Origin: 0,
		Data:   make([]float64, n),
		Pool:   pool,
	}
	return f
}

func (f Field) Copy() Field {
	c := f.CopyMeta()
	c.Data = make([]float64, len(f.Data))
	copy(c.Data, f.Data)
	return c
}

func (f Field) CopyMeta() Field {
	c := Field{
		Type:   f.Type,
		Origin: f.Origin,
		Data:   nil,
		Pool:   f.Pool,
	}
	return c
}

// Const return a copy of f with length n and a constant value of x.
// TODO: Ugly.
func (f Field) Const(x float64, n int) Field {
	c := Field{
		Type:   f.Type,
		Origin: f.Origin,
		Data:   make([]float64, n),
		Pool:   f.Pool,
	}
	for i := range c.Data {
		c.Data[i] = x
	}
	return c
}

func (f Field) Apply(t func(float64) float64) {
	if f.Type == String {
		panic("Cannot apply function to String column.")
	}

	for i, v := range f.Data {
		f.Data[i] = t(v)
	}
}

func (f Field) Discrete() bool { return f.Type.Discrete() }

func (f Field) Int(x float64) int64 {
	return int64(x) + f.Origin
}

func (f Field) AsInt() []int64 {
	ret := make([]int64, len(f.Data))
	for i, x := range f.Data {
		ret[i] = f.Int(x)
	}
	return ret
}

func (f Field) String(x float64) string {
	switch f.Type {
	case Float:
		return fmt.Sprintf("%f", x)
	case Int:
		return fmt.Sprintf("%d", f.Int(x))
	case Time:
		return f.Time(x).Format("2006-01-02 15:04:05")
	case String:
		i := int(x)
		if i >= 0 && i < len(f.Pool.pool) {
			return f.Pool.pool[i]
		}
		return "--NA--"
	}
	panic("Oooops")
}

func (f Field) Strings(x []float64) []string {
	ans := []string{}
	for _, v := range x {
		ans = append(ans, f.String(v))
	}
	return ans
}

func (f Field) AsString() []string {
	ret := make([]string, len(f.Data))
	for i, x := range f.Data {
		ret[i] = f.String(x)
	}
	return ret
}

func (f Field) Time(x float64) time.Time {
	n := int64(x) + f.Origin
	return time.Unix(n, 0)
}

func (f Field) AsTime() []time.Time {
	ret := make([]time.Time, len(f.Data))
	for i, x := range f.Data {
		ret[i] = f.Time(x)
	}
	return ret
}

// -------------------------------------------------------------------------
// Data Frames

// DataFrame is a collection of same-length columns.
type DataFrame struct {
	Name    string
	N       int
	Columns map[string]Field
	Pool    *StringPool
}

func (df *DataFrame) Has(field string) bool {
	_, has := df.Columns[field]
	return has
}

func (df *DataFrame) Append(a *DataFrame) {
	names := df.FieldNames()
	dfn := NewStringSetFrom(names)
	dfn.Remove(NewStringSetFrom(a.FieldNames()))
	if len(dfn) != 0 {
		panic("Bad append.")
	}

	df.N += a.N
	// TODO: handling of string objects?
	for _, n := range names {
		field := df.Columns[n]
		field.Data = append(field.Data, a.Columns[n].Data...)
		df.Columns[n] = field
	}
}

func (df *DataFrame) FieldNames() (names []string) {
	for name, _ := range df.Columns {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (df *DataFrame) Copy() *DataFrame {
	result := NewDataFrame(df.Name+"_copy", df.Pool)
	for name, field := range df.Columns {
		result.Columns[name] = field.Copy()
	}
	result.N = df.N
	return result
}

func (df *DataFrame) CopyMeta() *DataFrame {
	result := NewDataFrame(df.Name+"_metacopy", df.Pool)
	for name, field := range df.Columns {
		result.Columns[name] = field.CopyMeta()
	}
	result.N = 0
	return result
}

func (df *DataFrame) Rename(o, n string) {
	if o == n {
		return
	}
	col := df.Columns[o]
	delete(df.Columns, o)
	df.Columns[n] = col
}

func (df *DataFrame) Delete(fn string) {
	delete(df.Columns, fn)
}

func (df *DataFrame) Apply(field string, f func(float64) float64) {
	if df.Columns[field].Type == String {
		panic(fmt.Sprintf("Cannot transform String column %s in %s", field, df.Name))
	}

	column := df.Columns[field].Data
	for i := 0; i < df.N; i++ {
		column[i] = f(column[i])
	}
}

// -------------------------------------------------------------------------
// New Data Frames

func NewDataFrame(name string, pool *StringPool) *DataFrame {
	return &DataFrame{
		Name:    name,
		N:       0,
		Columns: make(map[string]Field),
		Pool:    pool,
	}
}

// NewDataFrameFrom construct a data frame from data. If data is already a
// *DataFrame a copy will be returned.
func NewDataFrameFrom(data interface{}, pool *StringPool) (*DataFrame, error) {
	if df, ok := data.(*DataFrame); ok {
		return df.Copy(), nil
	}

	t := reflect.TypeOf(data)
	switch t.Kind() {
	case reflect.Slice:
		return newSOMDataFrame(data, pool)
	case reflect.Struct:
		panic("COS data frame not implemented")
	}
	return &DataFrame{}, fmt.Errorf("cannot convert %T to data frame", t.String())
}

func newSOMDataFrame(data interface{}, pool *StringPool) (*DataFrame, error) {
	t := reflect.TypeOf(data).Elem()
	v := reflect.ValueOf(data)
	df := NewDataFrame(reflect.TypeOf(data).String(), pool)
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

		field := Field{
			Data: make([]float64, n),
			Pool: pool,
		}

		switch f.Type.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			field.Type = Int
			field.Origin = 0
			for j := 0; j < n; j++ {
				field.Data[j] = float64(v.Index(j).FieldByName(f.Name).Int())
			}
		case reflect.String:
			field.Type = String
			field.Origin = 0
			for j := 0; j < n; j++ {
				s := v.Index(j).FieldByName(f.Name).String()
				field.Data[j] = float64(pool.Add(s))
			}
		case reflect.Float32, reflect.Float64:
			field.Type = Float
			field.Origin = 0
			for j := 0; j < n; j++ {
				field.Data[j] = v.Index(j).FieldByName(f.Name).Float()
			}
		case reflect.Struct: // Checked above for beeing time.Time
			field.Type = Time
			if n > 0 {
				field.Origin = v.Index(0).FieldByName(f.Name).Interface().(time.Time).Unix()
			}
			for j := 0; j < n; j++ {
				t := v.Index(j).FieldByName(f.Name).Interface().(time.Time).Unix()
				field.Data[j] = float64(t - field.Origin)
			}
		}
		df.Columns[f.Name] = field

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

		field := Field{
			Data: make([]float64, n),
			Pool: pool,
		}

		switch mt.Out(0).Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			field.Type = Int
			for j := 0; j < n; j++ {
				field.Data[j] = float64(m.Func.Call([]reflect.Value{v.Index(j)})[0].Int())
			}
		case reflect.String:
			field.Type = String
			for j := 0; j < n; j++ {
				s := m.Func.Call([]reflect.Value{v.Index(j)})[0].String()
				field.Data[j] = float64(pool.Add(s))
			}
		case reflect.Float32, reflect.Float64:
			field.Type = Float
			for j := 0; j < n; j++ {
				field.Data[j] = m.Func.Call([]reflect.Value{v.Index(j)})[0].Float()
			}
		case reflect.Struct: // checked above for beeing time.Time
			field.Type = Float
			if n > 0 {
				field.Origin = m.Func.Call([]reflect.Value{v.Index(0)})[0].Interface().(time.Time).Unix()
			}
			for j := 0; j < n; j++ {
				t := m.Func.Call([]reflect.Value{v.Index(j)})[0].Interface().(time.Time).Unix()
				field.Data[j] = float64(t - field.Origin)
			}
		default:
			panic("Oooops")
		}

		df.Columns[m.Name] = field

		// println("newSOMDataFrame: added method Name =", m.Name, "   type =", df.Type[m.Name].String())
	}

	// TODO: Maybe pointer methods too?
	// v.Addr().MethodByName()

	return df, nil
}

// Filter extracts all rows from df where field==value.
// TODO: allow ranges
func Filter(df *DataFrame, field string, value interface{}) *DataFrame {
	if df == nil {
		return nil
	}

	dfft, ok := df.Columns[field]
	if !ok {
		// TODO: warn somhow...
		return df.Copy()
	}

	// Convert generic value into the float used for comparison.
	var floatVal float64
	switch reflect.TypeOf(value).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		floatVal = float64(reflect.ValueOf(value).Int())
	case reflect.String:
		sidx := df.Pool.Find(value.(string))
		if sidx == -1 {
			return nil // TODO: is this sensible?
		}
		floatVal = float64(sidx)
	case reflect.Float32, reflect.Float64:
		floatVal = float64(reflect.ValueOf(value).Float())
	case reflect.Struct:
		if !isTime(reflect.TypeOf(value)) {
			panic("Bad type of value" + reflect.TypeOf(value).String())
		}
		floatVal = float64(value.(time.Time).Unix() - dfft.Origin)
	default:
		panic("Bad type of value" + reflect.TypeOf(value).String())
	}

	result := NewDataFrame(fmt.Sprintf("%s|%s=%v", df.Name, field, value), df.Pool)

	// How many rows will be in the result data frame?
	col := df.Columns[field].Data
	result.N = 0
	for i := 0; i < df.N; i++ {
		if col[i] != floatVal {
			continue
		}
		result.N++
	}

	// Actual filtering.
	for name, field := range df.Columns {
		f := field.CopyMeta()
		f.Data = make([]float64, result.N)
		n := 0
		for i := 0; i < df.N; i++ {
			if col[i] != floatVal {
				continue
			}
			f.Data[n] = field.Data[i]
			n++
		}
		result.Columns[name] = f
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
func Levels(df *DataFrame, field string) FloatSet {
	if df == nil {
		return NewFloatSet()
	}
	t, ok := df.Columns[field]
	if !ok {
		panic(fmt.Sprintf("No such field %q in data frame %q.", field, df.Name))
	}
	if !t.Discrete() {
		panic(fmt.Sprintf("Field %q (%s) in data frame %q is not discrete.",
			field, t.Type, df.Name))
	}

	return df.Columns[field].Levels()
}

func (f Field) Levels() FloatSet {
	if !f.Discrete() {
		panic("Called Levels on non-discrete Field")
	}
	levels := NewFloatSet()
	for _, v := range f.Data {
		levels.Add(v)
	}

	return levels
}

// MinMax returns the minimum and maximum element and their indixes.
func MinMax(df *DataFrame, field string) (minval, maxval float64, minidx, maxidx int) {
	if df == nil {
		return math.NaN(), math.NaN(), -1, -1
	}
	_, ok := df.Columns[field]
	if !ok {
		return math.NaN(), math.NaN(), -1, -1
	}

	return df.Columns[field].MinMax()
}

func (f Field) MinMax() (minval, maxval float64, minidx, maxidx int) {
	if len(f.Data) == 0 {
		println("MinMax", f.Type.String(), ": no data -> NaN")
		return math.NaN(), math.NaN(), -1, -1
	}

	column := f.Data
	minval, maxval = column[0], column[0]
	// println("min/max start", minval, maxval)
	minidx, maxidx = 0, 0
	for i, v := range column {
		// println("  ", v)
		if v < minval {
			minval, minidx = v, i
			// println("    lower")
		} else if v > maxval {
			maxval, maxidx = v, i
			// println("    higher")
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
			field := df.Columns[name]
			fmt.Fprintf(w, "\t%s", field.String(field.Data[i]))
		}
		fmt.Fprintln(w)
	}
	w.Flush()

}

// GroupingField constructs a new Field of type String with the same length
// as data. The values are the concationation of the named columns.
// The named columns in data must be discrete.
func GroupingField(data *DataFrame, names []string) Field {
	// Check names
	for _, n := range names {
		if f, ok := data.Columns[n]; !ok {
			panic(fmt.Sprintf("Data frame %q has no column %q to group by.",
				data.Name, n))
		} else if !f.Discrete() {
			panic(fmt.Sprintf("Column %q in data frame %q is of type %s and cannot be used for grouping",
				n, data.Name, f.Type))
		}
	}

	field := NewField(data.N, String, data.Pool)
	for i := 0; i < data.N; i++ {
		group := ""
		for _, name := range names {
			f := data.Columns[name]
			val := f.Data[i]
			if group != "" {
				group += " | " // TODO: ist this clever? No. Maybe int-Type?
			}
			group += f.String(val)
		}
		field.Data[i] = float64(data.Pool.Add(group))
	}
	return field
}

func (f Field) Resolution() float64 {
	resolution := math.Inf(+1)
	d := f.Data
	for i := 0; i < len(f.Data)-1; i++ {
		r := math.Abs(d[i] - d[i+1])
		if r < resolution {
			resolution = r
		}
	}
	return resolution
}

// Partition df.
func Partition(df *DataFrame, field string, levels []float64) []*DataFrame {
	part := make([]*DataFrame, len(levels))
	idx := make(map[float64]int)
	for i, level := range levels {
		part[i] = df.CopyMeta()
		part[i].Delete(field)
		idx[level] = i
	}

	fc := df.Columns[field].Data
	for j := 0; j < df.N; j++ {
		level := fc[j]
		i := idx[level]
		for name, f := range df.Columns {
			if name == field {
				continue
			}
			t := part[i].Columns[name]
			t.Data = append(t.Data, f.Data[j])
			part[i].N = len(t.Data)
			part[i].Columns[name] = t
		}
	}

	return part
}
