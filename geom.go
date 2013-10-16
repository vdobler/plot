package plot

import "fmt"

var _ = fmt.Printf

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
// Geom Bar

type GeomBar struct {
	Style AesMapping // The individal fixed, aka non-mapped aesthetics
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
