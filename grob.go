package plot

import (
	"image/color"
)

type Grob interface {
	Draw(vp Viewport)
}

// -------------------------------------------------------------------------
// Grob Point

type GrobPoint struct {
	x, y  float64
	size  float64
	shape PointShape
	color color.Color
}

func (point GrobPoint) Draw(vp Viewport) {
}

// -------------------------------------------------------------------------
// Grob Line

type GrobLine struct {
	x0, y0, x1, y1 float64
	size           float64
	linetype       LineType
	color          color.Color
}

func (line GrobLine) Draw(vp Viewport) {
}

// -------------------------------------------------------------------------
// Grob Path

type GrobPath struct {
	points   []struct{ x, y float64 }
	size     float64
	linetype LineType
	color    color.Color
}

func (line GrobPath) Draw(vp Viewport) {
}
