package plot

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"
)

// String2Float parses s into a float64. The output is clipped to the
// interval [low,high]. The string s may end in some common units,
// namely %, °, pt, mm, cm and in.
func String2Float(s string, low, high float64) float64 {
	factor := 1.0
	if strings.HasSuffix(s, "%") {
		s = s[:len(s)-1]
		factor = 1 / 100
	} else if strings.HasSuffix(s, "°") {
		s = s[:len(s)-1]
		factor = math.Pi / 180
	} else if strings.HasSuffix(s, "pt") {
		s = s[:len(s)-2]
		factor = 1
	} else if strings.HasSuffix(s, "mm") {
		s = s[:len(s)-2]
		factor = 2.83464567
	} else if strings.HasSuffix(s, "cm") {
		s = s[:len(s)-2]
		factor = 28.3464567
	} else if strings.HasSuffix(s, "in") {
		s = s[:len(s)-2]
		factor = 72
	}
	s = strings.TrimSpace(s)

	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		fmt.Printf("Cannot parse style %q as float: %s\n", s, err.Error())
		return 1
	}
	value *= factor

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
	DotPoint
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

var shapeNames = []string{"blank", "dot", "circle", "square", "diamond", "delta", "nabla",
	"solidcircle", "solidsquare", "soliddiamond", "soliddelta", "solidnabla",
	"cross", "plus", "star"}

func (s PointShape) String() string {
	return shapeNames[s]
}

func String2PointShape(s string) PointShape {
	n, err := strconv.Atoi(s)
	if err == nil {
		return PointShape(n % (int(StarPoint) + 1))
	}
	for i, n := range shapeNames {
		if n == s {
			return PointShape(i)
		}
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

var linetypeNames = []string{"blank", "solid", "dashed", "dotted", "dotdash",
	"longdash", "twodash"}

func (lt LineType) String() string {
	return linetypeNames[lt]
}

func String2LineType(s string) LineType {
	n, err := strconv.Atoi(s)
	if err == nil {
		return LineType(n % (int(TwodashLine) + 1))
	}
	for i, n := range linetypeNames {
		if n == s {
			return LineType(i)
		}
	}
	return BlankLine
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
		return color.NRGBA{r, g, b, a}
	}
	if col, ok := BuiltinColors[s]; ok {
		return col
	}

	return color.NRGBA{0xaa, 0x66, 0x77, 0x7f}
}

func Color2String(c color.Color) string {
	if c == nil {
		return ""
	}
	r, g, b, a := c.RGBA()
	r >>= 8
	g >>= 8
	b >>= 8
	s := fmt.Sprintf("#%02x%02x%02x", r, g, b)
	if a != 0xffff {
		s += fmt.Sprintf("%02x", a>>8)
	}
	return s
}

// -------------------------------------------------------------------------
// Accessor functions

// Return a function which maps row number i in data to a color.
// The color is produced by the appropriate scale of the panel
// or a fixed value defined in aes.
func makeColorFunc(aes string, data *DataFrame, panel *Panel, style AesMapping) func(i int) color.Color {
	var f func(i int) color.Color
	if data.Has(aes) {
		d := data.Columns[aes].Data
		f = func(i int) color.Color {
			return panel.Scales[aes].Color(d[i])
		}
	} else {
		theColor := String2Color(style[aes])
		f = func(int) color.Color {
			return theColor
		}
	}
	return f
}

// Return a function which maps row number i in data to continuous float.
// If data contains aes the value returned is in the range [min,max] and
// the appropriate scale of panel is used to map from data to [min,max];
// otherwise the value is directly taken from style.
func makePosFunc(aes string, data *DataFrame, panel *Panel, style AesMapping, min, max float64) func(i int) float64 {
	var f func(i int) float64
	if data.Has(aes) {
		d := data.Columns[aes].Data
		f = func(i int) float64 {
			orig := panel.Scales[aes].Pos(d[i])
			return orig*(max-min) + min
		}
	} else {
		x := String2Float(style[aes], math.Inf(-1), math.Inf(+1))
		f = func(int) float64 {
			return x
		}
	}
	return f
}

func makeStyleFunc(aes string, data *DataFrame, panel *Panel, style AesMapping) func(i int) int {
	var f func(i int) int
	if data.Has(aes) {
		d := data.Columns[aes].Data
		f = func(i int) int {
			return panel.Scales[aes].Style(d[i])
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

func MergeStyles(aes ...AesMapping) AesMapping {
	result := make(AesMapping)
	for _, a := range aes {
		if len(a) == 0 {
			continue
		}

		for k, v := range a {
			if _, ok := result[k]; !ok {
				result[k] = v
			}
		}
	}
	return result
}
