package plot

import (
	"fmt"
	"image/color"
	"math"
	"strings"
)

type Grob interface {
	Draw(vp Viewport)
	String() string
}

// -------------------------------------------------------------------------
// Grob Point

type GrobPoint struct {
	x, y  float64
	size  float64
	shape PointShape
	color color.Color
}

var _ Grob = GrobPoint{}

func (point GrobPoint) Draw(vp Viewport) {
}

func (point GrobPoint) String() string {
	return fmt.Sprintf("Point(%.3f,%.3f %s %s %.1f)",
		point.x, point.y, Color2String(point.color),
		point.shape.String(), point.size)
}

// -------------------------------------------------------------------------
// Grob Line

type GrobLine struct {
	x0, y0, x1, y1 float64
	size           float64
	linetype       LineType
	color          color.Color
}

var _ Grob = GrobLine{}

func (line GrobLine) Draw(vp Viewport) {
}

func (line GrobLine) String() string {
	return fmt.Sprintf("Line(%.3f,%.3f - %.3f,%.3f %s %s %.1f)",
		line.x0, line.y0, line.x1, line.y1,
		Color2String(line.color), line.linetype.String(),
		line.size)
}

// -------------------------------------------------------------------------
// Grob Path

type GrobPath struct {
	points   []struct{ x, y float64 }
	size     float64
	linetype LineType
	color    color.Color
}

var _ Grob = GrobPath{}

func (path GrobPath) Draw(vp Viewport) {
}

func (path GrobPath) String() string {
	// Pretty print points
	ppp := func(points []struct{ x, y float64 }) string {
		s := []string{}
		for _, p := range points {
			s = append(s, fmt.Sprintf("%.2f,%.2f", p.x, p.y))
		}
		return strings.Join(s, " - ")
	}
	var points string
	if n := len(path.points); n <= 6 {
		points = ppp(path.points)
	} else {
		points = ppp(path.points[0:3]) + " ... " + ppp(path.points[n-3:n])
	}
	return fmt.Sprintf("Path(%s %s %s %.1f)",
		points, Color2String(path.color), path.linetype.String(),
		path.size)
}

// -------------------------------------------------------------------------
// Grob Text

type GrobText struct {
	x, y  float64
	text  string
	size  float64
	color color.Color
	angle float64

	family, fontface string
	vjust, hjust     float64
	lineheight       float64
}

var _ Grob = GrobText{}

func (text GrobText) Draw(vp Viewport) {
}

func (text GrobText) String() string {
	just := "r" // TODO: check if this is the proper def.
	if text.vjust < 1/3 {
		just = "l"
	} else if text.vjust < 2/3 {
		just = "c"
	}
	if text.hjust < 1/3 {
		just += "t"
	} else if text.hjust < 2/3 {
		just += "m"
	} else {
		just += "b"
	}

	fnt := fmt.Sprintf("%s/%s/%.0f", strings.Replace(text.family, " ", "-", -1),
		text.fontface, text.lineheight)

	return fmt.Sprintf("Text(%.3f,%.3f %q %s %.0f° %s %q)",
		text.x, text.y, text.text, Color2String(text.color),
		180*text.angle/math.Pi, just, fnt)
}

// -------------------------------------------------------------------------
// Grob Rect

type GrobRect struct {
	xmin, ymin float64
	xmax, ymax float64
	fill       color.Color
}

var _ Grob = GrobRect{}

func (rect GrobRect) Draw(vp Viewport) {
}

func (rect GrobRect) String() string {
	return fmt.Sprintf("Rect(%.3f,%.3f - %.3f,%.3f %s)",
		rect.xmin, rect.ymin, rect.xmax, rect.ymax,
		Color2String(rect.fill))
}
