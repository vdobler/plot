package plot

import (
	"fmt"
	"math"
)

// Stat is the interface of statistical transform.
//
// Statistical transform take a data frame and produce an other data frame.
// This is typically done by "summarizing", "modeling" or "transforming"
// the data in a statistically significant way.
//
// TODO: Location-/scale-invariance? f(x+a) = f(x)+a and f(x*a)=f(x*a) ??
type Stat interface {
	Name() string // Return the name of this stat.

	Apply(data *DataFrame, plot *Plot) *DataFrame

	// NeededAes are the aestetics which must be present in the
	// data frame. If not all needed aestetics are mapped this
	// statistics cannot be applied.
	NeededAes() []string

	// OptionalAes are the aestetocs which are used by this
	// statistics if present, but it is no error if they are
	// not mapped.
	OptionalAes() []string

	ExtraFieldHandling() ExtraFieldHandling
}

type ExtraFieldHandling int

const (
	IgnoreExtraFields ExtraFieldHandling = iota
	FailOnExtraFields
	GroupOnExtraFields
)

// -------------------------------------------------------------------------
// StatBin

type StatBin struct {
	BinWidth float64
	Drop     bool
	Origin   *float64 // TODO: both optional fields as *float64?
}

var _ Stat = StatBin{}

func (StatBin) Name() string                           { return "StatBin" }
func (StatBin) NeededAes() []string                    { return []string{"x"} }
func (StatBin) OptionalAes() []string                  { return []string{"weight"} }
func (StatBin) ExtraFieldHandling() ExtraFieldHandling { return GroupOnExtraFields }

func (s StatBin) Apply(data *DataFrame, plot *Plot) *DataFrame {
	if data == nil {
		return nil
	}
	min, max, _, _ := MinMax(data, "x")

	var binWidth float64 = s.BinWidth
	var numBins int

	var origin float64
	if binWidth == 0 {
		binWidth = (max - min) / 30
		numBins = 30
	} else {
		numBins = int((max-min)/binWidth + 0.5)
	}
	if s.Origin != nil {
		origin = *s.Origin
	} else {
		origin = math.Floor(min/binWidth) * binWidth // round origin TODO: might overflow
	}

	x2bin := func(x float64) int { return int((x - origin) / binWidth) }
	bin2x := func(b int) float64 { return float64(b)*binWidth + binWidth/2 }

	println("StatBin.Apply: binWidth =", binWidth, "   numBins =", numBins)
	counts := make([]int64, numBins+1) // TODO: Buggy here?
	column := data.Columns["x"].Data
	maxcount := int64(0)
	for i := 0; i < data.N; i++ {
		bin := x2bin(column[i])
		counts[bin]++
		if counts[bin] > maxcount {
			maxcount = counts[bin]
		}
	}

	result := NewDataFrame(fmt.Sprintf("%s binned by x", data.Name))
	nr := 0
	for _, count := range counts {
		if count == 0 && s.Drop {
			continue
		}
		nr++
	}

	result.N = nr
	X := NewField(nr)
	Count := NewField(nr)
	NCount := NewField(nr)
	Density := NewField(nr)
	NDensity := NewField(nr)
	X.Type = data.Columns["x"].Type
	Count.Type = Float
	NCount.Type = Float
	Density.Type = Float
	NDensity.Type = Float
	i := 0
	maxDensity := float64(0)
	for bin, count := range counts {
		if count == 0 && s.Drop {
			continue
		}
		X.Data[i] = bin2x(bin)
		Count.Data[i] = float64(count)
		NCount.Data[i] = float64(count) / float64(maxcount)
		density := float64(count) / binWidth / float64(data.N)
		Density.Data[i] = density
		if density > maxDensity {
			maxDensity = density
		}
		// println("bin =", bin, "   x =", bin2x(bin), "   count =", count)
		i++

	}
	i = 0
	for _, count := range counts {
		if count == 0 && s.Drop {
			continue
		}
		NDensity.Data[i] = Density.Data[i] / maxDensity
		i++
	}

	result.Columns["x"] = X
	result.Columns["count"] = Count
	result.Columns["ncount"] = NCount
	result.Columns["density"] = Density
	result.Columns["ndensity"] = NDensity

	return result

}

// -------------------------------------------------------------------------
// StatLinReg

type StatLinReq struct {
	A, B float64
}

var _ Stat = &StatLinReq{}

func (StatLinReq) Name() string                           { return "StatLinReq" }
func (StatLinReq) NeededAes() []string                    { return []string{"x", "y"} }
func (StatLinReq) OptionalAes() []string                  { return []string{"weight"} }
func (StatLinReq) ExtraFieldHandling() ExtraFieldHandling { return GroupOnExtraFields }

func (s *StatLinReq) Apply(data *DataFrame, plot *Plot) *DataFrame {
	if data == nil {
		return nil
	}
	xc, yc := data.Columns["x"].Data, data.Columns["y"].Data

	xm, ym := float64(0), float64(0)
	for i := 0; i < data.N; i++ {
		xm += xc[i]
		ym += yc[i]
	}
	xm /= float64(data.N)
	ym /= float64(data.N)

	sy, sx := float64(0), float64(0)
	for i := 0; i < data.N; i++ {
		x := xc[i]
		y := xc[i]
		dx := x - xm
		sx += dx * dx
		sy += dx * (y - ym)
	}

	s.B = sy / sx
	s.A = ym - s.B*xm
	aErr, bErr := s.A*0.2, s.B*0.1 // BUG

	result := NewDataFrame(fmt.Sprintf("linear regression of %s", data.Name))
	result.N = 1

	intercept, slope := NewField(1), NewField(1)
	intercept.Type, slope.Type = Float, Float
	intercept.Data[0], slope.Data[0] = s.A, s.B

	interceptErr, slopeErr := NewField(1), NewField(1)
	interceptErr.Type, slopeErr.Type = Float, Float
	interceptErr.Data[0], slopeErr.Data[0] = aErr, bErr

	result.Columns["intercept"] = intercept
	result.Columns["slope"] = slope
	result.Columns["interceptErr"] = interceptErr
	result.Columns["slopeErr"] = slopeErr
	return result
}

// -------------------------------------------------------------------------
// Stat Smooth

// Major TODO

type StatSmooth struct {
	A, B float64
}

var _ Stat = &StatSmooth{}

func (StatSmooth) Name() string                           { return "StatSmooth" }
func (StatSmooth) NeededAes() []string                    { return []string{"x", "y"} }
func (StatSmooth) OptionalAes() []string                  { return []string{"weight"} }
func (StatSmooth) ExtraFieldHandling() ExtraFieldHandling { return GroupOnExtraFields }

func (s *StatSmooth) Apply(data *DataFrame, plot *Plot) *DataFrame {
	if data == nil {
		return nil
	}
	xc, yc := data.Columns["x"].Data, data.Columns["y"].Data

	xm, ym := float64(0), float64(0)
	for i := 0; i < data.N; i++ {
		xm += xc[i]
		ym += yc[i]
	}
	xm /= float64(data.N)
	ym /= float64(data.N)

	sy, sx := float64(0), float64(0)
	for i := 0; i < data.N; i++ {
		x := xc[i]
		y := xc[i]
		dx := x - xm
		sx += dx * dx
		sy += dx * (y - ym)
	}

	s.B = sy / sx
	s.A = ym - s.B*xm
	aErr, bErr := s.A*0.2, s.B*0.1 // BUG

	result := NewDataFrame(fmt.Sprintf("linear regression of %s", data.Name))
	result.N = 100 // TODO
	xf := NewField(result.N)
	yf := NewField(result.N)
	yminf := NewField(result.N)
	ymaxf := NewField(result.N)
	xf.Type, yf.Type = Float, Float
	yminf.Type, ymaxf.Type = Float, Float

	minx, maxx, _, _ := MinMax(data, "x")
	// TODO: maybe rescale to full range
	xrange := maxx - minx
	for i := 0; i < result.N; i++ {
		x := minx + float64(i)*xrange/float64(result.N-1)
		xf.Data[i] = x
		yf.Data[i] = s.A*x + s.B
		yminf.Data[i] = (s.A-aErr)*x + (s.B - bErr) // BUG
		ymaxf.Data[i] = (s.A+aErr)*x + (s.B + bErr) // BUG
	}

	return result
}

// -------------------------------------------------------------------------
// StatLabel

type StatLabel struct {
	Format string
}

var _ Stat = StatLabel{}

func (StatLabel) Name() string                           { return "StatLabel" }
func (StatLabel) NeededAes() []string                    { return []string{"x", "y", "value"} }
func (StatLabel) OptionalAes() []string                  { return []string{"color"} }
func (StatLabel) ExtraFieldHandling() ExtraFieldHandling { return IgnoreExtraFields }

func (s StatLabel) Apply(data *DataFrame, plot *Plot) *DataFrame {
	println("==============\nStatLabel.Apply\n=============")
	result := NewDataFrame(fmt.Sprintf("labeling %s", data.Name))
	result.N = data.N
	textf := NewField(result.N)
	textf.Type = String

	value := data.Columns["value"].Data

	for i := 0; i < result.N; i++ {
		// BUG: what if value is time or string?
		t := fmt.Sprintf(s.Format, value[i])
		textf.Data[i] = float64(textf.AddStr(t))
	}

	result.Columns["x"] = data.Columns["x"].Copy()
	result.Columns["y"] = data.Columns["y"].Copy()
	result.Columns["text"] = textf

	return result

}
