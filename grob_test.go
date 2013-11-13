package plot

import (
	"code.google.com/p/plotinum/vg"
	"code.google.com/p/plotinum/vg/vgimg"
	"fmt"
	"math"
	"os"
	"testing"
)

func TestGraphicGrobs(t *testing.T) {
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
	innerVP := allVP.Sub(0.05, 0.05, 0.9, 0.9)
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

	// Draw lines with different styles and widths.
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

	// Draw rectangles
	rectVP := innerVP.Sub(0.1, 0.7, 0.4, 0.3)
	rect := []Grob{}
	bgr := GrobRect{xmin: 0, ymin: 0, xmax: 1, ymax: 1, fill: BuiltinColors["gray40"]}
	bgr.Draw(rectVP)
	x, y = 0.0, 0.0
	w, h := 0.5, 0.5
	for _, col := range cols {
		g := GrobRect{
			xmin: x,
			ymin: y,
			xmax: x + w,
			ymax: y + h,
			fill: BuiltinColors[col],
		}
		rect = append(rect, g)
		x += w
		y += h
		w /= 2
		h /= 2
	}
	for _, grob := range rect {
		grob.Draw(rectVP)
	}

	// Draw path
	pathVP := innerVP.Sub(0.55, 0.7, 0.4, 0.3)
	bgp := GrobRect{xmin: 0, ymin: 0, xmax: 1, ymax: 1, fill: BuiltinColors["gray"]}
	bgp.Draw(pathVP)
	sin := make([]struct{ x, y float64 }, 50)
	for i := range sin {
		k := float64(i) / float64(len(sin)-1)
		x = k * 2 * math.Pi
		println(float64(i)/float64(len(sin)), x, math.Sin(x))
		y = 0.4 * math.Sin(x)
		sin[i].x = k
		sin[i].y = 0.55 + y
	}
	g := GrobPath{
		points:   sin,
		size:     4,
		linetype: SolidLine,
		color:    BuiltinColors["white"],
	}
	g.Draw(pathVP)
	cos := make([]struct{ x, y float64 }, 25)
	for i := range cos {
		k := float64(i) / float64(len(cos)-1)
		x = k * 4 * math.Pi
		y = 0.3 * math.Cos(x)
		cos[i].x = k
		cos[i].y = 0.45 + y
	}
	g = GrobPath{
		points:   cos,
		size:     2,
		linetype: DottedLine,
		color:    BuiltinColors["green"],
	}
	g.Draw(pathVP)

	pngCanvas.WriteTo(file)
	file.Close()
}

func drawTextGrid(vp Viewport, angle float64) {
	black := BuiltinColors["black"]
	white := BuiltinColors["white"]
	GrobLine{x0: 0, y0: 0, x1: 1, y1: 0, size: 1, linetype: SolidLine, color: white}.Draw(vp)
	GrobLine{x0: 0, y0: 0.5, x1: 1, y1: 0.5, size: 1, linetype: SolidLine, color: white}.Draw(vp)
	GrobLine{x0: 0, y0: 1, x1: 1, y1: 1, size: 1, linetype: SolidLine, color: white}.Draw(vp)
	GrobLine{x0: 0, y0: 0, x1: 0, y1: 1, size: 1, linetype: SolidLine, color: white}.Draw(vp)
	GrobLine{x0: 0.5, y0: 0, x1: 0.5, y1: 1, size: 1, linetype: SolidLine, color: white}.Draw(vp)
	GrobLine{x0: 1, y0: 0, x1: 1, y1: 1, size: 1, linetype: SolidLine, color: white}.Draw(vp)

	for _, vjust := range []float64{0, 0.5, 1} {
		size := 10.0 // 10.0 + 4*vjust
		for _, hjust := range []float64{0, 0.5, 1} {
			fname := ""
			if hjust == 0.5 {
				fname = "Helvetica-Bold"
			} else if hjust == 1 {
				fname = "Times-Bold"
			}
			t := GrobText{
				x:     hjust,
				y:     vjust,
				text:  fmt.Sprintf("%.1f/%.1f", hjust, vjust),
				size:  size,
				color: black,
				vjust: vjust,
				hjust: hjust,
				angle: angle,
				font:  fname,
			}
			t.Draw(vp)
		}
	}
}

func TestTextGrobs(t *testing.T) {
	// Output
	file, err := os.Create("text.png")
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
	bg := GrobRect{xmin: 0, ymin: 0, xmax: 1, ymax: 1, fill: BuiltinColors["gray60"]}
	bg.Draw(allVP)

	gridVP := allVP.Sub(0.1, 0.1, 0.35, 0.35)
	drawTextGrid(gridVP, 0)
	gridVP = allVP.Sub(0.55, 0.1, 0.35, 0.35)
	drawTextGrid(gridVP, 45./180*math.Pi)
	gridVP = allVP.Sub(0.1, 0.55, 0.35, 0.35)
	drawTextGrid(gridVP, 135./180*math.Pi)
	gridVP = allVP.Sub(0.55, 0.55, 0.35, 0.35)
	drawTextGrid(gridVP, 90./180*math.Pi)

	pngCanvas.WriteTo(file)
	file.Close()
}
