package plot

import (
	"fmt"
	"image/color"
	"math"
)

// Scale provides position scales like x- and y-axis as well as color
// or other scales.
type Scale struct {
	Discrete    bool
	Type        string // pos (x/y), col/fill, size, type ... TODO: good like this?
	ExpandToTic bool

	DomainMin    float64
	DomainMax    float64
	DomainLevels FloatSet

	Transform *ScaleTransform

	// Also set up after training.
	Breaks []float64 // empty: auto
	Levels []string  // empty: auto, different length than breaks: bug

	// Set up later after real training
	Color func(x float64) color.Color // color, fill. Any color
	Pos   func(x float64) float64     // x, y, size, alpha. In [0,1]
	Style func(x float64) int         // point and line type. Range ???
}

// NewScale sets up a new scale for the given aesthetic, suitable for
// the given data in field.
func NewScale(aesthetic string, field Field) *Scale {
	scale := Scale{}
	scale.Discrete = field.Discrete()
	scale.Type = aesthetic
	scale.DomainMin = math.Inf(+1)
	scale.DomainMax = math.Inf(-1)
	scale.DomainLevels = NewFloatSet()

	return &scale
}

// Train updates the domain ranges of s according to the data found in f.
func (s *Scale) Train(f Field) {
	println("Train ", s.Type)
	if f.Discrete() {
		// TODO: this depends on using the same StrIdx.
		// Maybe there should be a single StrIdx per plot.
		// This would internalize string valuies properly.
		s.DomainLevels.Join(f.Levels())
	} else {
		// Continous data.
		min, max, mini, maxi := f.MinMax()
		println("Training continous scale ", s.Type, " with min =", min, "@", mini, " max =", max, "@", max)
		if mini != -1 {
			if min < s.DomainMin {
				s.DomainMin = min
			}
		}
		if maxi != -1 {
			if max > s.DomainMax {
				s.DomainMax = max
			}
		}
		println("Scale ", s.Type, " domain now = ", s.DomainMin, " - ", s.DomainMax)
	}
}

func (s *Scale) Retrain(aes string, geom Geom, df *DataFrame) {
	println("Train ", s.Type)
	if s.Discrete {
		// TODO: this depends on using the same StrIdx.
		// Maybe there should be a single StrIdx per plot.
		// This would internalize string valuies properly.
		panic("Implement me")
	} else {
		// Continous data.
		min, max := geom.Bounds(aes, df)
		if !math.IsNaN(min) {
			if min < s.DomainMin {
				s.DomainMin = min
			}
		}
		if !math.IsNaN(max) {
			if max > s.DomainMax {
				s.DomainMax = max
			}
		}
		println("Scale ", s.Type, " domain now = ", s.DomainMin, " - ", s.DomainMax)
	}
}

// Prepare initialises the remaining fields after training.
func (s *Scale) Prepare() {
	if s.Discrete {
		s.PrepareDiscrete()
	} else {
		s.PrepareContinous()
	}
}

func (s *Scale) PrepareDiscrete() {
	fmt.Printf("Scale %#v\n", *s)
	panic("Implement me")
}

// TODO: Scale needs access to data frame field to print string values
func (s *Scale) PrepareContinous() {
	fullRange := s.DomainMax - s.DomainMin
	expand := fullRange * 0.05
	min, max := s.DomainMin-expand, s.DomainMax+expand
	fullRange = max - min

	fmt.Printf("Scale %s, cont. domain=[%.3f,%.3f] expanded=[%.3f,%.3f]\n",
		s.Type, s.DomainMin, s.DomainMax, min, max)

	// Set up breaks and labels
	nb := 6
	s.Breaks = make([]float64, nb+1)
	s.Levels = make([]string, nb+1)
	for i := range s.Breaks {
		x := s.DomainMin + float64(i)*fullRange/float64(nb)
		s.Breaks[i] = x
		fmt.Printf("  break %d = %.3f\n", i, x)
	}
	// TODO: ugly should be initialised to identitiy transform
	var format func(float64, string) string
	if t := s.Transform; t != nil {
		format = t.Format
	} else {
		format = func(x float64, s string) string {
			return fmt.Sprintf("%g", x)
		}
	}
	for i, x := range s.Breaks {
		s.Levels[i] = format(x, "")
		fmt.Printf("  level %d = %s\n", i, s.Levels[i])
	}

	// Produce mapping functions
	s.Pos = func(x float64) float64 {
		return (x - min) / fullRange
	}
	s.Color = func(x float64) color.Color {
		c := s.Pos(x)
		// TODO (a lot)
		if c < 1/3 {
			r := uint8(c * 3 * 255)
			return color.RGBA{r, 0xff - r, 0, 0xff}
		} else if c < 2/3 {
			r := uint8((c - 1/3) * 3 * 255)
			return color.RGBA{0, r, 0xff - r, 0xff}
		} else {
			r := uint8((c - 2/3) * 3 * 255)
			return color.RGBA{0xff - r, 0, r, 0xff}
		}
		return color.RGBA{}
	}
	s.Style = func(x float64) int {
		c := s.Pos(x)
		c *= float64(StarPoint) // TODO
		return int(c)
	}
}
