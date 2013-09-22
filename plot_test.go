package plot

import (
	"testing"
)

func TestFaceting(t *testing.T) {
	df, _ := NewDataFrame(measurement)

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
