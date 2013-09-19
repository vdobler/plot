package plot

import (
	"testing"
)


func TestFaceting(t *testing.T) {
	fac := Faceting{
		Columns: "Group",
		Rows: "Origin",
		Totals: true,
	}

	p := Plot{
		Data: measurement,
		Faceting: fac,
	}

	p.draw()
}
