package plot

import (
	"code.google.com/p/plotinum/vg"
	"code.google.com/p/plotinum/vg/vgimg"
	// "fmt"
	"os"
	"testing"
)

func TestGrobs(t *testing.T) {
	// Output
	file, err := os.Create("grobs.png")
	if err != nil {
		t.Fatalf("%", err)
	}

	pngCanvas := vgimg.PngCanvas{Canvas: vgimg.New(vg.Inches(10), vg.Inches(8))}
	vg.Initialize(pngCanvas)

	allVP := Viewport{
		X0:     0,
		Y0:     0,
		Width:  vg.Inches(10),
		Height: vg.Inches(8),
		Canvas: pngCanvas,
	}
	innerVP := SubViewport(allVP, 0.05, 0.05, 0.9, 0.9)
	bg := GrobRect{xmin: 0, ymin: 0, xmax: 1, ymax: 1, fill: BuiltinColors["gray80"]}
	bg.Draw(innerVP)

	cols := []string{"red", "green", "blue", "cyan", "magenta", "yellow",
		"white", "gray", "black"}

	// Draw points in all shapes, three sizes and all builtin colors.
	points := []Grob{}
	x, y := 0.1, 0.1
	for shape := DotPoint; shape <= StarPoint; shape++ {
		for size := 2; size < 7; size += 2 {
			y = 0.05
			for _, col := range cols {
				g := GrobPoint{
					x:     x,
					y:     y,
					size:  float64(size),
					shape: shape,
					color: BuiltinColors[col],
				}
				points = append(points, g)
				y += 0.035
			}
			x += 0.021
		}
	}
	x, y = 0.02, 0.05
	for _, col := range cols {
		g := GrobText{
			x:     x,
			y:     y,
			text:  col,
			size:  10,
			color: BuiltinColors[col],
			vjust: 0.5,
			hjust: 0,
		}
		points = append(points, g)
		y += 0.035
	}
	x, y = 0.121, 0.36
	for shape := DotPoint; shape <= StarPoint; shape++ {
		dy := float64(shape%2) * 0.015
		g := GrobText{
			x:     x,
			y:     y + dy,
			text:  shape.String(),
			size:  10,
			color: BuiltinColors["black"],
			vjust: 0.5,
			hjust: 0.5,
		}
		points = append(points, g)
		x += 3 * 0.021
	}
	for _, grob := range points {
		grob.Draw(innerVP)
	}

	lines := []Grob{}
	x, y = 0.1, 0.45
	for lt := SolidLine; lt <= TwodashLine; lt++ {
		x = 0.1
		for size := 1; size < 8; size += 2 {
			g := GrobLine{
				x0:       x,
				y0:       y,
				x1:       x + 0.18,
				y1:       y,
				size:     float64(size),
				linetype: lt,
				color:    BuiltinColors["black"],
			}
			lines = append(lines, g)
			x += 0.22
		}
		y += 0.04
	}
	for _, grob := range lines {
		grob.Draw(innerVP)
	}

	pngCanvas.WriteTo(file)
	file.Close()
}
