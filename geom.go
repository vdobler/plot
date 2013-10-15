package plot

import (
	"fmt"
	"image/color"
)

// Geom is a geometrical object, a type of visual for the plot.
//
// Setting aesthetics of a geom is a major TODO!
type Geom interface {
	Name() string
	NeededSlots() []string
	OptionalSlots() []string

	// Aes returns the merged default (fixed) aesthetics.
	Aes(plot *Plot) AesMapping

	// Apply position adjustments (dodge, stack, fill, identity, jitter)
	AdjustPosition(df *DataFrame, posAdj PositionAdjust)

	// Reparametirze to simpler Geom
	Reparametrize(df *DataFrame) Geom

	// Render interpretes data as the specific geom and produces Grobs.
	// TODO: Grouping?
	Render(p *Plot, data *DataFrame, aes AesMapping) []Grob
}

// -------------------------------------------------------------------------
// Geom Point

type GeomPoint struct {
	Style AesMapping // The individal fixed, aka non-mapped aesthetics
}

var _ Geom = GeomPoint{}

func (p GeomPoint) Name() string            { return "GeomPoint" }
func (p GeomPoint) NeededSlots() []string   { return []string{"x", "y"} }
func (p GeomPoint) OptionalSlots() []string { return []string{"color", "size", "type", "alpha"} }

func (p GeomPoint) Aes(plot *Plot) AesMapping {
	return p.Style.Merge(plot.Theme.PointAes, DefaultTheme.PointAes)
}

func (p GeomPoint) AdjustPosition(df *DataFrame, posAdj PositionAdjust) {
	// TODO
}

func (p GeomPoint) Reparametrize(df *DataFrame) Geom {
	// No reparamization in fundamental geom.
	return p
}

// Return a function which maps row number in df to a color.
// The color is produced by the appropriate scale of plot
// or a fixed value defined in aes.
func makeColorFunc(aes string, data *DataFrame, plot *Plot, style AesMapping) func(i int) color.Color {
	var f func(i int) color.Color
	if data.Has(aes) {
		d := data.Columns[aes].Data
		f = func(i int) color.Color {
			return plot.Scales[aes].Color(d[i])
		}
	} else {
		theColor := String2Color(style[aes])
		f = func(int) color.Color {
			return theColor
		}
	}
	return f
}

func makePosFunc(aes string, data *DataFrame, plot *Plot, style AesMapping) func(i int) float64 {
	var f func(i int) float64
	if data.Has(aes) {
		d := data.Columns[aes].Data
		f = func(i int) float64 {
			return plot.Scales[aes].Pos(d[i])
		}
	} else {
		x := String2Float(style[aes], 0, 1)
		f = func(int) float64 {
			return x
		}
	}
	return f
}

func makeStyleFunc(aes string, data *DataFrame, plot *Plot, style AesMapping) func(i int) int {
	var f func(i int) int
	if data.Has(aes) {
		d := data.Columns[aes].Data
		f = func(i int) int {
			return plot.Scales[aes].Style(d[i])
		}
	} else {
		var x int
		switch aes {
		case "shape":
			x = int(String2PointShape(style[aes]))
		case "linetype":
			x = int(String2LineType(style[aes]))
		default:
			fmt.Printf("Oooops, this should not happen.")
		}
		f = func(int) int {
			return x
		}
	}
	return f
}

func (p GeomPoint) Render(plot *Plot, data *DataFrame, style AesMapping) []Grob {
	points := make([]GrobPoint, data.N)
	x, y := data.Columns["x"], data.Columns["y"]

	colFunc := makeColorFunc("color", data, plot, style)
	// TODO: allow fill also

	sizeFunc := makePosFunc("size", data, plot, style)
	alphaFunc := makePosFunc("alpha", data, plot, style)
	shapeFunc := makePosFunc("shape", data, plot, style)

	for i := 0; i < data.N; i++ {
		points[i].x = x.Data[i]
		points[i].y = y.Data[i]
		color := colFunc(i)
		alpha := alphaFunc(i)
		points[i].color = SetAlpha(color, alpha)
		points[i].size = sizeFunc(i)
		points[i].shape = PointShape(shapeFunc(i))
	}

	grobs := make([]Grob, len(points))
	for i := range points {
		grobs[i] = points[i]
	}
	return grobs
}

// -------------------------------------------------------------------------
// Geom Bar
type GeomBar struct {
}
