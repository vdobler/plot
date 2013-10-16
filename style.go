package plot

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"
)

func String2Float(s string, low, high float64) float64 {
	factor := 1.0
	if strings.HasSuffix("s", "%") {
		s = s[:len(s)-1]
		factor = 100
	}
	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		fmt.Printf("Cannot parse style %q as float: %s", s, err.Error())
		return 0.5
	}
	value /= factor

	if value < low {
		return low
	} else if value > high {
		return high
	}
	return value
}

// Set alpha to a in color c. TODO: handle case if c has alpha.
func SetAlpha(c color.Color, a float64) color.Color {
	r, g, b, _ := c.RGBA()
	r >>= 8
	g >>= 8
	b >>= 8
	a *= float64(0xff)
	return color.NRGBA{uint8(r), uint8(g), uint8(b), uint8(a)}
}

// -------------------------------------------------------------------------
// Points

type PointShape int

const (
	BlankPoint PointShape = iota
	CirclePoint
	SquarePoint
	DiamondPoint
	DeltaPoint
	NablaPoint
	SolidCirclePoint
	SolidSquarePoint
	SolidDiamondPoint
	SolidDeltaPoint
	SolidNablaPoint
	CrossPoint
	PlusPoint
	StarPoint
)

func String2PointShape(s string) PointShape {
	n, err := strconv.Atoi(s)
	if err == nil {
		return PointShape(n % (int(StarPoint) + 1))
	}
	switch s {
	case "circle":
		return CirclePoint
	case "square":
		return SquarePoint
	case "diamond":
		return DiamondPoint
	case "delta":
		return DeltaPoint
	case "nabla":
		return NablaPoint
	case "solid-circle":
		return SolidCirclePoint
	case "solid-square":
		return SolidSquarePoint
	case "solid-diamond":
		return SolidDiamondPoint
	case "solid-delta":
		return SolidDeltaPoint
	case "solid-nabla":
		return SolidNablaPoint
	case "cross":
		return CrossPoint
	case "plus":
		return PlusPoint
	case "star":
		return StarPoint
	}
	return BlankPoint
}

func String2PointSize(s string) float64 {
	n, err := strconv.Atoi(s)
	if err == nil {
		return float64(n)
	}
	return 6
}

// -------------------------------------------------------------------------
// Lines

type LineType int

const (
	BlankLine LineType = iota
	SolidLine
	DashedLine
	DottedLine
	DotDashLine
	LongdashLine
	TwodashLine
)

func String2LineType(s string) LineType {
	n, err := strconv.Atoi(s)
	if err == nil {
		return LineType(n % (int(TwodashLine) + 1))
	}
	switch s {
	case "blank":
		return BlankLine
	case "solid":
		return SolidLine
	case "dashed":
		return DashedLine
	case "dotted":
		return DottedLine
	case "dotdash":
		return DotDashLine
	case "longdash":
		return LongdashLine
	case "twodash":
		return TwodashLine
	default:
		return BlankLine
	}
}

// -------------------------------------------------------------------------
// Colors

var BuiltinColors = map[string]color.NRGBA{
	"red":     color.NRGBA{0xff, 0x00, 0x00, 0xff},
	"green":   color.NRGBA{0x00, 0xff, 0x00, 0xff},
	"blue":    color.NRGBA{0x00, 0x00, 0xff, 0xff},
	"cyan":    color.NRGBA{0x00, 0xff, 0xff, 0xff},
	"magenta": color.NRGBA{0xff, 0x00, 0xff, 0xff},
	"yellow":  color.NRGBA{0xff, 0xff, 0x00, 0xff},
	"white":   color.NRGBA{0xff, 0xff, 0xff, 0xff},
	"gray20":  color.NRGBA{0x33, 0x33, 0x33, 0xff},
	"gray40":  color.NRGBA{0x66, 0x66, 0x66, 0xff},
	"gray":    color.NRGBA{0x7f, 0x7f, 0x7f, 0xff},
	"gray60":  color.NRGBA{0x99, 0x99, 0x99, 0xff},
	"gray80":  color.NRGBA{0xcc, 0xcc, 0xcc, 0xff},
	"black":   color.NRGBA{0x00, 0x00, 0x00, 0xff},
}

func String2Color(s string) color.Color {
	if strings.HasPrefix(s, "#") && len(s) >= 7 {
		var r, g, b, a uint8
		fmt.Sscanf(s[1:3], "%2x", &r)
		fmt.Sscanf(s[3:5], "%2x", &g)
		fmt.Sscanf(s[5:7], "%2x", &b)
		a = 0xff
		if len(s) >= 9 {
			fmt.Sscanf(s[7:9], "%2x", &a)
		}
		return color.RGBA{r, g, b, a}
	}
	if col, ok := BuiltinColors[s]; ok {
		return col
	}

	return color.RGBA{0xaa, 0x66, 0x77, 0x7f}
}

// -------------------------------------------------------------------------
// Accessor functions

// Return a function which maps row number in df to a color.
// The color is produced by the appropriate scale of plot
// or a fixed value defined in aes.
func makeColorFunc(aes string, data *DataFrame, plot *Plot, style AesMapping) func(i int) color.Color {
	var f func(i int) color.Color
	if data.Has(aes) {
		d := data.Columns[aes].Data
		f = func(i int) color.Color {
			return plot.Scales[aes].Color(d[i])
		}
	} else {
		theColor := String2Color(style[aes])
		f = func(int) color.Color {
			return theColor
		}
	}
	return f
}

func makePosFunc(aes string, data *DataFrame, plot *Plot, style AesMapping) func(i int) float64 {
	var f func(i int) float64
	if data.Has(aes) {
		d := data.Columns[aes].Data
		f = func(i int) float64 {
			return plot.Scales[aes].Pos(d[i])
		}
	} else {
		x := String2Float(style[aes], math.Inf(-1), math.Inf(+1))
		f = func(int) float64 {
			return x
		}
	}
	return f
}

func makeStyleFunc(aes string, data *DataFrame, plot *Plot, style AesMapping) func(i int) int {
	var f func(i int) int
	if data.Has(aes) {
		d := data.Columns[aes].Data
		f = func(i int) int {
			return plot.Scales[aes].Style(d[i])
		}
	} else {
		var x int
		switch aes {
		case "shape":
			x = int(String2PointShape(style[aes]))
		case "linetype":
			x = int(String2LineType(style[aes]))
		default:
			fmt.Printf("Oooops, this should not happen.")
		}
		f = func(int) int {
			return x
		}
	}
	return f
}
