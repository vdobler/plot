package plot

import (
	"testing"
	// "math"
)

type Ops struct {
	Age     int
	Origin  string
	Weight  float64
	Height  float64
	Special []byte
}

func (o Ops) BMI() float64 {
	return o.Weight / (o.Height * o.Height)
}

func (o Ops) Group() int {
	return 10*(o.Age/10) + 5
}

func (o Ops) Other() bool {
	return true
}

func (o Ops) Other2(a int) int {
	return 0
}

var measurement = []Ops{
	Ops{Age: 20, Origin: "de", Weight: 80, Height: 1.88},
	Ops{Age: 22, Origin: "de", Weight: 85, Height: 1.85},
	Ops{Age: 20, Origin: "de", Weight: 90, Height: 1.95},
	Ops{Age: 25, Origin: "de", Weight: 90, Height: 1.72},

	Ops{Age: 20, Origin: "ch", Weight: 77, Height: 1.78},
	Ops{Age: 20, Origin: "ch", Weight: 82, Height: 1.75},
	Ops{Age: 28, Origin: "ch", Weight: 85, Height: 1.80},
	Ops{Age: 20, Origin: "ch", Weight: 84, Height: 1.62},

	Ops{Age: 31, Origin: "de", Weight: 85, Height: 1.88},
	Ops{Age: 30, Origin: "de", Weight: 90, Height: 1.85},
	Ops{Age: 30, Origin: "de", Weight: 99, Height: 1.95},
	Ops{Age: 42, Origin: "de", Weight: 95, Height: 1.72},

	Ops{Age: 30, Origin: "ch", Weight: 80, Height: 1.78},
	Ops{Age: 30, Origin: "ch", Weight: 85, Height: 1.75},
	Ops{Age: 37, Origin: "ch", Weight: 87, Height: 1.80},
	Ops{Age: 47, Origin: "ch", Weight: 90, Height: 1.62},

	Ops{Age: 42, Origin: "uk", Weight: 60, Height: 1.68},
	Ops{Age: 42, Origin: "uk", Weight: 65, Height: 1.65},
	Ops{Age: 44, Origin: "uk", Weight: 55, Height: 1.52},
	Ops{Age: 44, Origin: "uk", Weight: 70, Height: 1.72},
}

func TestNewDataFrame(t *testing.T) {
	df, err := NewDataFrame(measurement)
	if err != nil {
		t.Fatalf("Unexpected error %s", err)
	}

	if df.N != 20 {
		t.Errorf("Got %d elements, want 20", df.N)
	}

	if len(df.Fields) != 6 {
		t.Errorf("Got %d fields, want 6", len(df.Fields))
	}
}

func TestFilter(t *testing.T) {
	df, _ := NewDataFrame(measurement)
	exactly20 := df.Filter("Age", 20)
	if exactly20.N != 5 {
		t.Errorf("Got %d, want 5", exactly20.N)
	}

	age30to39 := df.Filter("Group", 35)
	if age30to39.N != 6 {
		t.Errorf("Got %d, want 6", age30to39.N)
	}

	deOnly := df.Filter("Origin", "uk")
	if deOnly.N != 4 {
		t.Errorf("Got %d, want 4", deOnly.N)
	}
}

/*
func TestMinMax(t *testing.T) {
	amin, amax, auniq := MinMax(measurement, "Age")
	if amin.(int64) != 20 {
		t.Errorf("Got %d, want 20", amin.(int64))
	}
	if amax.(int64) != 47 {
		t.Errorf("Got %d, want 47", amax.(int64))
	}
	if len(auniq) != 10 {
		t.Errorf("Got %d, want 10", len(auniq))
	}


	bmin, bmax, buniq := MinMax(measurement, "BMI")
	if math.Abs(bmin.(float64)-21.2585) > 0.01  {
		t.Errorf("Got %f, want 21.2585", bmin.(float64))
	}
	if math.Abs(bmax.(float64) - 34.29355) > 0.01 {
		t.Errorf("Got %f, want 34.29355", bmax.(float64))
	}
	if len(buniq) != 0 {
		t.Errorf("Got %f, want 0", len(buniq))
	}


	omin, omax, ouniq := MinMax(measurement, "Origin")
	if omin.(string) != "ch"  {
		t.Errorf("Got %s, want ch", omin.(string))
	}
	if omax.(string) != "uk" {
		t.Errorf("Got %s, want uk", omax.(string))
	}
	if len(ouniq) != 3 || ouniq[1].(string) != "de" {
		t.Errorf("Got %v, want [ch de uk]", ouniq)
	}

}

*/
