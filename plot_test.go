package plot

import (
	"code.google.com/p/plotinum/vg"
	"code.google.com/p/plotinum/vg/vgimg"
	"fmt"
	"image/color"
	"os"
	"testing"
)

func TestStatBin(t *testing.T) {
	df, _ := NewDataFrameFrom(measurement)
	sb := StatBin{BinWidth: 2, Drop: true}
	df.Rename("BMI", "x")
	bined := sb.Apply(df, nil)
	bined.Print(os.Stdout)

	sb = StatBin{BinWidth: 5, Drop: false}
	df.Rename("Age", "x")
	bined = sb.Apply(df, nil)
	bined.Print(os.Stdout)

}

func TestString2Color(t *testing.T) {
	tests := []struct {
		s string
		c color.Color
	}{
		{"#1256ab", color.NRGBA{0x12, 0x56, 0xab, 0xff}},
		{"#1256abcd", color.NRGBA{0x12, 0x56, 0xab, 0xcd}},
		{"red", color.NRGBA{0xff, 0x00, 0x00, 0xff}},
		{"green", color.NRGBA{0x00, 0xff, 0x00, 0xff}},
		{"blue", color.NRGBA{0x00, 0x00, 0xff, 0xff}},
		{"nonsens", color.NRGBA{0xaa, 0x66, 0x77, 0x7f}},
	}

	for i, tc := range tests {
		got := String2Color(tc.s)
		rg, gg, bg, ag := got.RGBA()
		rw, gw, bw, aw := tc.c.RGBA()
		if rg != rw || gg != gw || bg != bw || ag != aw {
			t.Errorf("%d %q: got %04X, %04X, %04X, %04X want %04X, %04X, %04X, %04X",
				i, tc.s, rw, gw, bw, aw, rg, gg, bg, ag)
		}
	}
}

func TestIndividualSteps(t *testing.T) {
	aes := AesMapping{
		"x": "Height",
		"y": "Weight",
	}
	plot, err := NewPlot(measurement, aes)
	if err != nil {
		t.Fatalf("Unxpected error: %s", err)
	}

	//
	// Add layers to plot
	//

	rawData := Layer{
		Name: "Raw Data",
		Stat: nil, // identity
		Geom: GeomPoint{
			Style: AesMapping{
				"color": "red",
				"shape": "diamond",
			}}}
	plot.Layers = append(plot.Layers, &rawData)

	linReg := Layer{
		Name: "Linear regression",
		Stat: &StatLinReq{},
		Geom: GeomABLine{
			Style: AesMapping{
				"color":    "green",
				"linetype": "dashed",
			},
		},
		// StatLinReq produces intercept/slope suitable for GeomABLine
		GeomMapping: nil,
	}
	plot.Layers = append(plot.Layers, &linReg)

	ageLabel := Layer{
		Name: "Age Label",
		DataMapping: AesMapping{
			"value": "Age",
		},
		Stat: &StatLabel{Format: "%.0f years"},
		Geom: GeomText{
			Style: AesMapping{
				"color":  "blue",
				"angle":  "45°",
				"family": "Helvetica",
			},
		},
		GeomMapping: nil,
	}
	plot.Layers = append(plot.Layers, &ageLabel)

	histogram := Layer{
		Name:        "Histogram",
		DataMapping: AesMapping{"y": ""}, // clear mapping of y to Height
		Stat:        &StatBin{Drop: true},
		StatMapping: AesMapping{
			"y": "count",
		},
		Geom: GeomBar{
			Style: AesMapping{
				"fill": "gray50",
			},
		},
	}
	plot.Layers = append(plot.Layers, &histogram)

	// Set up the one panel.
	plot.CreatePanels()
	if len(plot.Panels) != 1 {
		t.Fatalf("Got %d panel rows, expected 1.", len(plot.Panels))
	}
	if len(plot.Panels[0]) != 1 {
		t.Fatalf("Got %d panel cols, expected 1.", len(plot.Panels[0]))
	}
	panel := plot.Panels[0][0]

	// 2. PrepareData
	panel.PrepareData()
	if fields := panel.Layers[0].Data.FieldNames(); !same(fields, []string{"x", "y"}) {
		t.Errorf("Layer 0 DF has fields %v", fields)
	}
	if fields := panel.Layers[1].Data.FieldNames(); !same(fields, []string{"x", "y"}) {
		t.Errorf("Layer 1 DF has fields %v", fields)
	}
	if sx, ok := panel.Scales["x"]; !ok {
		t.Errorf("Missing x scale")
	} else {
		if sx.Discrete || sx.Transform != &IdentityScale || sx.Aesthetic != "x" {
			t.Errorf("Scale x = %+v", sx)
		}
	}
	if sy, ok := panel.Scales["y"]; !ok {
		t.Errorf("Missing y scale")
	} else {
		if sy.Discrete || sy.Transform != &IdentityScale || sy.Aesthetic != "y" {
			t.Errorf("Scale y = %+v", sy)
		}
	}

	// 3. ComputeStatistics
	panel.ComputeStatistics()

	// No statistic on layer 0: data field is unchanges
	if fields := panel.Layers[0].Data.FieldNames(); !same(fields, []string{"x", "y"}) {
		t.Errorf("Layer 0 DF has fields %v", fields)
	}
	// StatLinReq produces intercept and slope
	if fields := panel.Layers[1].Data.FieldNames(); !same(fields, []string{"intercept", "slope", "interceptErr", "slopeErr"}) {
		t.Errorf("Layer 1 DF has fields %v", fields)
	}
	data := panel.Layers[1].Data
	if data.N != 1 {
		t.Errorf("Got %d data in lin req df.", panel.Layers[1].Data.N)
	}
	t.Logf("Intercept = %.2f   Slope = %.2f",
		data.Columns["intercept"].Data[0],
		data.Columns["slope"].Data[0])
	// StatLabels produces labels
	if fields := panel.Layers[2].Data.FieldNames(); !same(fields, []string{"x", "y", "text"}) {
		t.Errorf("Layer 2 %q has fields %v", panel.Layers[3].Name, fields)
	}
	data = panel.Layers[2].Data
	if data.N != 20 {
		t.Errorf("Got %d data in label df.", panel.Layers[2].Data.N)
	}
	// StatBin produces bins
	if fields := panel.Layers[3].Data.FieldNames(); !same(fields, []string{"x", "count", "ncount", "density", "ndensity"}) {
		t.Errorf("Layer 3 %q has fields %v", panel.Layers[3].Name, fields)
	}
	data = panel.Layers[3].Data
	if data.N != 11 {
		t.Errorf("Got %d data in binned df.", panel.Layers[3].Data.N)
	}

	// 4. Wireing
	panel.WireStatToGeom()

	for a, s := range panel.Scales {
		fmt.Printf("====== Scale %s %q ========\n", a, s.Name)
		fmt.Printf("%s\n", s.String())
	}

	// 5. Test Construct ConstructGeoms. This shouldn't change much as
	// GeomABLine doesn't reparametrize and we don't do position adjustments.
	panel.ConstructGeoms()

	// 6. FinalizeScales
	panel.FinalizeScales()
	// Only x and y are set up
	if sx, ok := panel.Scales["x"]; !ok {
		t.Errorf("Missing x scale")
	} else {
		if sx.Pos == nil {
			t.Errorf("Missing Pos for x scale.")
		}
		if sx.DomainMin > 1.62 || sx.DomainMax < 1.95 {
			t.Errorf("Bad training: %f %f", sx.DomainMin, sx.DomainMax)
		}
	}
	fmt.Printf("%s\n", panel.Scales["x"])
	fmt.Printf("%s\n", panel.Scales["y"])

	// 7. Render Geoms to Grobs using scales (Step7).
	panel.RenderGeoms()
	fmt.Println("Layer 0, raw data")
	for _, grob := range panel.Layers[0].Grobs {
		fmt.Println("  ", grob.String())
	}
	fmt.Println("Layer 1, linear regression")
	for _, grob := range panel.Layers[1].Grobs {
		fmt.Println("  ", grob.String())
	}
	fmt.Println("Layer 2, labels")
	for _, grob := range panel.Layers[2].Grobs {
		fmt.Println("  ", grob.String())
	}
	fmt.Println("Layer 3, histogram")
	for _, grob := range panel.Layers[3].Grobs {
		fmt.Println("  ", grob.String())
	}

	// Output
	pngCanvas := vgimg.PngCanvas{Canvas: vgimg.New(800, 600)}
	vp := Viewport{
		X0:     50,
		Y0:     50,
		Width:  700,
		Height: 500,
		Canvas: pngCanvas,
	}
	file, err := os.Create("example.png")
	if err != nil {
		t.Fatalf("%", err)
	}

	fmt.Println("Layer 0, raw data")
	for _, grob := range panel.Layers[0].Grobs {
		grob.Draw(vp)
	}
	fmt.Println("Layer 1, linear regression")
	for _, grob := range panel.Layers[1].Grobs {
		grob.Draw(vp)
	}
	fmt.Println("Layer 2, labels")
	for _, grob := range panel.Layers[2].Grobs {
		grob.Draw(vp)
	}
	fmt.Println("Layer 3, histogram")
	for _, grob := range panel.Layers[3].Grobs {
		grob.Draw(vp)
	}
	pngCanvas.WriteTo(file)
	file.Close()

}

func TestSimplePlot(t *testing.T) {
	aes := AesMapping{
		"x": "Height",
		"y": "Weight",
	}
	plot, err := NewPlot(measurement, aes)
	if err != nil {
		t.Fatalf("Unxpected error: %s", err)
	}
	plot.Title = "Sample 12.3"

	rawData := Layer{
		Name: "Raw Data",
		Stat: nil, // identity
		Geom: GeomPoint{
			Style: AesMapping{
				"color": "red",
				"shape": "diamond",
			}}}
	plot.Layers = append(plot.Layers, &rawData)

	linReg := Layer{
		Name: "Linear regression",
		Stat: &StatLinReq{},
		Geom: GeomABLine{
			Style: AesMapping{
				"color":    "green",
				"linetype": "dashed",
			},
		},
		// StatLinReq produces intercept/slope suitable for GeomABLine
		GeomMapping: nil,
	}
	plot.Layers = append(plot.Layers, &linReg)

	file, err := os.Create("simple.png")
	if err != nil {
		t.Fatalf("%", err)
	}
	plot.Draw(vg.Inches(10), vg.Inches(8), file)
	file.Close()
}
