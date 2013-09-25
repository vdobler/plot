package plot

import (
	"fmt"
	"image/color"
	"math"
	"reflect"
	"time"
)

var now = time.Now
var col = color.RGBA{}

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
	var cunq []interface{}
	var runq []interface{}

	if p.Faceting.Columns != "" {
		cunq = Levels(p.Data, p.Faceting.Columns)
		t := reflect.TypeOf(cunq[0])
		switch t.Kind() {
		case reflect.Int64, reflect.String:
		default:
			panic("Cannot facet over " + t.String())
		}
		cols = len(cunq)
	}

	if p.Faceting.Rows != "" {
		runq = Levels(p.Data, p.Faceting.Rows)
		t := reflect.TypeOf(runq[0])
		switch t.Kind() {
		case reflect.Int64, reflect.String:
		default:
			panic("Cannot facet over " + t.String())
		}
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
//
// The following formats are used:
//     "<fieldname>"        map aesthetic to this field
//     "fixed: <value>"     set aesthetics to the given value
//     "stat: <fieldname>   map aesthetic to this field, but use the computed stat
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

// Merge merges set values in all the as into am and returns the merged mapping.
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
}

// Stat is the interface of statistical transform.
type Stat interface {
	Apply(data *DataFrame, mapping AesMapping) *DataFrame
	NeededAes() []string
}

type StatBin struct {
	BinWidth float64
	Drop     bool
	Origin   *float64 // TODO: both optional fields as *float64?
}

func (StatBin) NeededAes() []string {
	return []string{"x"}
}

func (s StatBin) Apply(data *DataFrame, mapping AesMapping) *DataFrame {
	if data == nil {
		return nil
	}
	field := mapping.X
	min, max, _, _ := MinMax(data, field)
	ft := data.Type[field]

	var binWidth float64 = s.BinWidth
	var numBins int
	var x2bin func(interface{}) int
	var bin2x func(int) interface{}

	switch ft {
	case Float:
		fmin, fmax := min.(float64), max.(float64)
		var origin float64
		if binWidth == 0 {
			binWidth = (fmax - fmin) / 30
			numBins = 30
		} else {
			numBins = int((fmax-fmin)/binWidth + 0.5)
		}
		if s.Origin != nil {
			origin = *s.Origin
		} else {
			origin = math.Floor(fmin/binWidth) * binWidth // round origin TODO: might overflow
		}

		x2bin = func(x interface{}) int {
			xf := x.(float64)
			return int((xf - origin) / binWidth)
		}
		bin2x = func(b int) interface{} {
			return origin + (float64(b)+0.5)*binWidth
		}
	case Int:
		imin, imax := min.(int64), max.(int64)
		var origin int64
		var bw int64
		if binWidth == 0 {
			bw = (imax - imin) / 20
			if bw == 0 {
				bw = 1
			}
			numBins = int((imax - imin) / bw)
		} else {
			bw = int64(binWidth)
			numBins = int((imax - imin) / bw)
		}
		if s.Origin != nil {
			origin = int64(*s.Origin)
		} else {
			origin = (imin / bw) * bw // round origin TODO: might overflow
		}

		x2bin = func(x interface{}) int {
			xf := x.(int64)
			return int((xf - origin) / bw)
		}
		bin2x = func(b int) interface{} {
			return origin + int64(b)*bw
		}
	default:
		panic("Oooops")
	}

	println("StatBin.Apply: binWidth =", binWidth, "   numBins =", numBins)
	counts := make([]int64, numBins+1) // TODO: Buggy here
	column := data.Data[field]
	maxcount := int64(0)
	for i := 0; i < data.N; i++ {
		bin := x2bin(column[i])
		counts[bin]++
		if counts[bin] > maxcount {
			maxcount = counts[bin]
		}
	}

	result := NewDataFrame(fmt.Sprintf("%s binned by %s", data.Name, field))
	result.Type["X"] = ft
	result.Type["Count"] = Int
	result.Type["NCount"] = Float
	result.Type["Density"] = Float
	result.Type["NDensity"] = Float
	for bin, count := range counts {
		if count == 0 && s.Drop {
			continue
		}
		result.Data["X"] = append(result.Data["X"], bin2x(bin))
		result.Data["Count"] = append(result.Data["Count"], count)
		result.Data["NCount"] = append(result.Data["NCount"], float64(0)) // TODO: here and next two
		result.Data["Density"] = append(result.Data["Density"], float64(0))
		result.Data["NDensity"] = append(result.Data["NDensity"], float64(0))
		result.N++
	}

	return result

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
