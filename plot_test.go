package plot

import (
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
	mapping := AesMapping{X: "BMI"}
	bined := sb.Apply(df, mapping)
	bined.Print(os.Stdout)

	sb = StatBin{BinWidth: 5, Drop: false}
	mapping = AesMapping{X: "Age"}
	bined = sb.Apply(df, mapping)
	bined.Print(os.Stdout)

}
