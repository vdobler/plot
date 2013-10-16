package plot

import (
	"image/color"
	"os"
	"testing"
)

func TestFaceting(t *testing.T) {
	df, _ := NewDataFrameFrom(measurement)

	fac := Faceting{
		Columns: "Group",
		Rows:    "Origin",
		Totals:  true,
	}

	p := Plot{
		Data:     df,
		Faceting: fac,
	}

	p.Draw()
}

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
	df, _ := NewDataFrameFrom(measurement)
	plot := &Plot{
		Data: df,
		Aes: AesMapping{
			"x": "Height",
			"y": "Weight",
		},
		Layers: []*Layer{
			&Layer{
				Name: "Raw Data",
				Stat: nil, // identity
				Geom: GeomPoint{
					Style: AesMapping{
						"color": "red",
						"shape": "diamond",
					},
				},
			},
			&Layer{
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
			},
		},
		Scales: make(map[string]*Scale),
	}

	for i := range plot.Layers {
		plot.Layers[i].Plot = plot
	}

	// Test PrepareData
	plot.Layers[0].PrepareData()
	if fields := plot.Layers[0].Data.FieldNames(); !same(fields, []string{"x", "y"}) {
		t.Errorf("Layer 0 DF has fields %v", fields)
	}
	plot.Layers[1].PrepareData()
	if fields := plot.Layers[1].Data.FieldNames(); !same(fields, []string{"x", "y"}) {
		t.Errorf("Layer 1 DF has fields %v", fields)
	}
	if sx, ok := plot.Scales["x"]; !ok {
		t.Errorf("Missing x scale")
	} else {
		if sx.Discrete || sx.Transform != nil || sx.Type != "x" {
			t.Errorf("Scale x = %+v", sx)
		}
	}
	if sy, ok := plot.Scales["y"]; !ok {
		t.Errorf("Missing y scale")
	} else {
		if sy.Discrete || sy.Transform != nil || sy.Type != "y" {
			t.Errorf("Scale y = %+v", sy)
		}
	}

	// Test ComputeStatistics
	// No statistic on layer 0: data field is unchanges
	plot.Layers[0].ComputeStatistics()
	if fields := plot.Layers[0].Data.FieldNames(); !same(fields, []string{"x", "y"}) {
		t.Errorf("Layer 0 DF has fields %v", fields)
	}

	// StatLinReq produces intercept and slope
	plot.Layers[1].ComputeStatistics()
	if fields := plot.Layers[1].Data.FieldNames(); !same(fields, []string{"intercept", "slope", "interceptErr", "slopeErr"}) {
		t.Errorf("Layer 1 DF has fields %v", fields)
	}
	data := plot.Layers[1].Data
	if data.N != 1 {
		t.Errorf("Got %d data in lin req df.", plot.Layers[1].Data.N)
	}
	t.Logf("Intercept = %.2f   Slope = %.2f",
		data.Columns["intercept"].Data[0],
		data.Columns["slope"].Data[0])

	// Test Construct ConstructGeoms. This shouldn't change much as
	// GeomABLine doesn't reparametrize and we don't do position adjustments.
	plot.Layers[0].ConstructGeoms()
	plot.Layers[1].ConstructGeoms()

}
