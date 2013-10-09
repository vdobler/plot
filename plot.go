package plot

import (
	"fmt"
	"image/color"
	"math"
	"sort"
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
	Layers []Layer

	// Panels are the different panels for faceting
	Panels [][]Panel

	Scales map[string]Scale

	Theme Theme
}

// Scale provides position scales like x- and y-axis as well as color
// or other scales.
type Scale struct {
	Discrete    bool
	Type        string // pos (x/y), col/fill, size, type ... TODO: good like this?
	ExpandToTic bool

	Breaks []float64 // empty: auto
	Levels []string  // empty: auto, different length than breaks: bug

	// both 0: auto. Max > Min: manual
	DomainMin float64
	DomainMax float64

	Transform *ScaleTransform

	Color func(x float64) color.Color // color, fill
	Pos   func(x float64) float64     // x, y, size
	Style func(x float64) int         // point and line type
}

func contains(s []string, t string) bool {
	for _, ss := range s {
		if t == ss {
			return true
		}
	}
	return false
}

// Unfacetted plotting, Layers have no own data.
func (p *Plot) Simple() {
	for i, layer := range p.Layers {
		if layer.Data == nil && len(layer.Aes) == 0 {
			// This layer has no own data and no own aestetics:
			// No need for own modifications
			continue
		}

		// Set up data and aestetics mapping
		if layer.Data == nil {
			layer.Data = p.Data.Copy()
		}
		aes := layer.Aes
		if len(aes) == 0 {
			aes = p.Aes
		}

		// Map aestetics: Drop unused fields from data frame and
		// rename data frame fields to used aes.
		_, fields := aes.Used(false)
		for _, f := range layer.Data.FieldNames() {
			if contains(fields, f) {
				continue
			}
			delete(layer.Data.Columns, f)
		}
		for a, f := range aes {
			p.Data.Rename(f, a)
		}

		// TODO: Add group columns based on layer or plot grouping spez

		// Transform scales. TODO: This is ugly
		for aes := range p.Scales {
			if trans := p.Scales[aes].Transform; trans != nil {
				layer.Data.Apply(aes, trans.Trans)
			}
		}

		p.Layers[i].Data = layer.Data
	}

	// Compute statistics
	for i, layer := range p.Layers {
		data := layer.Data
		if data == nil {
			data = p.Data
		}

		if layer.Stat == nil {
			p.Layers[i].Data = data.Copy()
		} else {
			p.Layers[i].Data = layer.Stat.Apply(data, p.Aes)
		}
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
		cunq = Levels(p.Data, p.Faceting.Columns)
		cols = len(cunq)
	}

	if p.Faceting.Rows != "" {
		if f := p.Data.Columns[p.Faceting.Rows]; !f.Discrete() {
			panic(fmt.Sprintf("Cannot facet over %s (type %s)", p.Faceting.Columns, f.Type.String()))
		}
		runq = Levels(p.Data, p.Faceting.Rows)
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

	Layers []Layer
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
//     "stat: <fieldname>   map aesthetic to this field, but use the computed stat
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

	// A nil Data will use the Data from the plot this Layer belongs to.
	Data *DataFrame

	// Stat is the statistical transformation used in this layer.
	Stat Stat

	// StatData contains the result of applying Stat to Data if Stat
	// is not nil.
	StatData DataFrame

	// Geom is the geom to use for this layer
	Geom Geom

	// Aes is the aestetics mapping for this layer. Not every mapping is
	// usefull for all Geoms.
	Aes AesMapping

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
	aes := m.Merge(p.Aes, plot.Theme.PointAes)
	grobs := make([]Grob, data.N)
	x, y := data.Columns[aes["x"]], data.Columns[aes["y"]]
	for i := 0; i < data.N; i++ {
		grobs[i] = GrobPoint{
			x: x.Data[i],
			y: y.Data[i],
		}
	}

	if col, ok := aes["color"]; ok {
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
		for i := 0; i < data.N; i++ {
			point := grobs[i].(GrobPoint)
			point.color = colFunc(data, i)
			grobs[i] = point
		}
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

/********************************************

// First example: boxplots
plot := Plot{
	Data: myData,
	Faceting: Faceting{
		Columns: "Gender",
		Rows: "Smoking",
		Totals: true,
	},
	Aes: "x=Continent, y=Weight, color=Age",
}

layer := Layer{
	Data: nil,
	Stat: stat.BoxPlot{},
	Geom: geom.BoxPlot{},
	Aes: "x=X, ymin=Ymin, lower=lower, middle=Middle, upper=Upper, ymax=Ymax",
}
// produces something like
var smokingMales MyData = filter(plot.Data, "Gender=Male", "Smoking=true")
var smokingMalesBox stat.BoxPlotData = layer.Stat.Apply(smokingMales, plot.Aes)
// train scales
layer.Geom.Render(smokingMalesBox, layer.Aes)


// Second example: Histograms
plot2 := Plot{
	Data: diamonds,
	Aes: "x=Carat, y=Price, color=Cut",
}

layer2 := Layer{
	Data: nil,
	Stat: stat.Bin{},
	Geom: geom.Bar{},
	Aes: "x=X, y=Count",
}
// produces something like
var binnedData stat.BinnedData = layer2.Stat.Apply(plot.Data, plot.Aes)
// train scales
layer2.Geom.Render(binnedData, layer.Aes)

*******************************************************************/
