package plot

import (
	"code.google.com/p/plotinum/vg"
	"code.google.com/p/plotinum/vg/vgimg"
	"fmt"
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

	innerVP := SubViewport(allVP, 0.1, 0.1, 0.8, 0.8)
	fmt.Printf("All-VP: %s\n", allVP.String())
	fmt.Printf("av 0.1: %.2fin %.2fin\n", allVP.X(0.1).Inches(), allVP.Y(0.1).Inches())
	fmt.Printf("av 0.9: %.2fin %.2fin\n", allVP.X(0.9).Inches(), allVP.Y(0.9).Inches())
	fmt.Printf("\nInn-VP: %s\n", innerVP.String())
	fmt.Printf("iv 0.0: %.2fin %.2fin\n", innerVP.X(0).Inches(), innerVP.Y(0).Inches())
	fmt.Printf("iv 1.0: %.2fin %.2fin\n\n", innerVP.X(1).Inches(), innerVP.Y(1).Inches())

	bg := GrobRect{xmin: 0, ymin: 0, xmax: 1, ymax: 1, fill: BuiltinColors["gray80"]}
	bg.Draw(innerVP)

	println("av 0.1: ", innerVP.X(0.1).Inches(), "in   ", innerVP.Y(0.1).Inches(), "in")
	println("av 0.8: ", innerVP.X(0.8).Inches(), "in   ", innerVP.Y(0.8).Inches(), "in")

	println("")

	cols := []string{"red", "green", "blue", "cyan", "magenta", "yellow", "white", "gray", "black"}

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
	for _, grob := range points {
		grob.Draw(innerVP)
	}
	pngCanvas.WriteTo(file)
	file.Close()
}
