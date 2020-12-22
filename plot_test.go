package plot

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"
	"os"
	"testing"

	"gonum.org/v1/plot/vg/vgimg"
)

func same(s []string, t []string) bool {
	ss := NewStringSetFrom(s)
	return ss.Equals(t)
}

func TestStatBin(t *testing.T) {
	pool := NewStringPool()
	df, _ := NewDataFrameFrom(measurement, pool)
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
				"angle":  "0.7", // "45°",  // TODO: parsing ° fails
				"family": "Helvetica",
				"size":   "10", // TODO: should come from DefaultTheme
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
	// pngCanvas.Translate(vg.Point{-400, -300})
	vp := Viewport{
		X0:     50,
		Y0:     50,
		Width:  700,
		Height: 500,
		Canvas: pngCanvas,
	}
	file, err := os.Create("example.png")
	if err != nil {
		t.Fatalf("%s", err)
	}

	panel.Draw(vp, true, true)
	if false {
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

	function := Layer{
		Name: "Sinus",
		Stat: &StatFunction{
			F: func(x float64) float64 {
				return 10*math.Sin(40*x) + 75
			},
		},
		Geom: GeomLine{
			Style: AesMapping{
				"color":    "blue",
				"linetype": "solid",
				"size":     "1",
			},
		},
		GeomMapping: nil,
	}
	plot.Layers = append(plot.Layers, &function)

	plot.WritePNG("simple.png", 800, 600)
}

func TestFaceting(t *testing.T) {
	diamonds, err := ReadDiamonds("data/alldiamonds.csv")
	if err != nil {
		t.Fatalf("Unxpected error: %s", err)
	}

	aes := AesMapping{
		"x": "Carat",
	}
	plot, err := NewPlot(diamonds, aes)
	if err != nil {
		t.Fatalf("Unxpected error: %s", err)
	}
	plot.Title = "Diamonds"
	// plot.Data.Print(os.Stdout)

	plot.Faceting = Faceting{
		Columns:   "Color",
		Rows:      "Cut",
		FreeScale: "y",
	}

	hist := Layer{
		Name: "Histogram",
		Stat: StatBin{
			Drop: true,
		},
		StatMapping: AesMapping{
			"y": "count",
		},
		Geom: GeomBar{
			Style: AesMapping{
				"fill": "gray20",
			}}}
	plot.Layers = append(plot.Layers, &hist)
	plot.WritePNG("hist.png", 800, 600)
}

func TestDiscreteXScale(t *testing.T) {
	aes := AesMapping{
		"x": "Country",
		"y": "BMI",
	}
	plot, err := NewPlot(measurement, aes)
	if err != nil {
		t.Fatalf("Unxpected error: %s", err)
	}
	plot.Title = "Discrete x scale"

	rawData := Layer{
		Name: "Raw Data",
		Stat: nil, // identity
		Geom: GeomPoint{},
	}
	plot.Layers = append(plot.Layers, &rawData)

	box := Layer{
		Name: "Boxplot",
		Stat: StatBoxplot{},
		Geom: GeomBoxplot{},
	}
	plot.Layers = append(plot.Layers, &box)

	plot.WritePNG("discrx.png", 800, 600)
}

func TestBoxplot(t *testing.T) {
	type d struct {
		x string
		y float64
		t int
		s string
	}
	data := make([]d, 120)
	for i := 0; i < 20; i++ {
		data[i].x = "1"
		data[i].y = rand.NormFloat64()*5 + 10
		data[i].t = 2

		data[20+i].x = "2"
		data[20+i].y = rand.NormFloat64()*2 + 5
		data[20+i].t = 3

		data[40+i].x = "3"
		data[40+i].y = rand.NormFloat64()*10 + 5
		data[40+i].t = 5

		data[60+i].x = "2"
		data[60+i].y = rand.NormFloat64()*3 + 8
		data[60+i].t = 5

		data[80+i].x = "3"
		data[80+i].y = rand.NormFloat64()*4 + 4
		data[80+i].t = 7

		data[100+i].x = "2"
		data[100+i].y = rand.NormFloat64()*5 + 0
		data[100+i].t = 10
	}
	// Produce some outliers.
	data[20].y = 15
	data[21].y = -5
	data[105].y = 24
	data[106].y = 25

	aes := AesMapping{
		"x":    "x",
		"y":    "y",
		"fill": "t",
	}
	plot, err := NewPlot(data, aes)
	if err != nil {
		t.Fatalf("Unxpected error: %s", err)
	}
	plot.Title = "Boxplot"

	box := Layer{
		Name: "Boxplot",
		Stat: StatBoxplot{},
		Geom: GeomBoxplot{Position: PosDodge},
	}
	plot.Layers = append(plot.Layers, &box)

	data1 := []d{
		{"1", 0, 1, "a"},
		{"1", 5, 2, "b"},
		{"1", 10, 3, "x"},
		{"2", 2, 4, "y"},
		{"2", 7, 5, "a"},
		{"2", 12, 6, "b"},
		{"3", 4, 7, "x"},
		{"3", 9, 8, "y"},
		{"3", 14, 9, "f"},
	}
	df, _ := NewDataFrameFrom(data1, plot.Pool)
	points := Layer{
		Name: "Points",
		Data: df,
		DataMapping: AesMapping{
			"x":     "x",
			"y":     "y",
			"shape": "s",
		},
		Geom: GeomPoint{
			Style: AesMapping{
				"color": "#ff00ff",
				"size":  "10",
			},
		},
	}
	plot.Layers = append(plot.Layers, &points)

	plot.WritePNG("boxplot.png", 800, 600)
}
