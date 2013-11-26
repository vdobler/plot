package plot

import (
	"code.google.com/p/plotinum/vg"
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
	fmt.Printf("    Training Scale %s/%q with %d %s\n",
		s.Aesthetic, s.Name, len(f.Data), f.Type.String())
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
		fmt.Printf("      data is discrete and has %d levels\n", len(f.Levels()))
	} else {
		// Continous data.
		// TODO: this might train a discrete scale...
		min, max, mini, maxi := f.MinMax()
		fmt.Printf("      data is continuous from %.2f to %.2f\n",
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
	fmt.Printf("      --> Domain [%.2f,%.2f] , %d levels\n",
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
	fmt.Printf("  Finalizing discrete scale %q %p\n    = %+v\n", s.Name, s, *s)
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
		// TODO: merge with code from continuous
		// TODO: 1. this uses the expanded range. useful?
		h := s.Pos(x) * (5.0 / 6.0) // rescale by 5/6 because hue is cyclic
		hi := int(h * 6)
		f := h*6 - float64(hi)
		s := 1.0 // TODO: make configurable?
		v := 0.8 // TODO: make configurable?
		p, q, t := v*(1-s), v*(1-s*f), v*(1-s*(1-f))
		vv, tt, pp, qq := uint8(v*255), uint8(t*255), uint8(p*255), uint8(q*255)
		switch hi {
		case 0, 6:
			return color.RGBA{vv, tt, pp, 0xff}
		case 1:
			return color.RGBA{qq, vv, pp, 0xff}
		case 2:
			return color.RGBA{pp, vv, tt, 0xff}
		case 3:
			return color.RGBA{pp, qq, vv, 0xff}
		case 4:
			return color.RGBA{tt, pp, vv, 0xff}
		case 5:
			return color.RGBA{vv, pp, qq, 0xff}
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
	fmt.Printf("  Finalizing continuos scale %q %p\n", s.Name, s)
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
		// TODO: 1. this uses the expanded range. useful?
		h := s.Pos(x) * (5.0 / 6.0)
		hi := int(h * 6)
		f := h*6 - float64(hi)
		s := 1.0 // TODO: make configurable?
		v := 0.8 // TODO: make configurable?
		p, q, t := v*(1-s), v*(1-s*f), v*(1-s*(1-f))
		vv, tt, pp, qq := uint8(v*255), uint8(t*255), uint8(p*255), uint8(q*255)
		switch hi {
		case 0, 6:
			return color.RGBA{vv, tt, pp, 0xff}
		case 1:
			return color.RGBA{qq, vv, pp, 0xff}
		case 2:
			return color.RGBA{pp, vv, tt, 0xff}
		case 3:
			return color.RGBA{pp, qq, vv, 0xff}
		case 4:
			return color.RGBA{tt, pp, vv, 0xff}
		case 5:
			return color.RGBA{vv, pp, qq, 0xff}
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

	fmt.Printf("    PrepareContinousBreaks(%.2f, %.2f, %d) delta=%.3f mag=%.3f f=%.3f step=%.3f x0=%.3f n=%d\n", min, max, num, delta, mag, f, step, math.Ceil(min/step)*step, len(s.Breaks))
}

// PrepareLabels sets up s.Labels (if empty) by formating s.Breaks.
func (s *Scale) PrepareLabels() {
	fmt.Printf("    PrepareLabels from %d breaks\n", len(s.Breaks))
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

// -------------------------------------------------------------------------
// Rendering of scales

func (s *Scale) Render() (grobs Grob, width vg.Length, height vg.Length) {
	if !s.Discrete && (s.Aesthetic == "color" || s.Aesthetic == "fill") {
		return s.renderColorContinuous()
	}
	return s.renderDiscrete()
}

// renderOther renders all non-color scales.
// TODO: combine with renderColorDiscrete
func (s *Scale) renderDiscrete() (g Grob, width vg.Length, height vg.Length) {
	size := float64(vg.Millimeters(6))
	dx := float64(vg.Millimeters(2))
	dy := float64(vg.Millimeters(2))

	grobs := []Grob{}
	bgCol := BuiltinColors["gray80"]

	y := 0.0
	for i, v := range s.Breaks {
		// Gray background and label.
		rect := GrobRect{
			xmin: 0, xmax: size,
			ymin: y, ymax: y + size,
			fill: bgCol,
		}
		label := GrobText{
			x:     size + dx,
			y:     y + size/2,
			text:  s.Labels[i],
			color: BuiltinColors["black"],
			vjust: 0.5, hjust: 0,
		}
		lw, _ := label.BoundingBox()
		if width < lw {
			width = lw
		}

		// The actual key. TODO: think abut it at least a tiny bit, please!
		// TODO: the non-mapped aestetics should be settable by the user/geom/theme/whatever.
		var key Grob
		switch s.Aesthetic {
		case "size":
			key = GrobPoint{
				x: size / 2, y: y + size/2,
				size:  1 + 9*s.Pos(v), // must match values in GeomPoint!
				shape: SolidCirclePoint,
				color: BuiltinColors["blue"],
			}
		case "shape":
			key = GrobPoint{
				x: size / 2, y: y + size/2,
				size:  5,
				shape: PointShape(s.Style(v)),
				color: BuiltinColors["blue"],
			}
		case "linetype":
			key = GrobLine{
				x0: 0, y0: y + size/2, x1: size, y1: y + size/2,
				size:     1.5,
				linetype: LineType(s.Style(v)),
				color:    BuiltinColors["blue"],
			}
		case "color", "fill":
			key = GrobPoint{
				x: size / 2, y: y + size/2,
				size:  6,
				shape: SolidCirclePoint,
				color: s.Color(v),
			}
		}

		y += size + dy
		grobs = append(grobs, rect)
		grobs = append(grobs, key)
		grobs = append(grobs, label)
	}

	title := GrobText{
		x: 0, y: y,
		text:  s.Name,
		color: BuiltinColors["black"],
		vjust: 0, hjust: 0,
	}
	tw, th := title.BoundingBox()
	if width < tw {
		width = tw
	}
	grobs = append(grobs, title)

	width += vg.Length(size + dx)
	height = vg.Length(y) + th

	return GrobGroup{elements: grobs}, width, height
}

// renders a continuous color scale
func (s *Scale) renderColorContinuous() (g Grob, width vg.Length, height vg.Length) {
	sizeX := float64(vg.Millimeters(6))
	sizeY := float64(vg.Millimeters(50))
	sep := float64(vg.Millimeters(2))
	tic := float64(vg.Millimeters(1.5))

	grobs := []Grob{}

	// The color gradient.
	y := 0.0
	dy := sizeY / 50
	v := s.Min
	dv := (s.Max - s.Min) / 50
	overlap := 0.4
	for y < sizeY {
		col := s.Color(v)
		grobs = append(grobs, GrobRect{
			xmin: 0, xmax: sizeX,
			ymin: y - overlap, ymax: y + dy + overlap,
			fill: col,
		})

		y += dy
		v += dv
	}

	// Levels and tics.
	for i, v := range s.Breaks {
		txt := s.Labels[i]
		y = s.Pos(v) * sizeY
		grobs = append(grobs, GrobLine{
			x0: 0, x1: tic,
			y0: y, y1: y,
			size:     1,
			linetype: SolidLine,
			color:    BuiltinColors["white"],
		})
		grobs = append(grobs, GrobLine{
			x0: sizeX - tic, x1: sizeX,
			y0: y, y1: y,
			size:     1,
			linetype: SolidLine,
			color:    BuiltinColors["white"],
		})
		if txt != "" {
			label := GrobText{
				x:     sizeX + sep,
				y:     y,
				text:  txt,
				color: BuiltinColors["black"],
				size:  12,         // TODO: make configurable
				vjust: 0.5, hjust: 0,
			}
			lw, _ := label.BoundingBox()
			if width < lw {
				width = lw
			}
			grobs = append(grobs, label)
		}
	}

	// Title
	title := GrobText{
		x: 0, y: sizeY + sep,
		text:  s.Name,
		size:  12, // TODO: make configurable
		color: BuiltinColors["black"],
		vjust: 0, hjust: 0,
	}
	grobs = append(grobs, title)
	tw, th := title.BoundingBox()
	if width < tw {
		width = tw
	}

	width += vg.Length(sizeX + sep)
	height = vg.Length(sizeY+sep) + th

	return GrobGroup{elements: grobs}, width, height
}
