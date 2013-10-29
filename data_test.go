package plot

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"testing"
)

var _ = fmt.Printf
var _ = math.Floor

type Obs struct {
	Age     int
	Origin  string
	Weight  float64
	Height  float64
	Special []byte
}

func (o Obs) BMI() float64 {
	return o.Weight / (o.Height * o.Height)
}

func (o Obs) Group() int {
	return 10*(o.Age/10) + 5
}

func (o Obs) Country() string {
	o2c := map[string]string{
		"ch": "Schweiz",
		"de": "Deutschland",
		"uk": "England",
	}
	return o2c[o.Origin]
}

func (o Obs) Other() bool {
	return true
}

func (o Obs) Other2(a int) int {
	return 0
}

var measurement = []Obs{
	Obs{Age: 20, Origin: "de", Weight: 80, Height: 1.88},
	Obs{Age: 22, Origin: "de", Weight: 85, Height: 1.85},
	Obs{Age: 20, Origin: "de", Weight: 90, Height: 1.95},
	Obs{Age: 25, Origin: "de", Weight: 90, Height: 1.72},

	Obs{Age: 20, Origin: "ch", Weight: 77, Height: 1.78},
	Obs{Age: 20, Origin: "ch", Weight: 82, Height: 1.75},
	Obs{Age: 28, Origin: "ch", Weight: 85, Height: 1.80},
	Obs{Age: 20, Origin: "ch", Weight: 84, Height: 1.62},

	Obs{Age: 31, Origin: "de", Weight: 85, Height: 1.88},
	Obs{Age: 30, Origin: "de", Weight: 90, Height: 1.85},
	Obs{Age: 30, Origin: "de", Weight: 99, Height: 1.95},
	Obs{Age: 42, Origin: "de", Weight: 95, Height: 1.72},

	Obs{Age: 30, Origin: "ch", Weight: 80, Height: 1.78},
	Obs{Age: 30, Origin: "ch", Weight: 85, Height: 1.75},
	Obs{Age: 37, Origin: "ch", Weight: 87, Height: 1.80},
	Obs{Age: 47, Origin: "ch", Weight: 90, Height: 1.62},

	Obs{Age: 42, Origin: "uk", Weight: 60, Height: 1.68},
	Obs{Age: 42, Origin: "uk", Weight: 65, Height: 1.65},
	Obs{Age: 44, Origin: "uk", Weight: 55, Height: 1.52},
	Obs{Age: 44, Origin: "uk", Weight: 70, Height: 1.72},
}

func TestNewDataFrame(t *testing.T) {
	df, err := NewDataFrameFrom(measurement)
	if err != nil {
		t.Fatalf("Unexpected error %s", err)
	}

	if df.N != 20 {
		t.Errorf("Got %d elements, want 20", df.N)
	}

	if len(df.Columns) != 7 {
		t.Errorf("Got %d fields, want 7", len(df.Columns))
	}
}

func TestFilter(t *testing.T) {
	df, _ := NewDataFrameFrom(measurement)

	exactly20 := Filter(df, "Age", 20)
	if exactly20.N != 5 {
		t.Errorf("Got %d (%d), want 5", exactly20.N, len(exactly20.Columns["Age"].Data))
	}
	for i, a := range exactly20.Columns["Age"].Data {
		if a != 20 {
			t.Errorf("Element %d has age %v (want 20)", i, a)
		}
	}

	age30to39 := Filter(df, "Group", 35)
	if age30to39.N != 6 {
		t.Errorf("Got %d, want 6", age30to39.N)
	}
	for i, a := range age30to39.Columns["Age"].Data {
		if a < 30 || a > 39 {
			t.Errorf("Element %d has age %v (want 20)", i, a)
		}
	}

	ukOnly := Filter(df, "Origin", "uk")
	if ukOnly.N != 4 {
		t.Errorf("Got %d, want 4", ukOnly.N)
	}
	originField := ukOnly.Columns["Origin"]
	fmt.Printf("Str: %v\n", originField.Str)
	ukIdx := originField.StrIdx("uk")
	for i, o := range ukOnly.Columns["Origin"].Data {
		if int(o) != ukIdx || originField.String(o) != "uk" {
			t.Errorf("Element %d: Got %.1f %q, want %d \"uk\"", i, o, originField.String(o), ukIdx)
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
	ageLevels := Levels(df, "Age").Elements()
	if len(ageLevels) != 10 || ageLevels[0] != 20 || ageLevels[9] != 47 {
		t.Errorf("Got %v", ageLevels)
	}

	origLevels := Levels(df, "Origin").Elements()
	if len(origLevels) != 3 || origLevels[0] != 0 || origLevels[1] != 1 || origLevels[2] != 2 {
		t.Errorf("Got %#v want [0 1 2]", origLevels)
	}

	origStr := df.Columns["Origin"].Strings(origLevels)
	sort.Strings(origStr)
	if origStr[0] != "ch" || origStr[1] != "de" || origStr[2] != "uk" {
		t.Errorf("Got %v, want [ch, de, uk]", origStr)
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

type Diamond struct {
	Carat               float32
	Cut, Color, Clarity string
	Depth, Table        float32
	Price               int
	X, Y, Z             float32
}

func (d Diamond) RClarity() string {
	c := d.Clarity
	n := len(c) - 1
	if c[n] == '1' || c[n] == '2' {
		return c[:n]
	}
	return c
}

func ReadDiamonds(fname string) (diamonds []Diamond, err error) {
	file, err := os.Open(fname)
	if err != nil {
		return
	}
	defer file.Close()

	// Helper to convert srtrings to float and int.
	s2f := func(s string) float32 {
		value, err := strconv.ParseFloat(s, 32)
		if err != nil {
			value = math.NaN()
		}
		return float32(value)
	}
	s2i := func(s string) int {
		value, err := strconv.Atoi(s)
		if err != nil {
			value = -1
		}
		return value
	}

	csvReader := csv.NewReader(file)
	first := true
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return diamonds, err
		}
		if first {
			first = false
			continue
		}
		d := Diamond{
			Carat:   s2f(record[1]),
			Cut:     record[2],
			Color:   record[3],
			Clarity: record[4],
			Depth:   s2f(record[5]),
			Table:   s2f(record[6]),
			Price:   s2i(record[7]),
			X:       s2f(record[8]),
			Y:       s2f(record[9]),
			Z:       s2f(record[10]),
		}
		diamonds = append(diamonds, d)
	}

	return diamonds, nil
}
