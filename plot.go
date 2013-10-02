package plot

import (
	"fmt"
	"image/color"
	"math"
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

// Geom is a geometrical object, a type of visual for the plot.
type Geom interface {
	// Bounds computes the bounds of the given scale.
	Bounds(data DataFrame, scale string) (min, max float64) // TODO: return type?

	// Render draws the Geom onto plot
	Render(data DataFrame, aes AesMapping, plot Plot)
}

type GeomBar struct {
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
