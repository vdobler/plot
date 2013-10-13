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
	Apply(data *DataFrame, mapping AesMapping) *DataFrame

	// NeededAes are the aestetics which must be present in the
	// data frame. If not all needed aestetics are mapped this
	// statistics cannot be applied.
	NeededAes() []string

	// OptionalAes are the aestetocs which are used by this
	// statistics if present, but it is no error if they are
	// not mapped.
	OptionalAes() []string
}

type StatBin struct {
	BinWidth float64
	Drop     bool
	Origin   *float64 // TODO: both optional fields as *float64?
}

func (StatBin) NeededAes() []string { return []string{"x"} }
func (StatBin) OptionalAes() []string { return []string{"weight"} }

func (s StatBin) Apply(data *DataFrame, mapping AesMapping) *DataFrame {
	if data == nil {
		return nil
	}
	field := mapping["x"]
	min, max, _, _ := MinMax(data, field)

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
	counts := make([]int64, numBins+1) // TODO: Buggy here
	column := data.Columns[field].Data
	maxcount := int64(0)
	for i := 0; i < data.N; i++ {
		bin := x2bin(column[i])
		counts[bin]++
		if counts[bin] > maxcount {
			maxcount = counts[bin]
		}
	}

	result := NewDataFrame(fmt.Sprintf("%s binned by %s", data.Name, field))
	nr := 0
	for _, count := range counts {
		if count == 0 && s.Drop {
			continue
		}
		nr++
	}
	X := NewField(nr)
	Count := NewField(nr)
	NCount := NewField(nr)
	Density := NewField(nr)
	NDensity := NewField(nr)
	X.Type = data.Columns[field].Type
	Count.Type = Int
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

	}
	i = 0
	for _, count := range counts {
		if count == 0 && s.Drop {
			continue
		}
		NDensity.Data[i] = Density.Data[i] / maxDensity
		i++
	}

	return result

}

// -------------------------------------------------------------------------
// StatLinReg

type StatLinReq struct {
	A, B float64
}

func (StatLinReq) NeededAes() []string { return []string{"x", "y"}}
func (StatBin) OptionalAes() []string { return []string{"weight"} }

func (s *StatLinReq) Apply(data *DataFrame, mapping AesMapping) *DataFrame {
	if data == nil {
		return nil
	}
	fx, fy := "X", "Y" // BUG
	xc, yc := data.Columns[fx].Data, data.Columns[fy].Data

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

	minx, maxx, _, _ := MinMax(data, fx)
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
