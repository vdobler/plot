package plot

// Theme contains the stylable parameters of a plot.
type Theme struct {
	PointStyle, LineStyle, BarStyle AesMapping
	TextStyle, RectStyle            AesMapping
	PanelBG, GridMajor, GridMinor   AesMapping
	Strip, TicLabel, Tic            AesMapping
	Title, Label                    AesMapping
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
		"linetype": "solid",
		"color":    "gray50",
		"fill":     "gray50",
		"alpha":    "1",
	},
	PanelBG: AesMapping{
		"linetype": "blank",
		"color":    "#00000000",
		"size":     "0",
		"fill":     "gray80",
		"alpha":    "1",
	},
	GridMajor: AesMapping{
		"linetype": "solid",
		"color":    "white",
		"size":     "2",
		"alpha":    "1",
	},
	GridMinor: AesMapping{
		"linetype": "solid",
		"color":    "white",
		"size":     "1",
		"alpha":    "1",
	},
	Tic: AesMapping{
		"linetype": "solid",
		"color":    "gray40",
		"size":     "2",
		"length":   "2 mm",
		"alpha":    "1",
	},
	TicLabel: AesMapping{
		"color": "gray20",
		"size":  "12 pt",
		"angle": "0",
		"sep":   "0.5 mm",
	},
	Strip: AesMapping{
		"linetype": "blank",
		"color":    "black",  // Color of text
		"size":     "10 pt",  // Font size
		"fill":     "gray60", // Background color
		"alpha":    "1",
	},
	Title: AesMapping{
		"color": "black",
		"size":  "16 pt",
		"alpha": "1",
	},
	Label: AesMapping{
		"color": "black",
		"size":  "14 pt",
		"alpha": "1",
	},
}
