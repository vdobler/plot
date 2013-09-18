package plot

import (
	"testing"
)

type Ops struct {
	Age    int
	Origin string
	Weight float64
	Height float64
}

func (o Ops) BMI() float64 {
	return o.Weight / (o.Height * o.Height)
}

func (o Ops) Group() int {
	return o.Age / 10
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
	Ops{Age: 35, Origin: "de", Weight: 95, Height: 1.72},

	Ops{Age: 30, Origin: "ch", Weight: 80, Height: 1.78},
	Ops{Age: 30, Origin: "ch", Weight: 85, Height: 1.75},
	Ops{Age: 37, Origin: "ch", Weight: 87, Height: 1.80},
	Ops{Age: 30, Origin: "ch", Weight: 90, Height: 1.62},
}

func TestFilter(t *testing.T) {
	exactly20 := Filter(measurement, "Age", 20).([]Ops)
	if got := len(exactly20); got != 5 {
		t.Errorf("Got %d, want 5", got)
	}

	age30to39 := Filter(measurement, "Group", 3).([]Ops)
	if got := len(age30to39); got != 8 {
		t.Errorf("Got %d, want 8", got)
	}
}
