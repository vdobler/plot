package plot

import (
	"fmt"
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

	// Construct Geoms (Step 5)
	Construct(df *DataFrame, p *Panel) []Fundamental

	// Render interpretes data as the specific geom and produces Grobs.
	// TODO: Grouping?
	Render(p *Panel, data *DataFrame, aes AesMapping) []Grob
}

// TODO use one method returtning a GeomInfo instead of lots of methods.
type GeomInfo struct {
	Name     string   // Name of this Geom
	Needed   []string // The needed slots to construct this geom.
	Optional []string // The optional slots this geom understands.
}

// trainScales is a helper for geom construction: The some scales of p are
// trained on some fields of data. The spec arguments is of the form
//     "x:xmin,xmax y:ylow,yhigh"
// and determines which scales (here x and y) are trained on which fields.
func trainScales(p *Panel, data *DataFrame, spec string) {
	for _, scaleSpec := range strings.Split(spec, " ") {
		t := strings.Split(scaleSpec, ":")
		scaleName := t[0]
		scale, ok := p.Scales[scaleName]
		if !ok {
			continue
		}
		fields := strings.Split(t[1], ",")
		for _, field := range fields {
			if !data.Has(field) {
				continue
			}
			scale.Train(data.Columns[field])
		}
	}
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

// -------------------------------------------------------------------------
// Geom Point

type GeomPoint struct {
	Position PositionAdjust
	Style    AesMapping // The individal fixed, aka non-mapped aesthetics
}

var _ Geom = GeomPoint{}

func (p GeomPoint) Name() string            { return "GeomPoint" }
func (p GeomPoint) NeededSlots() []string   { return []string{"x", "y"} }
func (p GeomPoint) OptionalSlots() []string { return []string{"color", "size", "shape", "alpha"} }

func (p GeomPoint) Aes(plot *Plot) AesMapping {
	return MergeStyles(p.Style, plot.Theme.PointStyle, DefaultTheme.PointStyle)
}

func (p GeomPoint) Construct(df *DataFrame, panel *Panel) []Fundamental {
	// TODO: Handle p.Position == Jitter
	return []Fundamental{
		Fundamental{
			Geom: p,
			Data: df,
		}}
}

func (p GeomPoint) Render(panel *Panel, data *DataFrame, style AesMapping) []Grob {
	points := make([]GrobPoint, data.N)
	x, y := data.Columns["x"], data.Columns["y"]
	xf, yf := panel.Scales["x"].Pos, panel.Scales["y"].Pos

	colFunc := makeColorFunc("color", data, panel, style)
	sizeFunc := makePosFunc("size", data, panel, style, 1, 10)
	alphaFunc := makePosFunc("alpha", data, panel, style, 0, 1)
	shapeFunc := makeStyleFunc("shape", data, panel, style)

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

func (p GeomLine) Name() string          { return "GeomLine" }
func (p GeomLine) NeededSlots() []string { return []string{"x", "y"} }
func (p GeomLine) OptionalSlots() []string {
	return []string{"color", "size", "linetype", "alpha", "group"}
}

func (p GeomLine) Aes(plot *Plot) AesMapping {
	return MergeStyles(p.Style, plot.Theme.LineStyle, DefaultTheme.LineStyle)
}

func (p GeomLine) Construct(df *DataFrame, panel *Panel) []Fundamental {
	return []Fundamental{
		Fundamental{
			Geom: p,
			Data: df,
		}}
}

func (p GeomLine) Render(panel *Panel, data *DataFrame, style AesMapping) []Grob {
	scaleX, scaleY := panel.Scales["x"], panel.Scales["y"]
	grobs := make([]Grob, 0)

	var partitions []*DataFrame
	var levels []float64
	if data.Has("group") {
		groups := Levels(data, "group")
		levels = groups.Elements()
		partitions = Partition(data, "group", levels)
	} else {
		partitions = []*DataFrame{data}
	}

	for _, part := range partitions {
		x, y := part.Columns["x"], part.Columns["y"]
		colFunc := makeColorFunc("color", part, panel, style)
		sizeFunc := makePosFunc("size", part, panel, style, 0, 1)
		alphaFunc := makePosFunc("alpha", part, panel, style, 0, 1)
		typeFunc := makeStyleFunc("linetype", part, panel, style)
		if part.Has("color") || part.Has("size") ||
			part.Has("alpha") || part.Has("linetype") {
			// Some of the optional aesthetics are mapped (not set).
			// Cannot represent safely as a GrobPath; thus use lots
			// of GrobLine.
			// TODO: instead "of by one" why not use average?
			for i := 0; i < part.N-1; i++ {
				line := GrobLine{
					x0:       scaleX.Pos(x.Data[i]),
					y0:       scaleY.Pos(y.Data[i]),
					x1:       scaleX.Pos(x.Data[i+1]),
					y1:       scaleY.Pos(y.Data[i+1]),
					color:    SetAlpha(colFunc(i), alphaFunc(i)),
					size:     sizeFunc(i),
					linetype: LineType(typeFunc(i)),
				}
				grobs = append(grobs, line)
			}
		} else {
			// All segemtns have same color, linetype and size, use a GrobPath
			points := make([]struct{ x, y float64 }, part.N)
			for i := 0; i < part.N; i++ {
				points[i].x = scaleX.Pos(x.Data[i])
				points[i].y = scaleY.Pos(y.Data[i])
			}
			path := GrobPath{
				points:   points,
				color:    SetAlpha(colFunc(0), alphaFunc(0)),
				size:     sizeFunc(0),
				linetype: LineType(typeFunc(0)),
			}
			grobs = append(grobs, path)
		}
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

func (p GeomABLine) Construct(df *DataFrame, panel *Panel) []Fundamental {
	// Only scale training as rendering an abline is dead simple.

	ic, sc := df.Columns["intercept"].Data, df.Columns["slope"].Data
	scaleX, scaleY := panel.Scales["x"], panel.Scales["y"]
	xmin, xmax := scaleX.DomainMin, scaleX.DomainMax

	for i := 0; i < df.N; i++ {
		intercept, slope := ic[i], sc[i]
		ymin := slope*xmin + intercept
		ymax := slope*xmax + intercept
		scaleY.TrainByValue(ymin, ymax)
	}

	return []Fundamental{
		Fundamental{
			Geom: p,
			Data: df,
		}}
}

func (p GeomABLine) Render(panel *Panel, data *DataFrame, style AesMapping) []Grob {
	ic, sc := data.Columns["intercept"].Data, data.Columns["slope"].Data
	grobs := make([]Grob, data.N)
	colFunc := makeColorFunc("color", data, panel, style)
	sizeFunc := makePosFunc("size", data, panel, style, 0, 1)
	alphaFunc := makePosFunc("alpha", data, panel, style, 0, 1)
	typeFunc := makeStyleFunc("linetype", data, panel, style)

	scaleX, scaleY := panel.Scales["x"], panel.Scales["y"]
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

func (t GeomText) Construct(df *DataFrame, panel *Panel) []Fundamental {
	// Only scale training
	x, y := df.Columns["x"].Data, df.Columns["y"].Data
	sx, sy := panel.Scales["x"], panel.Scales["y"]
	for i := 0; i < df.N; i++ {
		sx.TrainByValue(x[i])
		sy.TrainByValue(y[i])
	}
	return []Fundamental{
		Fundamental{
			Geom: t,
			Data: df,
		}}
}

func (t GeomText) Render(panel *Panel, data *DataFrame, style AesMapping) []Grob {
	x, y, s := data.Columns["x"], data.Columns["y"], data.Columns["text"]
	xf, yf := panel.Scales["x"].Pos, panel.Scales["y"].Pos

	colFunc := makeColorFunc("color", data, panel, style)
	sizeFunc := makePosFunc("size", data, panel, style, 0, 1)
	alphaFunc := makePosFunc("alpha", data, panel, style, 0, 1)
	angleFunc := makePosFunc("angle", data, panel, style, 0, 1)

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
	Style    AesMapping // The individal fixed, aka non-mapped aesthetics
	Position PositionAdjust
}

var _ Geom = GeomBar{}

func (b GeomBar) Name() string            { return "GeomBar" }
func (b GeomBar) NeededSlots() []string   { return []string{"x", "y"} }
func (b GeomBar) OptionalSlots() []string { return []string{"color", "size", "linetype", "alpha"} }

func (b GeomBar) Aes(plot *Plot) AesMapping {
	return MergeStyles(b.Style, plot.Theme.BarStyle, DefaultTheme.BarStyle)
}

func (b GeomBar) Construct(df *DataFrame, panel *Panel) []Fundamental {
	xf := df.Columns["x"]
	xd := xf.Data
	if !df.Has("width") {
		width := xf.Resolution() * 0.9 // TODO: read from style
		println("GeomBar: no width in data frame, will use", width)
		wf := xf.Const(width, df.N)
		df.Columns["width"] = wf
	}
	yd, wd := df.Columns["y"].Data, df.Columns["width"].Data

	pool := df.Pool
	xminf, yminf := NewField(df.N, Float, pool), NewField(df.N, Float, pool)
	xmaxf, ymaxf := NewField(df.N, Float, pool), NewField(df.N, Float, pool)
	xmin, ymin := xminf.Data, yminf.Data
	xmax, ymax := xmaxf.Data, ymaxf.Data

	runningYmax := make(map[float64]float64)
	barsAt := make(map[float64]float64) // Number of bars at each x pos.
	for i := 0; i < df.N; i++ {
		if y := yd[i]; y > 0 {
			ymin[i] = 0
			ymax[i] = y
		} else {
			ymin[i] = y
			ymax[i] = 0
		}
		x, wh := xd[i], wd[i]/2
		xmin[i] = x - wh
		xmax[i] = x + wh

		switch b.Position {
		case PosStack, PosFill:
			r := runningYmax[x]
			h := ymax[i] - ymin[i]
			runningYmax[x] = r + h
			ymax[i] += r
			ymin[i] += r
		case PosDodge:
			barsAt[x] = barsAt[x] + 1
		}
	}

	switch b.Position {
	case PosFill:
		for x, sum := range runningYmax {
			for i := 0; i < df.N; i++ {
				if x != xd[i] {
					continue
				}
				ymin[i] /= sum
				ymax[i] /= sum
			}
		}
	case PosDodge:
		/******
		     +------------------- width -----------------+
		n=3  |--------------|--------------|----- we ----|
		n=4  |----------|----------|----------|----------|
		     +-------- wh --------+X
		               ********/
		for x, n := range barsAt {
			j := 0.0
			for i := 0; i < df.N; i++ {
				if x != xd[i] {
					continue
				}
				wh := wd[i] / 2
				we := wd[i] / n
				xmin[i] = x - wh + j*we
				xmax[i] = x - wh + (j+1)*we
				j++
			}
		}
	}

	df.Columns["xmin"] = xminf
	df.Columns["ymin"] = yminf
	df.Columns["xmax"] = xmaxf
	df.Columns["ymax"] = ymaxf
	df.Delete("width")
	df.Delete("x")
	df.Delete("y")

	trainScales(panel, df, "x:xmin,xmax y:ymin,ymax")
	// TODO: fill, color, .. too?

	return []Fundamental{
		Fundamental{
			Geom: GeomRect{
				Style: b.Style.Copy(),
			},
			Data: df,
		}}
}

func (b GeomBar) Render(panel *Panel, data *DataFrame, style AesMapping) []Grob {
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

func (r GeomRect) Construct(df *DataFrame, panel *Panel) []Fundamental {
	trainScales(panel, df, "x:xmin,xmax y:ymin,ymax")
	// TODO: optional fields too?
	return []Fundamental{
		Fundamental{
			Geom: r,
			Data: df,
		}}
}

func (r GeomRect) Render(panel *Panel, data *DataFrame, style AesMapping) []Grob {
	xmin, ymin := data.Columns["xmin"].Data, data.Columns["ymin"].Data
	xmax, ymax := data.Columns["xmax"].Data, data.Columns["ymax"].Data
	xf, yf := panel.Scales["x"].Pos, panel.Scales["y"].Pos

	colFunc := makeColorFunc("color", data, panel, style)
	fillFunc := makeColorFunc("fill", data, panel, style)
	linetypeFunc := makeStyleFunc("linetype", data, panel, style)
	alphaFunc := makePosFunc("alpha", data, panel, style, 0, 1)
	sizeFunc := makePosFunc("size", data, panel, style, 0, 1)

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
		// fmt.Printf("GeomRect: %d %v rect = %s\n", i, fillFunc(i), rect.String())

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
		// fmt.Printf("GeomRect: border = %s\n", border.String())
	}

	return grobs
}

// -------------------------------------------------------------------------
// Geom Boxplot

type GeomBoxplot struct {
	Style    AesMapping // The individal fixed, aka non-mapped aesthetics
	Position PositionAdjust
}

var _ Geom = GeomBoxplot{}

func (b GeomBoxplot) Name() string { return "GeomBoxplot" }
func (b GeomBoxplot) NeededSlots() []string {
	return []string{"x", "min", "low", "mid", "high", "max"}
}
func (b GeomBoxplot) OptionalSlots() []string { return []string{"fill"} }

func (b GeomBoxplot) Aes(plot *Plot) AesMapping {
	return MergeStyles(b.Style, plot.Theme.RectStyle, DefaultTheme.RectStyle)
}

func (b GeomBoxplot) Construct(data *DataFrame, panel *Panel) []Fundamental {
	// min, max := data.Columns["min"].Data, data.Columns["max"].Data
	low, high := data.Columns["low"].Data, data.Columns["high"].Data
	q1, q3 := data.Columns["q1"].Data, data.Columns["q3"].Data
	x, mid := data.Columns["x"].Data, data.Columns["mid"].Data
	outf := data.Columns["outliers"]

	width := 0.9 // TODO: determine from data

	rects := NewDataFrame("Rects of Boxplot of "+data.Name, data.Pool)
	rects.N = data.N
	ymin := NewField(data.N, Float, data.Pool)
	ymax := NewField(data.N, Float, data.Pool)
	xmin := NewField(data.N, Float, data.Pool)
	xmax := NewField(data.N, Float, data.Pool)

	lines := NewDataFrame("Lines of Boxplot of "+data.Name, data.Pool)
	lines.N = 6 * data.N
	xx := NewField(6*data.N, Float, data.Pool)
	yy := NewField(6*data.N, Float, data.Pool)
	gg := NewField(6*data.N, Int, data.Pool)

	outliers := NewDataFrame("Outliers of Boxplot of "+data.Name, data.Pool)
	ox := NewField(0, Float, data.Pool)
	oy := NewField(0, Float, data.Pool)

	// Count how many bars are draw at each x value.
	barsAt := make(map[float64]float64)
	drawnAt := make(map[float64]float64)
	for i := 0; i < data.N; i++ {
		barsAt[x[i]]++
	}

	for i := 0; i < data.N; i++ {
		i = int(i)

		xc := x[i]
		wh := width / 2

		if b.Position == PosDodge {
			total := barsAt[xc]
			drawn := drawnAt[xc]
			drawnAt[xc]++
			wh /= total
			xc += (2*drawn - (total - 1)) * wh
		}

		xmin.Data[i], xmax.Data[i] = xc-wh, xc+wh

		y1, y3 := q1[i], q3[i]
		ymin.Data[i], ymax.Data[i] = y1, y3

		yl, yh := low[i], high[i]
		xx.Data[6*i], xx.Data[6*i+1] = xc, xc
		xx.Data[6*i+2], xx.Data[6*i+3] = xc, xc
		yy.Data[6*i], yy.Data[6*i+1] = yl, y1
		yy.Data[6*i+2], yy.Data[6*i+3] = y3, yh

		xx.Data[6*i+4], xx.Data[6*i+5] = xc-wh, xc+wh
		yy.Data[6*i+4], yy.Data[6*i+5] = mid[i], mid[i]

		group := float64(3 * i)
		gg.Data[6*i], gg.Data[6*i+1] = group, group
		gg.Data[6*i+2], gg.Data[6*i+3] = group+1, group+1
		gg.Data[6*i+4], gg.Data[6*i+5] = group+2, group+2

		for _, q := range outf.GetVec(i) {
			ox.Data = append(ox.Data, xc)
			oy.Data = append(oy.Data, q)
		}
	}

	rects.Columns["xmin"] = xmin
	rects.Columns["xmax"] = xmax
	rects.Columns["ymin"] = ymin
	rects.Columns["ymax"] = ymax
	if data.Has("fill") {
		rects.Columns["fill"] = data.Columns["fill"]
	}

	lines.Columns["x"] = xx
	lines.Columns["y"] = yy
	lines.Columns["group"] = gg

	outliers.Columns["x"] = ox
	outliers.Columns["y"] = oy
	outliers.N = len(ox.Data)

	trainScales(panel, rects, "x:xmin,xmax")
	trainScales(panel, lines, "y:yy") // Bug: should train on real ymin/max

	outlierStyle := b.Style.Copy()
	outlierStyle["color"] = "#aa0000"
	outlierStyle["shape"] = "star"

	return []Fundamental{
		Fundamental{
			Geom: GeomRect{
				Style: b.Style.Copy(),
			},
			Data: rects,
		},
		Fundamental{
			Geom: GeomLine{
				Style: b.Style.Copy(),
			},
			Data: lines,
		},
		Fundamental{
			Geom: GeomPoint{
				Style: outlierStyle,
			},
			Data: outliers,
		},
	}
}

func (b GeomBoxplot) Render(panel *Panel, data *DataFrame, style AesMapping) []Grob {
	panic("Should not be called...")
}
