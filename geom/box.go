package geom

import (
	"github.com/vdobler/plot"
	"image/color"
)

type Box struct {
	Width float64 // TODO: populated where? Here? How?
	Color color.Color
}

func (b Box) Render(data plot.DataFrame, aes plot.AesMapping, p plot.Plot) {
	am := p.Aes.Merge(aes, plot.DefaultTheme.BoxAes)
	for i := 0; i<plot.Length(data); i++ {
		x := plot.Field(data,aes.X)
		lower := plot.Field(data,aes.Lower)
		upper := plot.Field(data,aes.Upper)

		w := b.Width

		var fill color.Colour
		if plot.FixedAes(am.Fill) {
			fill = am.FixedColor(am.Fill)
		} else {
			fill = plot.Field(data, am.Fill)
		}

		// now draw a rectangle from
		//  (x-w/2, lower) to (x+w/2, upper) filled with fill
	}
}
