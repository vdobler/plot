package plot

type Theme struct {
	PointStyle, LineStyle, BarStyle AesMapping
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
}
