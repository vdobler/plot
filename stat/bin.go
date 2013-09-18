package stat

import (
	"github.com/vdobler/plot"
)

// Bin groups data into bins and counts occurencens in these bins.
// A nil options will use the default Options.
func Bin(data plot.DataFrame, options *BinOptions) []BinnedData {

}

type BinnedData struct {
	X        float64
	Count    int64
	Density  float64
	NCount   float64
	NDensity float64
}

type BinOptions struct {
	BinWidth float64
}

// BoxPlot calculates components of a box and whisker plot.
func BoxPlot(data plot.DataFrame, coef float64) []BoxPlotData {

}

type BoxPlotData struct {
	Width        float64
	YMin, Lower  float64
	Middle       float64
	Higher, YMax float64
	Outliers     []float64
}

type BoxPlotOptions struct {
	Coef
}
