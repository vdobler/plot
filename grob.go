package plot

import (
	"code.google.com/p/plotinum/vg"
	"fmt"
	"image/color"
	"math"
	"strings"
)

type Grob interface {
	Draw(vp Viewport)
	String() string
}

// -------------------------------------------------------------------------
// Grob Point

type GrobPoint struct {
	x, y  float64
	size  float64
	shape PointShape
	color color.Color
}

var _ Grob = GrobPoint{}

func (point GrobPoint) Draw(vp Viewport) {
	vp.Canvas.Push()
	vp.Canvas.SetColor(point.color)
	vp.Canvas.SetLineWidth(1)
	x, y := vp.X(point.x), vp.Y(point.y)
	s := vg.Points(point.size)
	var p vg.Path

	fmt.Printf("Point at %.0f,%.0f size %.1f (%.2f,%.2f)\n",
		x, y, s, point.x, point.y)

	draw := vp.Canvas.Stroke
	if point.shape >= SolidCirclePoint && point.shape <= SolidNablaPoint {
		draw = vp.Canvas.Fill
	}

	switch point.shape {
	case BlankPoint:
		return
	case DotPoint:
		dpi := vp.Canvas.DPI()
		p.Arc(x, y, vg.Inches(1/dpi), 0, 2*math.Pi)
		p.Close()
		vp.Canvas.Fill(p)
	case CirclePoint, SolidCirclePoint:
		p.Arc(x, y, vg.Points(point.size), 0, 2*math.Pi)
		p.Close()
		draw(p)
	case SquarePoint, SolidSquarePoint:
		p.Move(x-s, y-s)
		p.Line(x+s, y-s)
		p.Line(x+s, y+s)
		p.Line(x-s, y+s)
		p.Close()
		draw(p)
	case DiamondPoint, SolidDiamondPoint:
		p.Move(x, y-s)
		p.Line(x+s, y)
		p.Line(x, y+s)
		p.Line(x-s, y)
		p.Close()
		draw(p)
	case DeltaPoint, SolidDeltaPoint:
		ss := 0.57735 * s
		p.Move(x, y+2*ss)
		p.Line(x-s, y-ss)
		p.Line(x+s, y-ss)
		p.Close()
		draw(p)
	case NablaPoint, SolidNablaPoint:
		ss := 0.57735 * s
		p.Move(x, y-2*ss)
		p.Line(x-s, y+ss)
		p.Line(x+s, y+ss)
		p.Close()
		draw(p)
	case CrossPoint:
		ss := s / 1.3
		p.Move(x-ss, y-ss)
		p.Line(x+ss, y+ss)
		p.Move(x-ss, y+ss)
		p.Line(x+ss, y-ss)
		draw(p)
	case PlusPoint:
		p.Move(x-s, y)
		p.Line(x+s, y)
		p.Move(x, y-s)
		p.Line(x, y+s)
		draw(p)
	case StarPoint:
		ss := s / 1.3
		p.Move(x-ss, y-ss)
		p.Line(x+ss, y+ss)
		p.Move(x-ss, y+ss)
		p.Line(x+ss, y-ss)
		p.Move(x-s, y)
		p.Line(x+s, y)
		p.Move(x, y-s)
		p.Line(x, y+s)
		draw(p)
	default:
		println("Implement Draw for points " + point.shape.String())
	}
	vp.Canvas.Pop()
}

func (point GrobPoint) String() string {
	return fmt.Sprintf("Point(%.3f,%.3f %s %s %.1f)",
		point.x, point.y, Color2String(point.color),
		point.shape.String(), point.size)
}

// -------------------------------------------------------------------------
// Grob Line

type GrobLine struct {
	x0, y0, x1, y1 float64
	size           float64
	linetype       LineType
	color          color.Color
}

var _ Grob = GrobLine{}

var dashLength = [][]vg.Length{
	[]vg.Length{1},
	[]vg.Length{10},
	[]vg.Length{10, 10},
	[]vg.Length{5, 4},
	[]vg.Length{10, 4, 5, 4},
	[]vg.Length{20, 10},
	[]vg.Length{10, 10, 20, 10},
}

func (line GrobLine) Draw(vp Viewport) {
	vp.Canvas.SetColor(line.color)
	vp.Canvas.SetLineWidth(vg.Points(line.size))
	vp.Canvas.SetLineDash(dashLength[line.linetype], 0)
	x0, y0 := vp.X(line.x0), vp.Y(line.y0)
	x1, y1 := vp.X(line.x1), vp.Y(line.y1)
	var p vg.Path

	p.Move(x0, y0)
	p.Line(x1, y1)
	vp.Canvas.Stroke(p)
}

func (line GrobLine) String() string {
	return fmt.Sprintf("Line(%.3f,%.3f - %.3f,%.3f %s %s %.1f)",
		line.x0, line.y0, line.x1, line.y1,
		Color2String(line.color), line.linetype.String(),
		line.size)
}

// -------------------------------------------------------------------------
// Grob Path

type GrobPath struct {
	points   []struct{ x, y float64 }
	size     float64
	linetype LineType
	color    color.Color
}

var _ Grob = GrobPath{}

func (path GrobPath) Draw(vp Viewport) {
	vp.Canvas.Push()
	vp.Canvas.SetColor(path.color)
	vp.Canvas.SetLineWidth(vg.Points(path.size))
	vp.Canvas.SetLineDash(dashLength[path.linetype], 0)
	x, y := vp.X(path.points[0].x), vp.Y(path.points[0].y)
	var p vg.Path

	p.Move(x, y)
	for i := 1; i < len(path.points); i++ {
		x, y = vp.X(path.points[i].x), vp.Y(path.points[i].y)
		p.Line(x, y)
	}
	vp.Canvas.Stroke(p)
	vp.Canvas.Pop()
}

func (path GrobPath) String() string {
	// Pretty print points
	ppp := func(points []struct{ x, y float64 }) string {
		s := []string{}
		for _, p := range points {
			s = append(s, fmt.Sprintf("%.2f,%.2f", p.x, p.y))
		}
		return strings.Join(s, " - ")
	}
	var points string
	if n := len(path.points); n <= 6 {
		points = ppp(path.points)
	} else {
		points = ppp(path.points[0:3]) + " ... " + ppp(path.points[n-3:n])
	}
	return fmt.Sprintf("Path(%s %s %s %.1f)",
		points, Color2String(path.color), path.linetype.String(),
		path.size)
}

// -------------------------------------------------------------------------
// Grob Text

type GrobText struct {
	x, y  float64
	text  string
	size  float64
	color color.Color
	angle float64

	family, fontface string
	vjust, hjust     float64
	lineheight       float64
}

var _ Grob = GrobText{}

func (text GrobText) Draw(vp Viewport) {
	vp.Canvas.Push()
	vp.Canvas.SetColor(text.color)
	x, y := vp.X(text.x), vp.Y(text.y)
	vp.Canvas.Translate(x, y)
	vp.Canvas.Rotate(text.angle)
	font, err := vg.MakeFont("Courier", vg.Points(text.size))
	if err != nil {
		panic(err.Error())
	}
	vp.Canvas.FillString(font, 0, 0, text.text)
	vp.Canvas.Pop()
}

func (text GrobText) String() string {
	just := "r" // TODO: check if this is the proper def.
	if text.vjust < 1/3 {
		just = "l"
	} else if text.vjust < 2/3 {
		just = "c"
	}
	if text.hjust < 1/3 {
		just += "t"
	} else if text.hjust < 2/3 {
		just += "m"
	} else {
		just += "b"
	}

	fnt := fmt.Sprintf("%s/%s/%.0f", strings.Replace(text.family, " ", "-", -1),
		text.fontface, text.lineheight)

	return fmt.Sprintf("Text(%.3f,%.3f %q %s %.0f° %s %q)",
		text.x, text.y, text.text, Color2String(text.color),
		180*text.angle/math.Pi, just, fnt)
}

// -------------------------------------------------------------------------
// Grob Rect

type GrobRect struct {
	xmin, ymin float64
	xmax, ymax float64
	fill       color.Color
}

var _ Grob = GrobRect{}

func (rect GrobRect) Draw(vp Viewport) {
	vp.Canvas.Push()
	vp.Canvas.SetColor(rect.fill)
	vp.Canvas.SetLineWidth(2)
	xmin, ymin := vp.X(rect.xmin), vp.Y(rect.ymin)
	xmax, ymax := vp.X(rect.xmax), vp.Y(rect.ymax)
	var p vg.Path

	p.Move(xmin, ymin)
	p.Line(xmax, ymin)
	p.Line(xmax, ymax)
	p.Line(xmin, ymax)
	p.Close()
	vp.Canvas.Fill(p)
	vp.Canvas.Pop()
}

func (rect GrobRect) String() string {
	return fmt.Sprintf("Rect(%.3f,%.3f - %.3f,%.3f %s)",
		rect.xmin, rect.ymin, rect.xmax, rect.ymax,
		Color2String(rect.fill))
}

// -------------------------------------------------------------------------
// Viewport

type Viewport struct {
	// The lower left corner, width and height of this vp
	// in canvas units
	X0, Y0, Width, Height vg.Length
	Canvas                vg.Canvas
}

func (vp Viewport) String() string {
	return fmt.Sprintf("%.2f x %.2f + %.2f + %.2f (inches)",
		vp.X0.Inches(), vp.Y0.Inches(), vp.Width.Inches(), vp.Height.Inches())
}

// SubViewport returns the area described by x0,y0,width,height in
// natural grob coordinates [0,1] as a viewport.
func SubViewport(vp Viewport, x0, y0, width, height float64) Viewport {
	sub := Viewport{
		X0:     vp.X0 + vg.Length(x0)*vp.Width,
		Y0:     vp.Y0 + vg.Length(y0)*vp.Height,
		Width:  vg.Length(width) * vp.Width,
		Height: vg.Length(height) * vp.Height,
		Canvas: vp.Canvas,
	}

	fmt.Printf("SubVieport(width=%.2f) Width=%.2fin\n", width, sub.Width.Inches())
	return sub
}

// X and Y turn natural grob coordinates [0,1] to canvas lengths.
func (vp Viewport) X(x float64) vg.Length {
	ans := vp.X0 + vg.Length(x)*vp.Width
	// fmt.Printf("X( %.3f ) = %.1fin\n", x, ans.Inches())
	return ans
}
func (vp Viewport) Y(y float64) vg.Length {
	ans := vp.Y0 + vg.Length(y)*vp.Height
	// fmt.Printf("Y( %.3f ) = %.1fin\n", y, ans.Inches())
	return ans
}
