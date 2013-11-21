package plot

import (
	"fmt"
	"image/color"
	"math"
	"sort"
	"time"
)

// Scale provides position scales like x- and y-axis as well as color
// or other scales.
type Scale struct {
	Name string // Name is used as the title in legends or as axis labels.

	DomainType FieldType
	Discrete   bool
	Time       bool

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
	// used for this scale.  All finalized scales are continous so
	// there is no Levels field.
	Min, Max float64

	// All following fields are set in Prepare.
	// These functions map the domain space to the aesthetics space.
	Color func(x float64) color.Color // color, fill. Any color
	Pos   func(x float64) float64     // x, y, size, alpha. In [0,1]
	Style func(x float64) int         // point and line type. BUG: Range

	Finalized bool
}

// NewScale sets up a new scale for the given aesthetic, suitable for
// the given data in field.
func NewScale(aesthetic string, name string, ft FieldType) *Scale {
	scale := Scale{
		Name:         name,
		Aesthetic:    aesthetic,
		DomainType:   ft,
		DomainMin:    math.Inf(+1),
		DomainMax:    math.Inf(-1),
		DomainLevels: NewFloatSet(),
		Transform:    &IdentityScale,
		ExpandRel:    0.05,
		ExpandAbs:    0.0,
	}

	switch ft {
	case Time:
		scale.Time = true
	case String:
		scale.Discrete = true
	}

	return &scale
}

// String pretty prints s.
func (s *Scale) String() string {
	f2t := func(x float64) string {
		return time.Unix(int64(x), 0).Format("2006-01-02 15:04:05")
	}

	t := fmt.Sprintf("Scale %q %p named %q: ", s.Aesthetic, s, s.Name)
	if s.Discrete {
		t += "discrete\n    Domain:    "
		t += s.DomainLevels.String()
	} else {
		if s.Time {
			t += "time\n    Domain:    "
			t += f2t(s.DomainMin) + " -- " + f2t(s.DomainMax)
		} else {
			t += "continous\n    Domain:    "
			t += fmt.Sprintf("%.2f -- %.2f", s.DomainMin, s.DomainMax)
		}
	}
	t += "\n    Transform: " + s.Transform.Name
	t += "\n    Breaks:    "
	if len(s.Breaks) == 0 {
		t += "- empty -"
	} else {
		for _, b := range s.Breaks {
			t += fmt.Sprintf("%8.1f", b)
		}
		t += "\n    Labels:    "
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
		t += "\n    Status:    not prepared"
	} else {
		t += "\n    Status:    prepared"
	}
	return t
}

// -------------------------------------------------------------------------
// Training

// Train updates the domain ranges of s according to the data found in f.
func (s *Scale) Train(f Field) {
	fmt.Printf("Training Scale %s/%s with %d %s\n",
		s.Name, s.Aesthetic, len(f.Data), f.Type.String())
	if f.Discrete() {
		s.DomainLevels.Join(f.Levels())
		levels := s.DomainLevels.Elements()
		if n := len(levels); n > 0 {
			if levels[0] < s.DomainMin {
				s.DomainMin = levels[0]
			}
			if levels[n-1] > s.DomainMax {
				s.DomainMax = levels[n-1]
			}
		}
		fmt.Printf("  data is discrete and has %d levels\n", len(f.Levels()))
	} else {
		// Continous data.
		// TODO: this might train a discrete scale...
		min, max, mini, maxi := f.MinMax()
		fmt.Printf("  data is continuous from %.2f to %.2f\n",
			min, max)
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
	fmt.Printf("  --> Domain [%.2f,%.2f] , %d levels\n",
		s.DomainMin, s.DomainMax, len(s.DomainLevels))
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
func (s *Scale) Finalize(pool *StringPool) {
	if s.Finalized {
		return
	}

	if s.Discrete {
		s.FinalizeDiscrete(pool)
	} else {
		s.FinalizeContinous()
	}

	s.Finalized = true
}

// Convert the discrete x value with possible adjustemnts in (-0.5,+0.5)
// to a continous value by looking up xi....   Arghh...
func discreteToCont(x float64, levels []float64) float64 {
	xi := math.Floor(x + 0.5)
	dx := x - xi
	i := -1
	for j, v := range levels {
		if v == xi {
			i = j
			break
		}
	}
	w := float64(i+1) + dx
	return w
}

// FinalizeDiscrete
func (s *Scale) FinalizeDiscrete(pool *StringPool) {
	fmt.Printf("Finalizing discrete scale %q %p\n%+v\n", s.Name, s, *s)
	// TODO: Manual setting the values.

	// Position the n levels on 1, 2, ..., n but consider the
	// posibility that the geom might be broad and require extra space.
	n := len(s.DomainLevels)
	levels := s.DomainLevels.Elements()
	s.Min, s.Max = 1, float64(n)
	// This works only because the levels are sorted.
	if x := discreteToCont(s.DomainMin, levels); x < s.Min {
		s.Min = x
	}
	if x := discreteToCont(s.DomainMax, levels); x > s.Max {
		s.Max = x
	}

	expand := (s.Max-s.Min)*s.ExpandRel + s.ExpandAbs
	if expand < 0.1 {
		expand = 0.1
	}
	s.Min -= expand
	s.Max += expand
	fullRange := s.Max - s.Min

	// Breaks are put on integer values [1,n].
	s.Breaks = make([]float64, n)
	for i := range s.Breaks {
		s.Breaks[i] = float64(i + 1)
	}

	// Ordering and labels of the discrete levels.
	s.Breaks = make([]float64, n)
	s.Labels = make([]string, n)
	sort.Float64s(levels)
	for i := range s.Breaks {
		s.Breaks[i] = levels[i]
	}
	switch s.DomainType {
	case String:
		for i := range s.Labels {
			s.Labels[i] = pool.Get(int(levels[i]))
		}
	case Int:
		for i := range s.Labels {
			s.Labels[i] = fmt.Sprintf("%d", int(levels[i]))
		}
	default:
		panic(fmt.Sprintf("Bad domain type %s for discrete scale %s (%s)",
			s.DomainType.String(), s.Aesthetic, s.Name))
	}

	// Produce mapping functions
	s.Pos = func(x float64) float64 {
		xi := math.Floor(x + 0.5)
		dx := x - xi
		i := -1
		for j, v := range levels {
			if v == xi {
				i = j
				break
			}
		}
		w := float64(i+1) + dx
		// Scale to [0,1]
		z := (w - s.Min) / fullRange
		// fmt.Printf("s.Pos(%.2f) xi=%.0f dx=%.2f i=%d w=%.1f --> %.2f  (s.Min=%.1f s.Max=%.1f)\n",
		//	x, xi, dx, i, w, z, s.Min, s.Max)

		return z
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
		c *= float64(StarPoint) // TODO same as below
		return int(c)
	}
}

// FinalizeContinous sets up the fields Breaks, Labels and the
// functions from Domain to [0,1] (x,y,time, etc), color or int.
func (s *Scale) FinalizeContinous() {
	fmt.Printf("Finalizing continuos scale %q %p\n", s.Name, s)
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

	x := math.Ceil(min/step) * step
	for x < s.DomainMax {
		s.Breaks = append(s.Breaks, x)
		x += step
	}

	fmt.Printf("PrepareContinousBreaks(%.2f, %.2f, %d)\n  delta = %.3f  mag  = %.3f   f    = %.3f\n  step = %.3f  x0  =  %.3f   n=%d\n", min, max, num, delta, mag, f, step, math.Ceil(min/step)*step, len(s.Breaks))
}

// PrepareLabels sets up s.Labels (if empty) by formating s.Breaks.
func (s *Scale) PrepareLabels() {
	println("PrepareLabels with ", len(s.Breaks))
	if len(s.Breaks) == 0 {
		return
	}
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
	f := "%.1f"
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
