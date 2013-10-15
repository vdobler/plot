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

func TestHex2Color(t *testing.T) {
	tests := []struct {
		s string
		c color.Color
	}{
		{"#1256ab", color.RGBA{0x12, 0x56, 0xab, 0xff}},
		{"#1256abcd", color.RGBA{0x12, 0x56, 0xab, 0xcd}},
		{"red", color.RGBA{0xff, 0x00, 0x00, 0xff}},
		{"green", color.RGBA{0x00, 0xff, 0x00, 0xff}},
		{"blue", color.RGBA{0x00, 0x00, 0xff, 0xff}},
		{"nonsens", color.RGBA{0xaa, 0x66, 0x77, 0x7f}},
	}

	for i, tc := range tests {
		got := Hex2Color(tc.s)
		rg, gg, bg, ag := got.RGBA()
		rw, gw, bw, aw := tc.c.RGBA()
		if rg != rw || gg != gw || bg != bw || ag != aw {
			t.Errorf("%d %q: got %04X, %04X, %04X, %04X want %04X, %04X, %04X, %04X",
				i, tc.s, rw, gw, bw, aw, rg, gg, bg, ag)
		}
	}
}
