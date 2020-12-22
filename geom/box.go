package geom

import (
	"image/color"

	"github.com/vdobler/plot"
)

type Box struct {
	Width float64 // TODO: populated where? Here? How?
	Color color.Color
}

func (b Box) Render(data plot.DataFrame, aes plot.AesMapping, pt plot.Plot) {
	am := plot.MergeAes(aes, plot.DefaultTheme.BoxAes)
	for i := 0; i < plot.Length(data); i++ {
		x := pt.Field(data, aes.X)
		lower := pt.Field(data, aes.Lower)
		upper := pt.Field(data, aes.Upper)

		w := b.Width

		var fill color.Color
		if plot.FixedAes(am.Fill) {
			fill = am.FixedColor(am.Fill)
		} else {
			fill = pt.Field(data, am.Fill)
		}

		// now draw a rectangle from
		//  (x-w/2, lower) to (x+w/2, upper) filled with fill
	}
}
