package plot

import (
	"fmt"
	"math"
	"os"
	"sort"
)

var _ = os.Open

// Stat is the interface of statistical transform.
//
// Statistical transform take a data frame and produce an other data frame.
// This is typically done by "summarizing", "modeling" or "transforming"
// the data in a statistically significant way.
//
// TODO: Location-/scale-invariance? f(x+a) = f(x)+a and f(x*a)=f(x*a) ??
type Stat interface {
	// Name returns the name of this statistic.
	Name() string

	// Apply this statistic to data. The panel can be used to
	// access the current scales, e.g. if the x-range is needed.
	Apply(data *DataFrame, panel *Panel) *DataFrame

	// Info returns the StatInfo which describes how this
	// statistic can be used.
	Info() StatInfo
}

// StatInfo contains information about how a stat can be used.
type StatInfo struct {
	// NeededAes are the aestetics which must be present in the
	// data frame. If not all needed aestetics are mapped this
	// statistics cannot be applied.
	NeededAes []string

	// OptionalAes are the aestetocs which are used by this
	// statistics if present, but it is no error if they are
	// not mapped.
	OptionalAes []string

	ExtraFieldHandling ExtraFieldHandling

	// TODO: Add information about resulting data frame?
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

func (StatBin) Name() string { return "StatBin" }

func (StatBin) Info() StatInfo {
	return StatInfo{
		NeededAes:          []string{"x"},
		OptionalAes:        []string{"weight"},
		ExtraFieldHandling: GroupOnExtraFields,
	}
}

func (s StatBin) Apply(data *DataFrame, _ *Panel) *DataFrame {
	if data == nil || data.N == 0 {
		return nil
	}

	// println("StatBin Data:")
	// data.Print(os.Stdout)

	min, max, mini, maxi := MinMax(data, "x")
	if mini == -1 && maxi == -1 {
		return nil
	}
	// println("min/max", min, max)
	if min == max {
		// TODO. Also NaN and Inf
		min -= 1
		max += 1
	}

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
	bin2x := func(b int) float64 { return float64(b)*binWidth + binWidth/2 + origin }

	counts := make([]int64, numBins+1) // TODO: Buggy here?
	// println("StatBin, made counts", len(counts), min, max, origin, binWidth)
	column := data.Columns["x"].Data
	maxcount := int64(0)
	for i := 0; i < data.N; i++ {
		bin := x2bin(column[i])
		// println("  StatBin ", i, column[i], bin)
		counts[bin]++
		if counts[bin] > maxcount {
			maxcount = counts[bin]
		}
	}

	pool := data.Pool
	result := NewDataFrame(fmt.Sprintf("%s binned by x", data.Name), pool)
	nr := 0
	for _, count := range counts {
		if count == 0 && s.Drop {
			continue
		}
		nr++
	}

	result.N = nr
	X := NewField(nr, data.Columns["x"].Type, pool)
	Count := NewField(nr, Float, pool) // TODO: Int?
	NCount := NewField(nr, Float, pool)
	Density := NewField(nr, Float, pool)
	NDensity := NewField(nr, Float, pool)
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
	// TODO: all in one loop?
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

func (StatLinReq) Name() string { return "StatLinReq" }

func (StatLinReq) Info() StatInfo {
	return StatInfo{
		NeededAes:          []string{"x", "y"},
		OptionalAes:        []string{"weight"},
		ExtraFieldHandling: GroupOnExtraFields,
	}
}

func (s *StatLinReq) Apply(data *DataFrame, _ *Panel) *DataFrame {
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
		y := yc[i]
		dx := x - xm
		sx += dx * dx
		sy += dx * (y - ym)
	}

	s.B = sy / sx
	s.A = ym - s.B*xm
	aErr, bErr := s.A*0.2, s.B*0.1 // BUG
	// See http://en.wikipedia.org/wiki/Simple_linear_regression#Normality_assumption
	// for convidance intervalls of A and B.

	pool := data.Pool
	result := NewDataFrame(fmt.Sprintf("linear regression of %s", data.Name), pool)
	result.N = 1

	intercept, slope := NewField(1, Float, pool), NewField(1, Float, pool)
	intercept.Data[0], slope.Data[0] = s.A, s.B

	interceptErr, slopeErr := NewField(1, Float, pool), NewField(1, Float, pool)
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

func (StatSmooth) Name() string { return "StatSmooth" }

func (StatSmooth) Info() StatInfo {
	return StatInfo{
		NeededAes:          []string{"x", "y"},
		OptionalAes:        []string{"weight"},
		ExtraFieldHandling: GroupOnExtraFields,
	}
}

func (s *StatSmooth) Apply(data *DataFrame, _ *Panel) *DataFrame {
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

	pool := data.Pool
	result := NewDataFrame(fmt.Sprintf("linear regression of %s", data.Name), pool)
	result.N = 100 // TODO
	xf := NewField(result.N, Float, pool)
	yf := NewField(result.N, Float, pool)
	yminf := NewField(result.N, Float, pool)
	ymaxf := NewField(result.N, Float, pool)

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

func (StatLabel) Name() string { return "StatLabel" }

func (StatLabel) Info() StatInfo {
	return StatInfo{
		NeededAes:          []string{"x", "y", "value"},
		OptionalAes:        []string{"color"},
		ExtraFieldHandling: IgnoreExtraFields,
	}
}

func (s StatLabel) Apply(data *DataFrame, _ *Panel) *DataFrame {
	pool := data.Pool
	result := NewDataFrame(fmt.Sprintf("labeling %s", data.Name), pool)
	result.N = data.N
	textf := NewField(result.N, String, pool)

	value := data.Columns["value"].Data

	for i := 0; i < result.N; i++ {
		// BUG: what if value is time or string?
		t := fmt.Sprintf(s.Format, value[i])
		textf.Data[i] = float64(pool.Add(t))
	}

	result.Columns["x"] = data.Columns["x"].Copy()
	result.Columns["y"] = data.Columns["y"].Copy()
	result.Columns["text"] = textf

	return result

}

// -------------------------------------------------------------------------
// StatFunction

// StatFunction draws the functions F interpolating it by N points.
type StatFunction struct {
	F func(x float64) float64
	N int
}

var _ Stat = StatFunction{}

func (StatFunction) Name() string { return "StatFunction" }

func (StatFunction) Info() StatInfo {
	return StatInfo{
		NeededAes:          []string{},
		OptionalAes:        []string{},
		ExtraFieldHandling: IgnoreExtraFields,
	}
}

func (s StatFunction) Apply(data *DataFrame, panel *Panel) *DataFrame {
	sx := panel.Scales["x"]
	n := s.N
	if n == 0 {
		n = 101
	}
	xmin, xmax := sx.DomainMin, sx.DomainMax // TODO
	fmt.Printf("StatFunction %.2f -- %.2f\n", xmin, xmax)

	delta := (xmax - xmin) / float64(n-1)

	result := NewDataFrame("function", data.Pool)
	result.N = n
	xf := NewField(n, Float, data.Pool)
	yf := NewField(n, Float, data.Pool)

	for i := 0; i < n; i++ {
		x := xmin + float64(i)*delta
		xf.Data[i] = x
		yf.Data[i] = s.F(x)
		if i%10 == 0 {
			fmt.Printf("sin:  x=%.2f  y=%.2f\n", x, yf.Data[i])
		}
	}

	result.Columns["x"] = xf
	result.Columns["y"] = yf

	return result

}

// -------------------------------------------------------------------------
// StatBoxplot

type StatBoxplot struct {
}

var _ Stat = StatBoxplot{}

func (StatBoxplot) Name() string { return "StatBoxplot" }

func (StatBoxplot) Info() StatInfo {
	return StatInfo{
		NeededAes:          []string{"x", "y"},
		OptionalAes:        []string{},
		ExtraFieldHandling: GroupOnExtraFields,
	}
}

type boxplot struct {
	min, low, q1, med, q3, high, max float64
	outliers                         []float64
}

// TODO: handle extreme cases
func computeBoxplot(d []float64) (b boxplot) {
	n := len(d)
	sort.Float64s(d)

	// Compute the five boxplot values.
	b.min, b.max = d[0], d[n-1]
	if n%2 == 1 {
		b.med = d[(n-1)/2]
	} else {
		b.med = (d[n/2] + d[n/2-1]) / 2
	}
	b.q1, b.q3 = d[n/4], d[3*n/4]

	iqr := b.q3 - b.q1
	lo, hi := b.q1-1.5*iqr, b.q3+1.5*iqr
	b.low, b.high = b.max, b.min

	// Compute low, high and outliers.
	for _, y := range d {
		if y >= lo && y < b.low {
			b.low = y
		}
		if y <= hi && y > b.high {
			b.high = y
		}
		if y < lo || y > hi {
			b.outliers = append(b.outliers, y)
		}
	}

	return b
}

func (s StatBoxplot) Apply(data *DataFrame, _ *Panel) *DataFrame {
	if data == nil || data.N == 0 {
		return nil
	}
	xd, yd := data.Columns["x"].Data, data.Columns["y"].Data

	xs := Levels(data, "x").Elements()
	sort.Float64s(xs)
	n := len(xs)
	fmt.Printf("StatBoxplot of %d values %v\n", n, xs)
	ys := make(map[float64][]float64)

	pool := data.Pool
	xf := NewField(n, data.Columns["x"].Type, pool)
	medf := NewField(n, Float, pool)
	minf, maxf := NewField(n, Float, pool), NewField(n, Float, pool)
	lowf, highf := NewField(n, Float, pool), NewField(n, Float, pool)
	q1f, q3f := NewField(n, Float, pool), NewField(n, Float, pool)

	for i := 0; i < data.N; i++ {
		x, y := xd[i], yd[i]
		ys[x] = append(ys[x], y)
	}
	i := 0
	for x, y := range ys {
		b := computeBoxplot(y)
		xf.Data[i] = x
		minf.Data[i] = b.min
		lowf.Data[i] = b.low
		q1f.Data[i] = b.q1
		medf.Data[i] = b.med
		q3f.Data[i] = b.q3
		highf.Data[i] = b.high
		maxf.Data[i] = b.max
		i++
		fmt.Printf("x=%.1f  %d samples, median=%.1f\n", x, len(y), b.med)
	}

	result := NewDataFrame(fmt.Sprintf("boxplot of %s", data.Name), pool)
	result.N = n
	result.Columns["x"] = xf
	result.Columns["min"] = minf
	result.Columns["low"] = lowf
	result.Columns["q1"] = q1f
	result.Columns["mid"] = medf
	result.Columns["q3"] = q3f
	result.Columns["high"] = highf
	result.Columns["max"] = maxf

	fmt.Printf("Result of StatBoxplot:\n")
	result.Print(os.Stdout)
	return result

}
