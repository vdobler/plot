package plot

import (
	"os"
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

func (o Ops) Country() string {
	o2c := map[string]string{
		"ch": "Schweiz",
		"de": "Deutschland",
		"uk": "England",
	}
	return o2c[o.Origin]
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
	df, err := NewDataFrameFrom(measurement)
	if err != nil {
		t.Fatalf("Unexpected error %s", err)
	}

	if df.N != 20 {
		t.Errorf("Got %d elements, want 20", df.N)
	}

	if len(df.Type) != 7 || len(df.Data) != 7 {
		t.Errorf("Got %d, %d fields, want 7", len(df.Type), len(df.Data))
	}
}

func TestFilter(t *testing.T) {
	df, _ := NewDataFrameFrom(measurement)

	exactly20 := Filter(df, "Age", 20)
	if exactly20.N != 5 {
		t.Errorf("Got %d, want 5", exactly20.N)
	}
	for i, a := range exactly20.Data["Age"] {
		if a != 20 {
			t.Errorf("Element %d has age %v (want 20)", i, a)
		}
	}

	age30to39 := Filter(df, "Group", 35)
	if age30to39.N != 6 {
		t.Errorf("Got %d, want 6", age30to39.N)
	}
	for i, a := range age30to39.Data["Age"] {
		if a < 30 || a > 39 {
			t.Errorf("Element %d has age %v (want 20)", i, a)
		}
	}

	ukOnly := Filter(df, "Origin", "uk")
	if ukOnly.N != 4 {
		t.Errorf("Got %d, want 4", ukOnly.N)
	}
	ukIdx := float64(ukOnly.Type["Origin"].StrIdx("uk"))
	for i, o := range ukOnly.Data["Origin"] {
		if o != ukIdx {
			t.Errorf("Element %d has origin %v (want uk)", i, o)
		}
	}

	/*
		deOnly := Filter(df, "Country", "Deutschland")
		if deOnly.N != 8 {
			t.Errorf("Got %d, want 8", deOnly.N)
		}
		for i, o := range deOnly.Data["Origin"] {
			if o.(string) != "de" {
				t.Errorf("Element %d has origin %v (want de)", i, o)
			}
		}
	*/
}

func TestLevels(t *testing.T) {
	df, _ := NewDataFrameFrom(measurement)
	ageLevels := Levels(df, "Age")
	if len(ageLevels) != 10 || ageLevels[0].(float64) != 20 || ageLevels[9].(float64) != 47 */{
		t.Errorf("Got %v", ageLevels)
	}

	origLevels := Levels(df, "Origin")
	if len(origLevels) != 3 || origLevels[0].(string) != "ch" || origLevels[1].(string) != "de" || origLevels[2].(string) != "uk" {
		t.Errorf("Got %#v", origLevels)
	}
}

func TestMinMax(t *testing.T) {
	df, _ := NewDataFrameFrom(measurement)

	min, max, a, b := MinMax(df, "Weight")
	if min != 55 || a != 18 {
		t.Errorf("Min: Got %f/%d, want 55.00/18", min, a)
	}
	if max != 99.0 || b != 10 {
		t.Errorf("Min: Got %f/%d, want 99.00/10", max, b)
	}
}

func TestPrint(t *testing.T) {
	df, _ := NewDataFrameFrom(measurement)
	df.Print(os.Stdout)
}
