package plot

type Theme struct {
	PointStyle, LineStyle, BarStyle AesMapping
	TextStyle                       AesMapping
	RectStyle                       AesMapping
}

var DefaultTheme = Theme{
	PointStyle: AesMapping{
		"size":  "5",
		"shape": "circle",
		"color": "#222222",
		"fill":  "#222222",
		"alpha": "1",
	},
	LineStyle: AesMapping{
		"size":     "2",
		"linetype": "solid",
		"color":    "#222222",
		"alpha":    "1",
	},
	BarStyle: AesMapping{
		"linetype": "blank",
		"color":    "gray20",
		"fill":     "gray20",
		"alpha":    "1",
	},
	TextStyle: AesMapping{
		"family":     "Helvetica",
		"fontface":   "regular",
		"lineheight": "15",
		"color":      "black",
		"vjust":      "0.5",
		"hjust":      "0.5",
		"angle":      "0",
	},
	RectStyle: AesMapping{
		"linetype": "blank",
		"color":    "#00000000",
		"fill":     "gray50",
		"alpha":    "1",
	},
}
