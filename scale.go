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
	Name string // Name is used as the title in legends or as axis labels.

	Discrete bool
	Time     bool

	// pos (x/y), col/fill, size, type ... TODO: good like this?
	Aesthetic string // should be same like the map key in Plot.Scales

	// Transformation of values and guides.
	Transform *ScaleTransform

	// The Fix... fields can be used to manually set the domain
	// of this scale to fixed values. FixLevels is used for discrete
	// scales and FixMin/Max are used for continuous scales.
	// A empty FixLevels or FixMin==FixMax results in an automatical
	// determination of the domain of this scale based on the data
	// plotted. Otherwise if FixMin!=FixMax the given values are
	// used.
	FixMin    float64
	FixMax    float64
	FixLevels FloatSet

	// Relative and absolute expansion of scale.
	ExpandRel, ExpandAbs float64

	// Breaks controls the position of the tics. Empty: auto
	Breaks []float64

	// Labels are the labels for the tics. Empty: print Breaks
	Labels []string

	// Empirical range of the Domain, as [DomainMin,DomainMax] interval
	// for  continuous scales or as a set DomainLevels of values.
	// These values are populated during the trainings.
	DomainMin    float64
	DomainMax    float64
	DomainLevels FloatSet

	// The actual min and max (continuous scales) or levels (discrete)
	// used for this scale.
	Min, Max float64
	Levels   FloatSet

	// All following fields are set in Prepare.
	Color func(x float64) color.Color // color, fill. Any color
	Pos   func(x float64) float64     // x, y, size, alpha. In [0,1]
	Style func(x float64) int         // point and line type. Range ???

	Finalized bool
}

// NewScale sets up a new scale for the given aesthetic, suitable for
// the given data in field.
func NewScale(aesthetic string, name string, ft FieldType) *Scale {
	scale := Scale{}
	switch ft {
	case Time:
		scale.Time = true
	case String:
		scale.Discrete = true
	}
	scale.Aesthetic = aesthetic
	scale.DomainMin = math.Inf(+1)
	scale.DomainMax = math.Inf(-1)
	scale.DomainLevels = NewFloatSet()

	scale.Transform = &IdentityScale

	scale.ExpandRel = 0.05
	scale.ExpandAbs = 0.0

	return &scale
}

// String pretty prints s.
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
				}
				t += fmt.Sprintf("%8s", l)
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

// -------------------------------------------------------------------------
// Training

// Train updates the domain ranges of s according to the data found in f.
func (s *Scale) Train(f Field) {
	if f.Discrete() {
		// TODO: this depends on using the same StrIdx.
		// Maybe there should be a single StrIdx per plot.
		// This would internalize string valuies properly.
		s.DomainLevels.Join(f.Levels())
	} else {
		// Continous data.
		min, max, mini, maxi := f.MinMax()
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
	}
}

func (s *Scale) TrainByValue(xs ...float64) {
	if s.Discrete {
		panic("Implement me")
	} else {
		for _, x := range xs {
			if math.IsNaN(x) {
				continue
			}
			if x < s.DomainMin {
				s.DomainMin = x
			} else if x > s.DomainMax {
				s.DomainMax = x
			}

		}
	}
}

// -------------------------------------------------------------------------
// Preparing a scale

// Prepare initialises the remaining fields after training.
func (s *Scale) Finalize() {
	if s.Finalized {
		return
	}

	if s.Discrete {
		s.FinalizeDiscrete()
	} else {
		s.FinalizeContinous()
	}

	s.Finalized = true
}

func (s *Scale) FinalizeDiscrete() {
	fmt.Printf("Scale %#v\n", *s)
	panic("Implement me")
}

// TODO: Scale needs access to data frame field to print string values
func (s *Scale) FinalizeContinous() {
	s.Min, s.Max = s.DomainMin, s.DomainMax
	if s.FixMin != s.FixMax {
		s.Min, s.Max = s.FixMin, s.FixMax
	}
	expand := (s.Max-s.Min)*s.ExpandRel + s.ExpandAbs
	s.Min -= expand
	s.Max += expand
	fullRange := s.Max - s.Min

	// Set up breaks and labels
	if len(s.Breaks) == 0 {
		// All auto.
		s.PrepareBreaks(s.Min, s.Max, 5)
	}
	s.PrepareLabels()

	// Produce mapping functions
	s.Pos = func(x float64) float64 {
		return (x - s.Min) / fullRange
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

// PrepareBreaks populates s.breaks with suitable values.
// Suitable values for a range of [55,125] are [60,80,100,120].
// TODO: For a log10 transformed scale the breaks should be
// plain integers:
//   Raw data [0.01,100] --log10--> [-2,2] --break--> [-2,-1,0,1,2]
// this should work, but what with
// raw [12,88] --log10--> [1.08,1.94] --break--> [1.2,1.4,1.6,1.8]
// which gives [15.8, 25.1, 39.8, 63.1] wich is ugly. More
// dramatic on sqrt or 1/x transforms.
func (s *Scale) PrepareBreaks(min, max float64, num int) {
	if s.Discrete || s.Time {
		panic("Shuld not happen")
	}

	if s.Time {
		s.PrepareTimeBreaks(min, max, num)
	} else {
		s.PrepareContinousBreaks(min, max, num)
	}
}

// TODO: Code below is suboptimal
func (s *Scale) PrepareTimeBreaks(min, max float64, num int) {
	s.Breaks = []float64{min, max, (min + max) / 2}
}

// PrepepareContinousBreaks automatically populates s.Breaks
// with suitable values.
func (s *Scale) PrepareContinousBreaks(min, max float64, num int) {
	fullRange := max - min

	// Decompose delta into the form delta = f * mag
	// with mag a power of 10 and 0 < f < 10.
	delta := fullRange / float64(num)
	mag := math.Pow10(int(math.Floor(math.Log10(delta))))
	f := delta / mag

	step := 0.0

	switch {
	case f < 1.8:
		step = 1
	case f < 3:
		step = 2.5
	case f < 4:
		step = 2
	case f < 9:
		step = 5
	default:
		step = 1
		mag *= 10
	}
	step *= mag

	x := math.Ceil(min / step)
	for x < s.DomainMax {
		s.Breaks = append(s.Breaks, x)
		x += step
	}

}

// PrepareLabels sets up s.Labels (if empty) by formating s.Breaks.
func (s *Scale) PrepareLabels() {
	if len(s.Labels) == 0 {
		// Automatic label creation.
		formatter := s.ChooseFloatFormatter()
		for _, b := range s.Breaks {
			s.Labels = append(s.Labels, formatter(b))
		}
	} else {
		// User provided labels. Sanitize them.
		nl, nb := len(s.Labels), len(s.Breaks)
		if nl > nb {
			s.Labels = s.Labels[:nb]
		} else if nl < nb {
			panic("Implement me")
		}
	}
}

// TODO: Much more logic needed
func (s *Scale) ChooseFloatFormatter() func(x float64) string {
	f := "%d"
	if math.Abs(s.Breaks[0]) < 1 || math.Abs(s.Breaks[len(s.Breaks)-1]) < 1 {
		f = "%.1f" // BUG
	}
	return func(x float64) string {
		return fmt.Sprintf(f, x)
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
