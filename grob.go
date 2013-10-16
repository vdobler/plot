package plot

import (
	"fmt"
	"image/color"
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
