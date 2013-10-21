package plot

import (
	"fmt"
	"math"
	"strings"
)

var _ = fmt.Printf

// Geom is a geometrical object, a type of visual for the plot.
//
// Setting aesthetics of a geom is a major TODO!
type Geom interface {
	Name() string            // The name of the geom.
	NeededSlots() []string   // The needed slots to construct this geom.
	OptionalSlots() []string // The optional slots this geom understands.

	// Aes returns the merged default (fixed) aesthetics.
	Aes(plot *Plot) AesMapping

	// Apply position adjustments (dodge, stack, fill, identity, jitter).
	AdjustPosition(df *DataFrame, posAdj PositionAdjust)

	// Reparametrize to fundamental Geom.
	Reparametrize(df *DataFrame) Geom

	// Compute min and max of the given aestetics on df.
	Bounds(aes string, df *DataFrame) (min, max float64)

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
func (p GeomPoint) OptionalSlots() []string { return []string{"color", "size", "shape", "alpha"} }

func (p GeomPoint) Aes(plot *Plot) AesMapping {
	return MergeStyles(p.Style, plot.Theme.PointStyle, DefaultTheme.PointStyle)
}

func (p GeomPoint) AdjustPosition(df *DataFrame, posAdj PositionAdjust) {
	// TODO
}

func (p GeomPoint) Reparametrize(df *DataFrame) Geom {
	// No reparamization in fundamental geom.
	return p
}

func (p GeomPoint) Bounds(aes string, df *DataFrame) (min, max float64) {
	return ComputeBounds(aes, df, nil)
}

func (p GeomPoint) Render(plot *Plot, data *DataFrame, style AesMapping) []Grob {
	points := make([]GrobPoint, data.N)
	x, y := data.Columns["x"], data.Columns["y"]
	xf, yf := plot.Scales["x"].Pos, plot.Scales["y"].Pos

	colFunc := makeColorFunc("color", data, plot, style)
	// TODO: allow fill also

	sizeFunc := makePosFunc("size", data, plot, style)
	alphaFunc := makePosFunc("alpha", data, plot, style)
	shapeFunc := makePosFunc("shape", data, plot, style)

	for i := 0; i < data.N; i++ {
		points[i].x = xf(x.Data[i])
		points[i].y = yf(y.Data[i])
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
// Geom Line
type GeomLine struct {
	Style AesMapping // The individal fixed, aka non-mapped aesthetics
}

var _ Geom = GeomLine{}

func (p GeomLine) Name() string            { return "GeomLine" }
func (p GeomLine) NeededSlots() []string   { return []string{"x", "y"} }
func (p GeomLine) OptionalSlots() []string { return []string{"color", "size", "linetype", "alpha"} }

func (p GeomLine) Aes(plot *Plot) AesMapping {
	return MergeStyles(p.Style, plot.Theme.LineStyle, DefaultTheme.LineStyle)
}

func (p GeomLine) AdjustPosition(df *DataFrame, posAdj PositionAdjust) {
	// TODO
}

func (p GeomLine) Reparametrize(df *DataFrame) Geom {
	// No reparamization in fundamental geom.
	return p
}

func (p GeomLine) Bounds(aes string, df *DataFrame) (min, max float64) {
	return ComputeBounds(aes, df, nil)
}

func (p GeomLine) Render(plot *Plot, data *DataFrame, style AesMapping) []Grob {
	x, y := data.Columns["x"], data.Columns["y"]
	grobs := make([]Grob, 0)
	colFunc := makeColorFunc("color", data, plot, style)
	sizeFunc := makePosFunc("size", data, plot, style)
	alphaFunc := makePosFunc("alpha", data, plot, style)
	typeFunc := makePosFunc("linetype", data, plot, style)

	// TODO: Grouping

	if data.Has("color") || data.Has("size") || data.Has("alpha") || data.Has("linetype") {
		// Some of the optional aesthetics are mapped (not set).
		// Cannot represent safely as a GrobPath; thus use lots
		// of GrobLine.
		// TODO: instead "of by one" why not use average?
		for i := 0; i < data.N-1; i++ {
			line := GrobLine{
				x0:       x.Data[i],
				y0:       y.Data[i],
				x1:       x.Data[i+1],
				y1:       y.Data[i+1],
				color:    SetAlpha(colFunc(i), alphaFunc(i)),
				size:     sizeFunc(i),
				linetype: LineType(typeFunc(i)),
			}
			grobs = append(grobs, line)
		}
	} else {
		// All segemtns have same color, linetype and size, use a GrobPath
		points := make([]struct{ x, y float64 }, data.N)
		for i := 0; i < data.N; i++ {
			points[i].x = x.Data[i]
			points[i].y = y.Data[i]
		}
		path := GrobPath{
			points:   points,
			color:    SetAlpha(colFunc(0), alphaFunc(0)),
			size:     sizeFunc(0),
			linetype: LineType(typeFunc(0)),
		}
		grobs = append(grobs, path)
	}

	return grobs
}

// -------------------------------------------------------------------------
// Geom ABLine
type GeomABLine struct {
	Intercept, Slope float64
	Style            AesMapping // The individal fixed, aka non-mapped aesthetics
}

var _ Geom = GeomABLine{}

func (p GeomABLine) Name() string            { return "GeomABLine" }
func (p GeomABLine) NeededSlots() []string   { return []string{"intercept", "slope"} }
func (p GeomABLine) OptionalSlots() []string { return []string{"color", "size", "linetype", "alpha"} }

func (p GeomABLine) Aes(plot *Plot) AesMapping {
	return MergeStyles(p.Style, plot.Theme.LineStyle, DefaultTheme.LineStyle)
}

func (p GeomABLine) AdjustPosition(df *DataFrame, posAdj PositionAdjust) {
	// TODO
}

func (p GeomABLine) Reparametrize(df *DataFrame) Geom {
	// No reparamization in fundamental geom.
	return p
}

func (p GeomABLine) Bounds(aes string, df *DataFrame) (min, max float64) {
	return ComputeBounds(aes, df, nil)
}

func (p GeomABLine) Render(plot *Plot, data *DataFrame, style AesMapping) []Grob {
	ic, sc := data.Columns["intercept"].Data, data.Columns["slope"].Data
	grobs := make([]Grob, data.N)
	colFunc := makeColorFunc("color", data, plot, style)
	sizeFunc := makePosFunc("size", data, plot, style)
	alphaFunc := makePosFunc("alpha", data, plot, style)
	typeFunc := makeStyleFunc("linetype", data, plot, style)

	scaleX, scaleY := plot.Scales["x"], plot.Scales["y"]
	xmin, xmax := scaleX.DomainMin, scaleX.DomainMax
	sxmin, sxmax := scaleX.Pos(xmin), scaleX.Pos(xmax)

	for i := 0; i < data.N; i++ {
		intercept, slope := ic[i], sc[i]
		line := GrobLine{
			x0:       sxmin,
			y0:       scaleY.Pos(xmin*slope + intercept),
			x1:       sxmax,
			y1:       scaleY.Pos(xmax*slope + intercept),
			color:    SetAlpha(colFunc(i), alphaFunc(i)),
			size:     sizeFunc(i),
			linetype: LineType(typeFunc(i)),
		}
		grobs[i] = line
	}

	return grobs
}

// -------------------------------------------------------------------------
// Geom Text

type GeomText struct {
	Style AesMapping // The individal fixed, aka non-mapped aesthetics
}

var _ Geom = GeomText{}

func (t GeomText) Name() string            { return "GeomText" }
func (t GeomText) NeededSlots() []string   { return []string{"x", "y", "text"} }
func (t GeomText) OptionalSlots() []string { return []string{"color", "size", "angle", "alpha"} }

func (t GeomText) Aes(plot *Plot) AesMapping {
	return MergeStyles(t.Style, plot.Theme.TextStyle, DefaultTheme.TextStyle)
}

func (t GeomText) AdjustPosition(df *DataFrame, posAdj PositionAdjust) {
	// TODO
}

func (t GeomText) Reparametrize(df *DataFrame) Geom {
	// No reparamization in fundamental geom.
	return t
}

func (t GeomText) Bounds(aes string, df *DataFrame) (min, max float64) {
	return ComputeBounds(aes, df, nil)
}

func (t GeomText) Render(plot *Plot, data *DataFrame, style AesMapping) []Grob {
	x, y, s := data.Columns["x"], data.Columns["y"], data.Columns["text"]
	xf, yf := plot.Scales["x"].Pos, plot.Scales["y"].Pos

	colFunc := makeColorFunc("color", data, plot, style)
	sizeFunc := makePosFunc("size", data, plot, style)
	alphaFunc := makePosFunc("alpha", data, plot, style)
	angleFunc := makePosFunc("angle", data, plot, style)

	grobs := make([]Grob, data.N)
	for i := 0; i < data.N; i++ {
		color := SetAlpha(colFunc(i), alphaFunc(i))
		text := s.String(s.Data[i])
		grob := GrobText{
			x:     xf(x.Data[i]),
			y:     yf(y.Data[i]),
			text:  text,
			color: color,
			size:  sizeFunc(i),
			angle: angleFunc(i),
		}
		grobs[i] = grob
	}
	return grobs
}

// -------------------------------------------------------------------------
// Geom Bar

type GeomBar struct {
	Style AesMapping // The individal fixed, aka non-mapped aesthetics
}

var _ Geom = GeomBar{}

func (b GeomBar) Name() string            { return "GeomBar" }
func (b GeomBar) NeededSlots() []string   { return []string{"x", "y"} }
func (b GeomBar) OptionalSlots() []string { return []string{"color", "size", "linetype", "alpha"} }

func (b GeomBar) Aes(plot *Plot) AesMapping {
	return MergeStyles(b.Style, plot.Theme.BarStyle, DefaultTheme.BarStyle)
}

func (b GeomBar) AdjustPosition(df *DataFrame, posAdj PositionAdjust) {
	// TODO
}

func (b GeomBar) Bounds(aes string, df *DataFrame) (min, max float64) {
	panic("Should not be called")
	return ComputeBounds(aes, df, nil)
}

func (b GeomBar) Reparametrize(df *DataFrame) Geom {
	xf := df.Columns["x"]
	if !df.Has("width") {
		width := xf.Resolution() * 0.9
		wf := xf.Const(width, df.N)
		df.Columns["width"] = wf
	}

	yf, wf := df.Columns["y"], df.Columns["width"]

	xmin, ymin := NewField(df.N), NewField(df.N)
	xmax, ymax := NewField(df.N), NewField(df.N)
	xmin.Type, ymin.Type, xmax.Type, ymax.Type = Float, Float, Float, Float

	for i := 0; i < df.N; i++ {
		if y := yf.Data[i]; y > 0 {
			ymin.Data[i] = 0
			ymax.Data[i] = y
		} else {
			ymin.Data[i] = y
			ymax.Data[i] = 0
		}
		x, wh := xf.Data[i], wf.Data[i]/2
		xmin.Data[i] = x - wh
		xmax.Data[i] = x + wh
	}
	df.Columns["xmin"] = xmin
	df.Columns["ymin"] = ymin
	df.Columns["xmax"] = xmax
	df.Columns["ymax"] = ymax
	df.Delete("width")
	df.Delete("x")
	df.Delete("y")

	return GeomRect{Style: b.Style.Copy()}
}

func (b GeomBar) Render(plot *Plot, data *DataFrame, style AesMapping) []Grob {
	panic("Bar has no own render") // TODO: ugly. Maybe remodel Geom inheritance
}

// -------------------------------------------------------------------------
// Geom Rect

type GeomRect struct {
	Style AesMapping // The individal fixed, aka non-mapped aesthetics
}

var _ Geom = GeomRect{}

func (r GeomRect) Name() string          { return "GeomRect" }
func (r GeomRect) NeededSlots() []string { return []string{"xmin", "ymin", "xmax", "ymax"} }
func (r GeomRect) OptionalSlots() []string {
	return []string{"color", "fill", "linetype", "alpha", "size"}
}

func (r GeomRect) Aes(plot *Plot) AesMapping {
	return MergeStyles(r.Style, plot.Theme.RectStyle, DefaultTheme.RectStyle)
}

func (r GeomRect) AdjustPosition(df *DataFrame, posAdj PositionAdjust) {
}

func (r GeomRect) Reparametrize(df *DataFrame) Geom {
	return r
}

func minOf(a, b float64) float64 {
	if math.IsNaN(a) {
		return b
	}
	if math.IsNaN(b) {
		return a
	}
	if a < b {
		return a
	}
	return b
}

func maxOf(a, b float64) float64 {
	if math.IsNaN(a) {
		return b
	}
	if math.IsNaN(b) {
		return a
	}
	if a > b {
		return a
	}
	return b
}

func ComputeBounds(aes string, df *DataFrame, combined map[string]string) (min, max float64) {
	min, max = math.NaN(), math.NaN()
	if len(combined) > 0 && combined[aes] != "" {
		for _, field := range strings.Split(combined[aes], ",") {
			low, high, _, _ := MinMax(df, field)
			min = minOf(min, low)
			max = maxOf(max, high)
		}
	} else {
		low, high, _, _ := MinMax(df, aes)
		min = minOf(min, low)
		max = maxOf(max, high)
	}
	return min, max
}

func (r GeomRect) Bounds(aes string, df *DataFrame) (min, max float64) {
	return ComputeBounds(aes, df, map[string]string{"x": "xmin,xmax", "y": "ymin,ymax"})
}

func (r GeomRect) Render(plot *Plot, data *DataFrame, style AesMapping) []Grob {
	xmin, ymin := data.Columns["xmin"].Data, data.Columns["ymin"].Data
	xmax, ymax := data.Columns["xmax"].Data, data.Columns["ymax"].Data
	xf, yf := plot.Scales["x"].Pos, plot.Scales["y"].Pos

	colFunc := makeColorFunc("color", data, plot, style)
	fillFunc := makeColorFunc("fill", data, plot, style)
	linetypeFunc := makeStyleFunc("linetype", data, plot, style)
	alphaFunc := makePosFunc("alpha", data, plot, style)
	sizeFunc := makePosFunc("size", data, plot, style)

	grobs := make([]Grob, 0)
	for i := 0; i < data.N; i++ {
		alpha := alphaFunc(i)
		if alpha == 0 {
			continue // Won't be visibale anyway....
		}

		// Coordinates of diagonal corners.
		x0, y0 := xf(xmin[i]), yf(ymin[i])
		x1, y1 := xf(xmax[i]), yf(ymax[i])
		// TODO: swap if wrong order

		rect := GrobRect{
			xmin: x0,
			ymin: y0,
			xmax: x1,
			ymax: y1,
			fill: SetAlpha(fillFunc(i), alpha),
		}
		grobs = append(grobs, rect)

		// Drown border only if linetype != blank.
		lt := LineType(linetypeFunc(i))
		if lt == BlankLine {
			continue
		}
		color := SetAlpha(colFunc(i), alpha)
		points := make([]struct{ x, y float64 }, 5)
		points[0].x, points[0].y = x0, y0
		points[1].x, points[1].y = x1, y0
		points[2].x, points[2].y = x1, y1
		points[3].x, points[3].y = x0, y1
		points[4].x, points[4].y = x0, y0
		border := GrobPath{
			points:   points,
			linetype: lt,
			color:    color,
			size:     sizeFunc(i),
		}
		grobs = append(grobs, border)
	}
	return grobs
}
