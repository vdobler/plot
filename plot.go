package plot

import (
	"fmt"
	"image/color"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

var now = time.Now
var col = color.RGBA{}
var floor = math.Floor

type Plot struct {
	// Data is the data to draw.
	Data *DataFrame

	// Faceting describes the Used Faceting
	Faceting Faceting

	// Mapping describes how fileds in data are mapped to Aesthetics
	Aes AesMapping

	// Layers contains all the layers displayed in the plot.
	Layers []*Layer

	// Panels are the different panels for faceting
	Panels [][]Panel

	Scales map[string]*Scale

	Theme Theme
}

func (p *Plot) Warnf(f string, args ...interface{}) {
	fmt.Printf("Warning "+f, args...)
}

// Scale provides position scales like x- and y-axis as well as color
// or other scales.
type Scale struct {
	Discrete    bool
	Type        string // pos (x/y), col/fill, size, type ... TODO: good like this?
	ExpandToTic bool

	DomainMin    float64
	DomainMax    float64
	DomainLevels FloatSet

	Transform *ScaleTransform

	// Also set up after training.
	Breaks []float64 // empty: auto
	Levels []string  // empty: auto, different length than breaks: bug

	// Set up later after real training
	Color func(x float64) color.Color // color, fill
	Pos   func(x float64) float64     // x, y, size
	Style func(x float64) int         // point and line type
}

// NewScale sets up a new scale for the given aesthetic, suitable for
// the given data in field.
func NewScale(aesthetic string, field Field) *Scale {
	scale := Scale{}
	scale.Discrete = field.Discrete()
	scale.Type = aesthetic
	scale.DomainMin = math.Inf(+1)
	scale.DomainMax = math.Inf(-1)
	scale.DomainLevels = NewFloatSet()

	return &scale
}

// Train updates the domain ranges of s according to the data found in f.
func (s *Scale) Train(f Field) {
	if f.Discrete() {
		// TODO: this depends on using the same StrIdx.
		// Maybe there should be a single StrIdx per plot.
		// This would internalize string valuies properly.
		s.DomainLevels.Join(f.Levels())
	} else {
		// Continous data.
		min, max, mini, maxi := f.MinMax()
		if mini != -1 {
			if min < s.DomainMin {
				s.DomainMin = min
			}
		}
		if maxi != -1 {
			if max > s.DomainMax {
				s.DomainMax = max
			}
		}
	}
}

func contains(s []string, t string) bool {
	for _, ss := range s {
		if t == ss {
			return true
		}
	}
	return false
}

// PrepareData is the first step in generating a plot.
// After preparing the data frame the following holds
//   - Layer has a own data frame (maybe a copy of plots data frame)
//   - This data frame has no unused (aka not mapped to aesthetics)
//     columns
//   - The columns name are the aestectics (e.g. x, y, size, color...)
//   - The columns have been transformed according to the
//     ScaleTransform associated with x, y, size, ....
//
// TODO: how about grouping? 69b0d2b contains grouping code.
func (layer *Layer) PrepareData(plot *Plot) {
	// Set up data and aestetics mapping.
	if layer.Data == nil {
		layer.Data = plot.Data.Copy()
	}
	aes := layer.DataMapping
	if len(aes) == 0 {
		aes = plot.Aes
	}

	// Drop all unused (unmapped) fields in the data frame.
	_, fields := aes.Used(false)
	for _, f := range layer.Data.FieldNames() {
		if contains(fields, f) {
			continue
		}
		delete(layer.Data.Columns, f)
	}

	// Rename mapped fields to their aestethic name
	for a, f := range aes {
		layer.Data.Rename(f, a)
	}

	// Add scale for these aesthetics if not jet set up.
	for a := range aes {
		if _, ok := plot.Scales[a]; !ok {
			// Add appropriate scale.
			plot.Scales[a] = NewScale(a, layer.Data.Columns[a])
		}
	}

	// Transform scales if needed.
	for a := range aes {
		scale := plot.Scales[a]
		if scale.Transform != nil {
			field := layer.Data.Columns[a]
			field.Apply(scale.Transform.Trans)
		}
	}

	// Pre-train scales
	for a := range aes {
		scale := plot.Scales[a]
		scale.Train(layer.Data.Columns[a])
	}
}

// ComputeStatistics computes the statistical transform. Might be the identity.
func (layer *Layer) ComputeStatistics(plot *Plot) {
	if layer.Stat == nil {
		return // The identity statistical transformation.
	}

	// Make sure all needed aesthetics (columns) are present in
	// our data frame.
	needed := layer.Stat.NeededAes()
	for _, aes := range needed {
		if _, ok := layer.Data.Columns[aes]; !ok {
			plot.Warnf("Stat %s in Layer %s needs column %s",
				layer.Stat.Name(), layer.Name, aes)
			// TODO: more cleanup?
			layer.Geom = nil // Don't draw anything.
			return
		}
	}

	// Handling of excess fields.
	usedByStat := NewStringSetFrom(needed)
	usedByStat.Join(NewStringSetFrom(layer.Stat.OptionalAes()))
	fields := NewStringSetFrom(layer.Data.FieldNames())
	fields.Remove(usedByStat)
	handling := layer.Stat.ExtraFieldHandling()
	if len(fields) == 0 || handling == IgnoreExtraFields {
		layer.Data = layer.Stat.Apply(layer.Data, plot)
	} else {
		if handling == FailOnExtraFields {
			plot.Warnf("Stat %s in Layer %s cannot cope with excess fields %v",
				layer.Stat.Name(), layer.Name, fields.Elements())
			// TODO: more cleanup?
			layer.Geom = nil // Don't draw anything.
			return
		}

		// Else, make sure all excess fields are discrete.
		for _, f := range fields.Elements() {
			if !layer.Data.Columns[f].Discrete() {
				plot.Warnf("Stat %s in Layer %s cannot cope with continous excess fields %s",
					layer.Stat.Name(), layer.Name, f)
				// TODO: more cleanup?
				layer.Geom = nil // Don't draw anything.
				return
			}
		}

		if len(fields) > 1 {
			panic("Implement me")
		}
		ef := fields.Elements()[0]
		f := layer.Data.Columns[ef]
		levels := Levels(layer.Data, ef)
		for i, level := range levels.Elements() {
			df := Filter(layer.Data, ef, level)
			delete(df.Columns, ef)
			res := layer.Stat.Apply(df, plot)
			res.Columns[ef] = f.Const(level, res.N)
			if i == 0 {
				layer.Data = res
			} else {
				layer.Data.Append(res)
			}
		}

	}

	return
}

// Unfacetted plotting, Layers have no own data.
func (p *Plot) Simple() {
	// Prepare data: map aestetics, add scales, clean data frame and
	// apply scale transformations. Mapped scales are pre-trained
	for _, layer := range p.Layers {
		layer.PrepareData(p)
	}

	// The second step: Compute statistics.
	// If a layer has a statistical transform: Apply this transformation
	// to the data frame of this layer.
	//
	for _, layer := range p.Layers {
		layer.ComputeStatistics(p)
	}

	// Construct geoms
	for i, layer := range p.Layers {
		i *= 2
		_ = layer
	}

	/*
		// Reparametrise. Skipped for the moment
		for i, layer := range p.Layers {
			// p.Layers[i].Data = layer.Geom.Reparametrise(p.Data)
		}

		// Apply position adjustment
		for i, layer := range p.Layers {
			if layer.Position == PosIdentity {
				continue
			}
			// p.Layers[i].Data = adjustPosition(layer)
		}
	*/

	// Retrain scales

}

func (p *Plot) Draw() {
	p.CreatePanels()

	// p.DistributeAes()

	// Transform scales
	// Compute statistics
	// p.ComputeStatistics()

	// Map aestetics
	// Train scales
	// Render
}

// CreatePanels populates p.Panels, coverned by p.Faceting.
//
// Not only p.Data is facetted but p.Layers also (if they contain own data).
func (p *Plot) CreatePanels() {
	// Process faceting: How many facets are there, how are they named
	rows, cols := 1, 1
	var cunq []float64
	var runq []float64

	if p.Faceting.Columns != "" {
		if f := p.Data.Columns[p.Faceting.Columns]; !f.Discrete() {
			panic(fmt.Sprintf("Cannot facet over %s (type %s)", p.Faceting.Columns, f.Type.String()))
		}
		cunq = Levels(p.Data, p.Faceting.Columns).Elements()
		cols = len(cunq)
	}

	if p.Faceting.Rows != "" {
		if f := p.Data.Columns[p.Faceting.Rows]; !f.Discrete() {
			panic(fmt.Sprintf("Cannot facet over %s (type %s)", p.Faceting.Columns, f.Type.String()))
		}
		runq = Levels(p.Data, p.Faceting.Rows).Elements()
		rows = len(runq)
	}

	p.Panels = make([][]Panel, rows, rows+1)
	for r := 0; r < rows; r++ {
		p.Panels[r] = make([]Panel, cols, cols+1)
		rdf := Filter(p.Data, p.Faceting.Rows, runq[r])
		for c := 0; c < cols; c++ {
			p.Panels[r][c].Data = Filter(rdf, p.Faceting.Columns, cunq[c])
			for _, layer := range p.Layers {
				if layer.Data != nil {
					layer.Data = Filter(layer.Data, p.Faceting.Rows, runq[r])
					layer.Data = Filter(layer.Data, p.Faceting.Columns, cunq[c])
				}
				p.Panels[r][c].Layers = append(p.Panels[r][c].Layers, layer)
			}

			if p.Faceting.Totals {
				p.Panels[r] = append(p.Panels[r], Panel{Data: rdf})
				for _, layer := range p.Layers {
					if layer.Data != nil {
						layer.Data = Filter(layer.Data, p.Faceting.Rows, runq[r])
					}
					p.Panels[r][c].Layers = append(p.Panels[r][c].Layers, layer)
				}
			}
		}
	}
	if p.Faceting.Totals {
		/*
			p.Panels = append(p.Panels, make([]Panel, cols+1))
			for c := 0; c < cols; c++ {
				cdf := p.Data.Filter(p.Faceting.Columns, cunq[c])
				p.Panels[rows][c] = Panel{Data: cdf}
			}
			p.Panels[rows][cols] = Panel{Data: p.Data}
			for _, layer := range p.Layers {
				if layer.Data != nil {
					layer.Data = layer.Data.Filter(p.Faceting.Columns, cunq[c])
				}
				p.Panels[rows][cols].Layers = append(p.Panels[rows][cols].Layers, layer)
			}
		*/
		cols++
		rows++
	}
}

/*
// merge plot aes into each layer aes
func (p *Plot) DistributeAes() {
	for r := range p.Panels {
		for c := range p.Panels[r] {
			for l := range p.Panels[r][c].Layers {
				p.Panels[r][c].Layers[l].Aes = plot.Aes.Merge(p.Panels[r][c].Layers[l].Aes)
			}
		}
	}
}

func (p *Plot) ComputeStatistics() {
	for r := range p.Panels {
		for c := range p.Panels[r] {
			p.Panels[r][c].Layers = []Layer{}
			for _, layer := range p.Layers {
				if layer.Stat != nil {
					statDF := layer.Stat.Apply(layer.Data, layer.Aes)
				}
			}
		}
	}
}
*/

type Panel struct {
	Data *DataFrame

	RowName string
	ColName string

	// Plot is the plot this panel belongs to
	Plot *Plot

	Layers []*Layer
}

type Theme struct {
	BoxAes, PointAes, BarAes, LineAes AesMapping
}

var DefaultTheme = Theme{
	BoxAes: AesMapping{
		"fill": "fixed: #cccccc",
		"line": "fixed: 0",
	},
	PointAes: AesMapping{
		"size":  "fixed: 5pt",
		"shape": "fixed: 1",
		"color": "fixed: #222222",
	},
}

type Faceting struct {
	// Columns and Rows are the faceting specification. Each may be a comma
	// seperated list of fields in the Data. An empty string means no
	// faceting in this dimension.
	Columns, Rows string

	// Totals controlls display of totals.
	Totals bool // TODO: fancier control needed

	FreeScale string // "": fixed, "x": x is free, "y": y is free, "xy": both are free

	FreeSpace string // as FreeScale but for size of panel
}

func UniqueStrings(s []string) (u []string) {
	if len(s) <= 1 {
		return s
	}
	sort.Strings(s)
	t := s[0]
	for i := 1; i <= len(s); i++ {
		if s[i] == t {
			continue
		}
		t = s[i]
		u = append(u, t)
	}
	return u
}

// AesMapping controlls the mapping of fields of a data frame to aesthetics.
// The zero value of AesMapping is the identity mapping.
//
// The following formats are used:
//     "<fieldname>"        map aesthetic to this field
//     "fixed: <value>"     set aesthetics to the given value
type AesMapping map[string]string

func (m AesMapping) Used(includeAll bool) (aes, names []string) {
	for a, n := range m {
		aes = append(aes, a)
		if includeAll || strings.Index(n, ":") == -1 {
			names = append(names, n)
		}
	}
	sort.Strings(aes)
	sort.Strings(names)
	return aes, names
}

func (m AesMapping) Copy() AesMapping {
	c := make(AesMapping, len(m))
	for a, n := range m {
		c[a] = n
	}
	return c
}

// Merge merges set values in all the ams into m and returns the merged mapping.
func (m AesMapping) Merge(ams ...AesMapping) AesMapping {
	merged := m.Copy()
	for _, am := range ams {
		for aes, fname := range am {
			if _, ok := merged[aes]; !ok {
				merged[aes] = fname
			}
		}
	}
	return merged
}

// Combine merges set values in all the ams into m and returns the merged mapping.
// Later values in ams overwrite earlier ones or values in m.
func (m AesMapping) Combine(ams ...AesMapping) AesMapping {
	merged := m.Copy()
	for _, am := range ams {
		for aes, fname := range am {
			merged[aes] = fname
		}
	}
	return merged
}

// Layer represents one layer of data
//
type Layer struct {
	Plot *Plot
	Name string

	// A nil Data will use the Data from the plot this Layer belongs to.
	Data        *DataFrame
	DataMapping AesMapping

	// Stat is the statistical transformation used in this layer.
	Stat        Stat
	StatMapping AesMapping

	// Geom is the geom to use for this layer
	Geom        Geom
	GeomMapping AesMapping

	// Aes is the aestetics mapping for this layer. Not every mapping is
	// usefull for all Geoms.

	Position PositionAdjust
}

type Viewport struct {
	// The underlying image

	// The rectangel of this vp

	// Functions to turn grob coordinates to pixel
}

type Grob interface {
	Draw(vp Viewport)
}

type GrobLine struct {
	x0, y0, x1, y1 float64
	width          float64
	style          LineStyle
	color          color.Color
}

func (line GrobLine) Draw(vp Viewport) {
}

type GrobPoint struct {
	x, y  float64
	size  float64
	style PointStyle
	color color.Color
}

func (point GrobPoint) Draw(vp Viewport) {
}

type LineStyle int

const (
	BlankLine LineStyle = iota
	SolidLine
	DashedLine
	DottedLine
	DotDashLine
	LongdashLine
	TwodashLine
)

type PointStyle int

const (
	BlankPoint PointStyle = iota
	CirclePoint
	SquarePoint
	DiamondPoint
	DeltaPoint
	NablaPoint
	SolidCirclePoint
	SolidSquarePoint
	SolidDiamondPoint
	SolidDeltaPoint
	SolidNablaPoint
	CrossPoint
	PlusPoint
	StarPoint
)

func String2PointStyle(s string) PointStyle {
	n, err := strconv.Atoi(s)
	if err == nil {
		return PointStyle(n)
	}
	switch s {
	case "circle":
		return CirclePoint
	case "square":
		return SquarePoint
	case "diamond":
		return DiamondPoint
	case "delta":
		return DeltaPoint
	case "nabla":
		return NablaPoint
	case "solid-circle":
		return SolidCirclePoint
	case "solid-square":
		return SolidSquarePoint
	case "solid-diamond":
		return SolidDiamondPoint
	case "solid-delta":
		return SolidDeltaPoint
	case "solid-nabla":
		return SolidNablaPoint
	case "cross":
		return CrossPoint
	case "plus":
		return PlusPoint
	case "star":
		return StarPoint
	}
	return BlankPoint
}

func String2PointSize(s string) float64 {
	n, err := strconv.Atoi(s)
	if err == nil {
		return float64(n)
	}
	return 6
}

// Geom is a geometrical object, a type of visual for the plot.
// Each geom.
type Geom interface {
	NeededSlots() []string
	OptionalSlots() []string

	// Render interpretes data according to m and produces Grobs.
	// TODO: Grouping?
	Render(p *Plot, data DataFrame, m AesMapping) []Grob
}

type GeomPoint struct {
	Aes AesMapping
}

var BuiltinColors = map[string]color.RGBA{
	"red":     color.RGBA{0xff, 0x00, 0x00, 0xff},
	"green":   color.RGBA{0x00, 0xff, 0x00, 0xff},
	"blue":    color.RGBA{0x00, 0x00, 0xff, 0xff},
	"cyan":    color.RGBA{0x00, 0xff, 0xff, 0xff},
	"magenta": color.RGBA{0xff, 0x00, 0xff, 0xff},
	"yellow":  color.RGBA{0xff, 0xff, 0x00, 0xff},
	"white":   color.RGBA{0xff, 0xff, 0xff, 0xff},
	"gray20":  color.RGBA{0x33, 0x33, 0x33, 0xff},
	"gray40":  color.RGBA{0x66, 0x66, 0x66, 0xff},
	"gray":    color.RGBA{0x7f, 0x7f, 0x7f, 0xff},
	"gray60":  color.RGBA{0x99, 0x99, 0x99, 0xff},
	"gray80":  color.RGBA{0xcc, 0xcc, 0xcc, 0xff},
	"black":   color.RGBA{0x00, 0x00, 0x00, 0xff},
}

func Hex2Color(s string) color.Color {
	if strings.HasPrefix(s, "#") && len(s) >= 7 {
		var r, g, b, a uint8
		fmt.Sscanf(s[1:3], "%2x", &r)
		fmt.Sscanf(s[3:5], "%2x", &g)
		fmt.Sscanf(s[5:7], "%2x", &b)
		a = 0xff
		if len(s) >= 9 {
			fmt.Sscanf(s[7:9], "%2x", &a)
		}
		return color.RGBA{r, g, b, a}
	}
	if col, ok := BuiltinColors[s]; ok {
		return col
	}

	return color.RGBA{0xaa, 0x66, 0x77, 0x7f}
}

func (p GeomPoint) NeededSlots() []string   { return []string{"x", "y"} }
func (p GeomPoint) OptionalSlots() []string { return []string{"color", "size", "type", "alpha"} }
func (p GeomPoint) Render(plot *Plot, data DataFrame, m AesMapping) []Grob {
	// TODO: The size, color and style should be populated to data earlier?

	aes := m.Merge(p.Aes, plot.Theme.PointAes, DefaultTheme.PointAes)
	points := make([]GrobPoint, data.N)
	x, y := data.Columns[aes["x"]], data.Columns[aes["y"]]

	col := aes["color"]
	var colFunc func(DataFrame, int) color.Color
	if strings.HasPrefix(col, "fixed: ") {
		theColor := Hex2Color(col[7:])
		colFunc = func(DataFrame, int) color.Color {
			return theColor
		}
	} else {
		colFunc = func(d DataFrame, i int) color.Color {
			return plot.Scales["color"].Color(d.Columns[col].Data[i])
		}
	}

	style := aes["style"]
	var styleFunc func(DataFrame, int) PointStyle
	if strings.HasPrefix(style, "fixed: ") {
		theStyle := String2PointStyle(style[7:])
		styleFunc = func(DataFrame, int) PointStyle {
			return theStyle
		}
	} else {
		styleFunc = func(d DataFrame, i int) PointStyle {
			return PointStyle(plot.Scales["pointstyle"].Style(d.Columns[style].Data[i]))
		}
	}

	size := aes["size"]
	var sizeFunc func(DataFrame, int) float64
	if strings.HasPrefix(size, "fixed: ") {
		theSize := String2PointSize(size[7:])
		sizeFunc = func(DataFrame, int) float64 {
			return theSize
		}
	} else {
		sizeFunc = func(d DataFrame, i int) float64 {
			return plot.Scales["size"].Pos(d.Columns[size].Data[i])
		}
	}

	for i := 0; i < data.N; i++ {
		points[i].x = x.Data[i]
		points[i].y = y.Data[i]
		points[i].color = colFunc(data, i)
		points[i].size = sizeFunc(data, i)
		points[i].style = styleFunc(data, i)
	}

	grobs := make([]Grob, len(points))
	for i := range points {
		grobs[i] = points[i]
	}
	return grobs
}

type GeomBar struct {
}

// -------------------------------------------------------------------------
// Scale Transformations

type ScaleTransform struct {
	Trans   func(float64) float64
	Inverse func(float64) float64
	Format  func(float64, string) string
}

var Log10Scale = ScaleTransform{
	Trans:   func(x float64) float64 { return math.Log10(x) },
	Inverse: func(y float64) float64 { return math.Pow(10, y) },
	Format:  func(y float64, s string) string { return fmt.Sprintf("10^{%s}", s) },
}

var IdentityScale = ScaleTransform{
	Trans:   func(x float64) float64 { return x },
	Inverse: func(y float64) float64 { return y },
	Format:  func(y float64, s string) string { return s },
}

// -------------------------------------------------------------------------
// Position Adjustments

type PositionAdjust int

const (
	PosIdentity PositionAdjust = iota
	PosJitter
	PosStack
	PosFill
	PosDodge
)
