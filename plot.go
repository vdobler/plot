package plot

import (
	"image/color"
	"reflect"
	"time"
)

var now = time.Now
var col = color.RGBA{}

// -------------------------------------------------------------------------
// Examples

type ExampleData struct {
	Price, Carat        float64
	Color, Clarity, Cut string
	N2                  float64
}

var Diamonds = []ExampleData{
	{2000, 0.1, "D", "J", "Fair", 0.000000456},
	{3330, 0.2, "D", "J", "Fair", 0.000000556},
	{6000, 0.15, "A", "O", "Perfect", 0.000000001},
}

type Panel struct {
	Data DataFrame
}

func (p *Plot) draw() {
	// Process faceting: How many facets are there, how are they named
	rows, cols := 1, 1
	var cunq []interface{}
	var runq []interface{}

	if p.Faceting.Columns != "" {
		_, _, cunq = MinMax(p.Data, p.Faceting.Columns)
		t := reflect.TypeOf(cunq[0])
		switch t.Kind() {
		case reflect.Int64, reflect.String:
		default:
			panic("Cannot facet over " + t.String())
		}
		cols = len(cunq)
	}

	if p.Faceting.Rows != "" {
		_, _, runq = MinMax(p.Data, p.Faceting.Rows)
		t := reflect.TypeOf(runq[0])
		switch t.Kind() {
		case reflect.Int64, reflect.String:
		default:
			panic("Cannot facet over " + t.String())
		}
		rows = len(runq)
	}

	panels := make([][]Panel, rows, rows+1)
	for r := 0; r < rows; r++ {
		panels[r] = make([]Panel, cols, cols+1)
		rdf := p.Data.Filter(p.Faceting.Rows, runq[r])
		for c := 0; c < cols; c++ {
			panels[r][c].Data = rdf.Filter(p.Faceting.Columns, cunq[c])
			println("row: ", runq[r].(string), " col: ", cunq[c].(int64), "  n =", panels[r][c].Data.N)
			if p.Faceting.Totals {
				panels[r] = append(panels[r], Panel{Data: rdf})
			}
		}
	}
	if p.Faceting.Totals {
		panels = append(panels, make([]Panel, cols+1))
		for c := 0; c < rows; c++ {
			cdf := p.Data.Filter(p.Faceting.Columns, cunq[c])
			panels[rows][c] = Panel{Data: cdf}
		}
		panels[rows][cols] = Panel{Data: p.Data}
		cols++
		rows++
	}

	// Transform scales
	// Compute statistics
	// Map aestetics
	// Train scales
	// Render
}

type Plot struct {
	// Data is the data to draw.
	Data DataFrame

	// Faceting describes the Used Faceting
	Faceting Faceting

	// Mapping describes how fileds in data are mapped to Aesthetics
	Aes AesMapping
}

type Theme struct {
	BoxAes, PointAes, BarAes, LineAes AesMapping
}

var DefaultTheme = Theme{
	BoxAes: AesMapping{
		Fill: "fixed: #ffffff",
		Line: "fixed: 0",
	},
	PointAes: AesMapping{
		Size:  "fixed: 5pt",
		Shape: "fixed: 1",
		Color: "fixed: #222222",
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

// AesMapping controlls the mapping of fields of a data frame to aesthetics.
// The zero value of AesMapping is the identity mapping.
type AesMapping struct {
	X     string
	Y     string
	Alpha string
	Color string
	Fill  string
	Size  string
	Shape string
	Line  string

	Lower, Middle, Upper   string
	Ymax, Ymin, Xmin, Xmax string
}

// Merge merges set values in all the as into am an retunrs the merge mapping.
func (am AesMapping) Merge(as ...AesMapping) AesMapping {
	for _, a := range as {
		if a.X != "" {
			am.X = a.X
		}
		if a.Y != "" {
			am.Y = a.Y
		}
		if a.Alpha != "" {
			am.Alpha = a.Alpha
		}
		if a.Color != "" {
			am.Color = a.Color
		}
		if a.Fill != "" {
			am.Fill = a.Fill
		}
		if a.Size != "" {
			am.Size = a.Size
		}
		if a.Shape != "" {
			am.Shape = a.Shape
		}
		if a.Line != "" {
			am.Line = a.Line
		}
		if a.Lower != "" {
			am.Lower = a.Lower
		}
		if a.Middle != "" {
			am.Middle = a.Middle
		}
		if a.Upper != "" {
			am.Upper = a.Upper
		}
		if a.Ymax != "" {
			am.Ymax = a.Ymax
		}
		if a.Ymin != "" {
			am.Ymin = a.Ymin
		}
		if a.Xmax != "" {
			am.Xmax = a.Xmax
		}
		if a.Xmin != "" {
			am.Xmin = a.Xmin
		}
	}
	return am
}

// Layer represents one layer of data
//
type Layer struct {
	// A nil Data will use the Data from the plot this Layer belongs to.
	Data interface{}

	// Stat is the statistical transformation used in this layer.
	Stat Stat

	// Geom is the geom to use for this layer
	Geom Geom

	// Aes is the aestetics mapping for this layer. Not every mapping is
	// usefull for all Geoms.
	// Each entry in Aes is of the form "aesthetics=field"
	Aes AesMapping
}

// Stat is the interface of statistical transform.
type Stat interface {
	Apply(data DataFrame, mapping AesMapping) DataFrame
}

// Geom is a geometrical object, a type of visual for the plot.
type Geom interface {
	// Bounds computes the bounds of the given scale.
	Bounds(data DataFrame, scale string) (min, max float64) // TODO: return type?

	// Render draws the Geom onto plot
	Render(data DataFrame, aes AesMapping, plot Plot)
}

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
