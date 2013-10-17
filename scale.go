package plot

import (
	"fmt"
	"image/color"
	"math"
	"time"
)

// Scale provides position scales like x- and y-axis as well as color
// or other scales.
type Scale struct {
	Discrete    bool
	Time        bool
	Aesthetic   string // pos (x/y), col/fill, size, type ... TODO: good like this?
	ExpandToTic bool

	// Range of the Domain, as [DomainMin,DomainMax] interval for
	// continuous scales or as a set DomainLevels of values.
	// These values are populated during the trainings.
	DomainMin    float64
	DomainMax    float64
	DomainLevels FloatSet

	// The Fix... fields can be used to manually set the domain
	// of this scale to fixed values.
	FixMin    float64
	FixMax    float64
	FixLevels FloatSet

	// Transformation of values and guids.
	Transform *ScaleTransform

	// All following fields are set in Prepare.

	// Breaks controls the position of the tics. Empty: auto
	Breaks []float64

	// Labels are the labels for the tics. Empty: print Breaks
	Labels []string

	// Set up later after real training
	Color func(x float64) color.Color // color, fill. Any color
	Pos   func(x float64) float64     // x, y, size, alpha. In [0,1]
	Style func(x float64) int         // point and line type. Range ???
}

func (s *Scale) String() string {
	f2t := func(x float64) string {
		return time.Unix(int64(x), 0).Format("2006-01-02 15:04:05")
	}

	t := fmt.Sprintf("Scale for %q: ", s.Aesthetic)
	if s.Discrete {
		t += "discrete\n    Domain = "
		t += s.FixLevels.String()
	} else {
		if s.Time {
			t += "time\n    Domain = "
			t += f2t(s.DomainMin) + " -- " + f2t(s.DomainMax)
		} else {
			t += "continous\n    Domain = "
			t += fmt.Sprintf("%.2f -- %.2f", s.DomainMin, s.DomainMax)
		}
	}
	t += "\n    Transform = " + s.Transform.Name
	t += "\n    Breaks: "
	if len(s.Breaks) == 0 {
		t += "- empty -"
	} else {
		for _, b := range s.Breaks {
			t += fmt.Sprintf("%8.1f", b)
		}
		t += "\n    Labels: "
		if len(s.Labels) == 0 {
			t += "- empty -"
		} else {
			for _, l := range s.Labels {
				if len(l) >= 8 {
					l = l[:7]
					t += fmt.Sprintf("%-8s", l)
				}
			}
		}
	}

	if s.Pos == nil && s.Color == nil && s.Style == nil {
		t += "\n    not prepared"
	} else {
		t += "\n    prepared"
	}
	return t
}

// NewScale sets up a new scale for the given aesthetic, suitable for
// the given data in field.
func NewScale(aesthetic string, field Field) *Scale {
	scale := Scale{}
	scale.Discrete = field.Discrete()
	if field.Type == Time {
		scale.Time = true
	}
	scale.Aesthetic = aesthetic
	scale.DomainMin = math.Inf(+1)
	scale.DomainMax = math.Inf(-1)
	scale.DomainLevels = NewFloatSet()

	scale.Transform = &IdentityScale

	return &scale
}

// Train updates the domain ranges of s according to the data found in f.
func (s *Scale) Train(f Field) {
	println("Train ", s.Aesthetic)
	if f.Discrete() {
		// TODO: this depends on using the same StrIdx.
		// Maybe there should be a single StrIdx per plot.
		// This would internalize string valuies properly.
		s.DomainLevels.Join(f.Levels())
	} else {
		// Continous data.
		min, max, mini, maxi := f.MinMax()
		println("Training continous scale ", s.Aesthetic, " with min =", min, "@", mini, " max =", max, "@", max)
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
		println("Scale ", s.Aesthetic, " domain now = ", s.DomainMin, " - ", s.DomainMax)
	}
}

func (s *Scale) Retrain(aes string, geom Geom, df *DataFrame) {
	println("Train ", s.Aesthetic)
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
		println("Scale ", s.Aesthetic, " domain now = ", s.DomainMin, " - ", s.DomainMax)
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
		s.Aesthetic, s.DomainMin, s.DomainMax, min, max)

	// Set up breaks and labels
	nb := 6
	s.Breaks = make([]float64, nb+1)
	s.Labels = make([]string, nb+1)
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
		s.Labels[i] = format(x, "")
		fmt.Printf("  level %d = %s\n", i, s.Labels[i])
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

// -------------------------------------------------------------------------
// Scale Transformations

type ScaleTransform struct {
	Name    string
	Trans   func(float64) float64
	Inverse func(float64) float64
	Format  func(float64, string) string
}

var IdentityScale = ScaleTransform{
	Name:    "Identity",
	Trans:   func(x float64) float64 { return x },
	Inverse: func(y float64) float64 { return y },
	Format:  func(y float64, s string) string { return s },
}

var Log10Scale = ScaleTransform{
	Name:    "Log10",
	Trans:   func(x float64) float64 { return math.Log10(x) },
	Inverse: func(y float64) float64 { return math.Pow(10, y) },
	Format:  func(y float64, s string) string { return fmt.Sprintf("10^{%s}", s) },
}

var InvScale = ScaleTransform{
	Name:    "1/x",
	Trans:   func(x float64) float64 { return 1 / x },
	Inverse: func(y float64) float64 { return 1 / y },
	Format:  func(y float64, s string) string { return fmt.Sprintf("1/{%s}", s) },
}

var SqrtScale = ScaleTransform{
	Name:    "Sqrt",
	Trans:   func(x float64) float64 { return math.Sqrt(x) },
	Inverse: func(y float64) float64 { return y * y },
	Format:  func(y float64, s string) string { return fmt.Sprintf("%.1f", y*y) },
}
