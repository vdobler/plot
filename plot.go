package plot

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/vgimg"
)

var now = time.Now
var col = color.RGBA{}
var floor = math.Floor
var _ = os.Open

// Plot represents a whole plot.
type Plot struct {
	// The title of the plot.
	Title string

	// Data is the data to draw. If nil all layers must provide their
	// own data.
	Data *DataFrame

	// Faceting describes the used Faceting.
	Faceting Faceting

	// Mapping describes how fieleds in data are mapped to Aesthetics.
	Aes AesMapping

	// Layers contains all the layers displayed in the plot.
	Layers []*Layer

	// Scales can be used to set up non-default scales, e.g.
	// scales with a transformation or manualy set breaks.
	//
	// The scales of plot are distributed to the individual panels
	// only and are not used directly as scales.
	Scales map[string]*Scale

	// Panels are the different panels for faceting.
	Panels [][]*Panel

	// Theme contains the visual defaults to use when drawing things.
	Theme Theme

	// Layout maps components like "Title" or "Y-Label" to their viewport.
	Viewports map[string]Viewport

	// Pool is the string pool used to keep the string-float64 mapping
	Pool *StringPool

	// Grobs contains the handful of plot (not part of panels) of
	// graphical objects.
	Grobs map[string]Grob

	constructed bool

	// ugly hack to save dimensions of plot visuals between rendering
	// and layouting...
	renderInfo map[string]vg.Length
}

// NewPlot creates a new plot. Data must be a SOM or a COS and aesthetics
// maps fields in data to plot aesthetics.
func NewPlot(data interface{}, aesthetics AesMapping) (*Plot, error) {
	pool := NewStringPool()
	df, err := NewDataFrameFrom(data, pool)
	if err != nil {
		return nil, err
	}

	if aesthetics == nil {
		aesthetics = make(AesMapping)
	}

	plot := Plot{
		Data:        df,
		Faceting:    Faceting{},
		Aes:         aesthetics,
		Layers:      nil,
		Scales:      make(map[string]*Scale),
		Panels:      nil,
		Theme:       DefaultTheme,
		Pool:        pool,
		Grobs:       make(map[string]Grob),
		constructed: false,
		renderInfo:  make(map[string]vg.Length),
	}

	return &plot, nil
}

// Compute facets the plot data, computes the statistics, constructs the
// geoms, scales the axes and prepares everything for rendering the plot.
func (plot *Plot) Compute() {
	plot.CreatePanels()

	for r := range plot.Panels {
		for c := range plot.Panels[r] {
			panel := plot.Panels[r][c]

			// Prepare data: map aestetics, add scales, clean data frame and
			// apply scale transformations. Mapped scales are pre-trained.
			// Step 2
			panel.PrepareData()

			// The second step: Compute statistics.
			// If a layer has a statistical transform: Apply this transformation
			// to the data frame of this layer.
			// Step 3
			panel.ComputeStatistics()

			// Make sure the output of the stat matches the input expected
			// by the geom.
			// Step 4
			panel.WireStatToGeom()

			// Construct geoms
			// Apply geom specific position adjustments, train the involved
			// scales and produce a set of fundamental geoms.
			// Step 5
			panel.ConstructGeoms()
		}
	}

	for r := range plot.Panels {
		for _, panel := range plot.Panels[r] {

			// Finalize scales: Setup remaining fields.
			// This can be done only after each panel completed
			// the steps 2-5 ConstructGeoms which might change
			// the scales.
			// Step 6
			panel.FinalizeScales()
		}
	}

	for r := range plot.Panels {
		for _, panel := range plot.Panels[r] {
			// Render the fundamental Geoms to Grobs using scales.
			// Step 7
			panel.RenderGeoms()
		}
	}

	// Render rest of elements (guides, titels, factting, ...)
	// Step 8
	plot.RenderVisuals()

	plot.constructed = true
}

// DumpTo will render plot to canvas. The size of the generetad plot is
// determined by width and height.
func (plot *Plot) DumpTo(canvas vg.Canvas, width, height vg.Length) {
	if !plot.constructed {
		plot.Compute()
	}

	// Layouting the plot determines the viewports of all the elements
	// in the plot, especially the panels.
	plot.Layout(canvas, width, height)

	// Actual drawing of the general stuff.
	for _, element := range []string{"Title", "X-Label", "Y-Label", "Guides"} {
		if grob, ok := plot.Grobs[element]; ok {
			grob.Draw(plot.Viewports[element])
		}
	}

	// Drawing of the individual panels.
	showX, showY := false, false
	for r := range plot.Panels {
		showX = r == 0
		for c, panel := range plot.Panels[r] {
			showY = c == 0
			panelId := fmt.Sprintf("Panel-%d,%d", r, c)
			panel.Draw(plot.Viewports[panelId], showX, showY)
		}
	}
}

// WritePNG renders plot to a png file of size width x height.
func (p *Plot) WritePNG(filename string, width, height vg.Length) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	canvas := vgimg.PngCanvas{Canvas: vgimg.New(width, height)}
	// canvas.Translate(vg.Point{-width / 2, -height / 2})
	p.DumpTo(canvas, width, height)
	canvas.WriteTo(file)
	return nil
}

// Warnf prints args formated by f. TODO: proper error handling?
func (p *Plot) Warnf(f string, args ...interface{}) {
	if !strings.HasSuffix(f, "\n") {
		f = f + "\n"
	}
	fmt.Printf("Warning "+f, args...)
}

// -------------------------------------------------------------------------
// Layers, Panels and Faceting

// Layer represents one layer of data in a plot.
type Layer struct {
	// The Name of the layer used used in printing.
	Name string

	// Panel is the Panal this Layer belogs to.
	Panel *Panel

	// A nil Data will use the Data from the plot this layer belongs to.
	Data *DataFrame

	// DataMapping is combined with the plot's AesMapping and used to
	// map fields in Data to aesthetics.
	// TODO: Why not name it AesMapping like in Plot?
	DataMapping AesMapping

	// Stat is the statistical transformation used in this layer.
	// A nil stat is the identity transformation.
	Stat Stat

	// StatMapping is used to map new (i.e. generated by the stat) fields
	// to plot aesthetics.
	StatMapping AesMapping

	// Geom is the geom to use for this layer.
	Geom Geom

	// GeomMapping is used to wire fields output by the statistical
	// transform to the input fields used by the geom.
	GeomMapping AesMapping

	// The fundamental geoms to draw.
	Fundamentals []Fundamental

	// Grobs contains the graphical object after drawing.
	Grobs []Grob
}

// A Panel is one panel, typically in a facetted plot.
// It does not differ much from a Plot as it actually represents
// one of the plots in a facetted plot.
type Panel struct {
	// Name of the Panel
	Name string

	// The plot this panel belongs to
	Plot *Plot

	// Data is the data to draw.
	Data *DataFrame

	// Mapping, describes how fieleds in data are mapped to Aesthetics.
	Aes AesMapping

	// Layers contains all the layers displayed in the plot.
	Layers []*Layer

	// Scales contains the scales for this panel. Normaly all panels
	// share all scales, but x and y might be free on faceted plots.
	Scales map[string]*Scale

	// The viewport this panel will be draw to.
	Viewport Viewport

	// Grobs contains the non-layer grobs like panel background
	// and grid lines.
	Grobs []Grob

	// Top, Right, Buttom and Left viewports and grobs.
	// Used for axis and strips.
	// TODO: a bit ugly...
	Tvp, Rvp, Bvp, Lvp Viewport
	Tgr, Rgr, Bgr, Lgr []Grob
}

// Facetting describes the facetting to use. The zero value indicates
// no facetting.
type Faceting struct {
	// Columns and Rows are the faceting specification. Each may be a
	// field in the Data. An empty string means no faceting in this
	// dimension.
	Columns, Rows string

	// Totals controlls display of row and column totals.
	Totals bool // TODO: fancier control needed

	// FreeScale determines which scales are free i.e. not shared
	// between rows and/or columns:
	//     ""    all scales shared
	//     "x"   x is free (might be different on each column)
	//     "y"   y is free (might be different on each row)
	//     "xy"  both x and y are free
	// (This is different from ggplot2. Here each row will share a common
	// x-sclae and each row will share a common y-scale.)
	FreeScale string

	// FreeSpace determines which dimension of a panel has fixed
	// size and which are free on a per row and/or column base.
	// Arguments like FreeScale.  BUG: Currently not implemented.
	FreeSpace string

	// ColStrips and RowStrips contain the strip labels.
	// TODO: decide how to set manually.
	ColStrips, RowStrips []string
}

// Fundamental is a simple geometrical object.
type Fundamental struct {
	// Geom is one if the few fundamental Geoms.
	Geom Geom

	// Data is the data for the fundamental geom.
	Data *DataFrame
}

// -------------------------------------------------------------------------
// Step 2: Preparing Data and Scales

// PrepareData is the first step in generating a plot.
// After preparing the data frame the following holds:
//   - Layer has a own data frame (maybe a copy of plots data frame).
//   - This data frame has no unused (aka not mapped to aesthetics)
//     columns.
//   - The columns name are the aestectics (e.g. x, y, size, color...)
//   - The columns have been transformed according to the
//     ScaleTransform associated with x, y, size, ....
//
// TODO: how about grouping? 69b0d2b contains grouping code.
//
// Step 2 in design.
func (panel *Panel) PrepareData() {
	fmt.Printf("Panel %q: PrepareData()\n", panel.Name)

	for i, layer := range panel.Layers {
		fmt.Printf("  Layer %d %q: PrepareData()\n", i, layer.Name)
		// Step 2a

		// Set up data and aestetics mapping.
		if layer.Data == nil {
			layer.Data = layer.Panel.Data.Copy()
		}
		aes := MergeAes(layer.DataMapping, layer.Panel.Plot.Aes)

		// Drop all unused (unmapped) fields in the data frame.
		_, fields := aes.Used(false)
		for _, f := range layer.Data.FieldNames() {
			found := false
			for _, ss := range fields {
				if f == ss {
					found = true
					break
				}
			}
			if found {
				continue
			}

			delete(layer.Data.Columns, f)
		}

		// Rename mapped fields to their aestethic name
		for a, f := range aes {
			layer.Data.Rename(f, a)
		}

		// Step 2b
		layer.Panel.Plot.PrepareScales(layer.Data, aes)

		for a := range aes {
			scale, ok := panel.Scales[a]
			if !ok {
				continue
			}
			scale.Train(layer.Data.Columns[a])
		}
	}
}

// PrepareScales makes sure plot contains all scales needed for the
// aesthetics in aes, the data is scale transformed if requested by the
// scale and the scales are pre-trained.
//
// Only continuous, non-time-scales can be transformed.
//
// Step 2b
func (plot *Plot) PrepareScales(data *DataFrame, aes AesMapping) {
	scaleable := map[string]bool{
		"x":        true,
		"y":        true,
		"color":    true,
		"fill":     true,
		"alpha":    true,
		"size":     true,
		"linetype": true,
		"shape":    true,
	}

	for a := range aes {
		if !scaleable[a] {
			fmt.Printf("    PrepareScales() %q is un-scalable\n", a)
			continue
		}

		plotScale, plotOk := plot.Scales[a]
		panelOk := false
		if len(plot.Panels) > 0 {
			_, panelOk = plot.Panels[0][0].Scales[a]
		}
		switch {
		case plotOk && panelOk:
			// Scale exists and has been distributet to the panels
			// already.
			fmt.Printf("    PrepareScales() %q already distributed\n", a)
		case plotOk && !panelOk:
			// Must be a user set scale on plot; just distribute.
			fmt.Printf("    PrepareScales() %q distributed from plot\n", a)
			plot.distributeScale(plotScale, a)
		case !plotOk && !panelOk:
			// Auto-generated scale, first occurence of this scale.
			name, typ := aes[a], data.Columns[a].Type
			fmt.Printf("    PrepareScales() %q create new and distribute\n", a)
			plotScale = NewScale(a, name, typ)
			plot.Scales[a] = plotScale
			plot.distributeScale(plotScale, a)
		case !plotOk && panelOk:
			panic("This should never happen.")
		}

		// Transform data if scale request such a transform.
		if plotScale.Transform != nil && plotScale.Transform != &IdentityScale {
			// TODO: This test should happen much earlier.
			if plotScale.Discrete || plotScale.Time {
				plot.Warnf("Cannot transform discrete or time scale %s %q",
					plotScale.Aesthetic, plotScale.Name)
				plotScale.Transform = &IdentityScale
			} else {
				field := data.Columns[a]
				field.Apply(plotScale.Transform.Trans)
				fmt.Printf("    Transformed data on scale %v", a)
			}
		}

		/***********************
				// Pre-train scales on all panels
				// TODO: This is wrong, or?  Training is per panel!
				for r := range plot.Panels {
					for c := range plot.Panels[r] {
						scale := plot.Panels[r][c].Scales[a]
						scale.Train(data.Columns[a])
						fmt.Printf("After training Scale %s on panel %s: [ %.2f, %.2f ]\n", a, plot.Panels[r][c].Name, scale.DomainMin, scale.DomainMax)
					}
				}
		                ***********************/
	}

}

// distributeScale will distribute scale to all panels. Most of the time
// all panels share one instance of a scale. But x- and y-scales may be free
// between rows and columns in which the panels recieve a copy.
func (plot *Plot) distributeScale(scale *Scale, aes string) {
	sharing := plot.scaleSharing(aes)
	switch sharing {
	case "all-panels":
		// All panels share the same scale.
		for r := range plot.Panels {
			for c := range plot.Panels[r] {
				plot.Panels[r][c].Scales[aes] = scale
			}
		}
	case "row-shared":
		// Each column share an individual copy of the scale.
		for r := range plot.Panels {
			cpy := *scale
			for c := range plot.Panels[r] {
				plot.Panels[r][c].Scales[aes] = &cpy
			}
		}
	case "col-shared":
		// Add appropriate scale to all panels.
		for c := range plot.Panels[0] {
			cpy := *scale
			for r := range plot.Panels {
				plot.Panels[r][c].Scales[aes] = &cpy
			}
		}
	default:
		panic("Ooops " + plot.scaleSharing(aes))
	}
}

// scaleSharing determines which panels share the sclae for the aesthetic aes:
// Only x and y can be not-shared in a feceted plot with set FreeScale.
func (plot *Plot) scaleSharing(aes string) string {
	if aes != "x" && aes != "y" {
		return "all-panels"
	}

	free := plot.Faceting.FreeScale

	if free == "" {
		return "all-panels"
	}

	if aes == "x" && strings.Index(free, "x") != -1 {
		// Scale for x axis, but x is 'free' i.e. each column may have
		// its own x-scale, but this one is shared along the whole
		// column.
		return "col-shared"
	}

	if aes == "y" && strings.Index(free, "y") != -1 {
		return "row-shared"
	}

	return "all-panels"
}

// -------------------------------------------------------------------------
// Step 3: Satistical Transformation

func (p *Panel) ComputeStatistics() {
	fmt.Printf("Panel %q: ComputeStatistics()\n", p.Name)

	for _, layer := range p.Layers {
		layer.ComputeStatistics()
	}
}

// ComputeStatistics computes the statistical transform. Might be the identity.
//
// Step 3 in design.
func (layer *Layer) ComputeStatistics() {
	if layer.Stat == nil {
		fmt.Printf("  Layer %q: ComputeStatistics() nil stat\n", layer.Name)
		return // The identity statistical transformation.
	}

	// Make sure all needed aesthetics (columns) are present in
	// our data frame.
	// Step 3a.
	needed := layer.Stat.Info().NeededAes
	for _, aes := range needed {
		if _, ok := layer.Data.Columns[aes]; !ok {
			layer.Panel.Plot.Warnf("Stat %s in Layer %s needs column %s",
				layer.Stat.Name(), layer.Name, aes)
			// TODO: more cleanup?
			layer.Geom = nil // Don't draw anything.
			return
		}
	}

	// Handling of excess fields. TODO: Massive refactoring needed.
	usedByStat := NewStringSetFrom(needed)
	usedByStat.Join(NewStringSetFrom(layer.Stat.Info().OptionalAes))
	additionalFields := NewStringSetFrom(layer.Data.FieldNames())
	additionalFields.Remove(usedByStat)

	// TODO: all handling related code
	/***********************************************************

		// Make sure all excess fields are discrete and abort on
		// any continuous field.
		for _, f := range fields.Elements() {
			if !layer.Data.Columns[f].Discrete() {
				layer.Plot.Warnf("Stat %s in Layer %s cannot cope with continous excess fields %s",
					layer.Stat.Name(), layer.Name, f)
				// TODO: more cleanup?
				layer.Geom = nil // Don't draw anything.
				return
			}
		}
	        *************************************************************/

	before := fmt.Sprintf("%s %d %v", layer.Stat.Name(), layer.Data.N, layer.Data.FieldNames())

	handling := layer.Stat.Info().ExtraFieldHandling
	switch handling {
	case FailOnExtraFields:
		if len(additionalFields) > 0 {
			layer.Panel.Plot.Warnf(
				"Cannot apply stat %s to %s on layer %s: additional fields %v present",
				layer.Stat.Name(), layer.Data.Name, layer.Name, additionalFields)
		}
		layer.Data = nil
	case IgnoreExtraFields:
		// Do simple transform. Step 3.
		layer.Data = layer.Stat.Apply(layer.Data, layer.Panel)
	case GroupOnExtraFields:
		// Do the transform recursively. Step 3b
		layer.Data = applyRec(layer.Data, layer.Stat, layer.Panel, additionalFields.Elements())
	}

	if layer.Data != nil {
		fmt.Printf("  Layer %q: ComputeStatistics() %s --> %d %v\n",
			layer.Name, before, layer.Data.N, layer.Data.FieldNames())
	} else {
		fmt.Printf("  Layer %q: ComputeStatistics() %s --> nil\n",
			layer.Name, before)
	}
}

// Recursively partition data on the the additional fields, apply stat and
// combine the result.
func applyRec(data *DataFrame, stat Stat, p *Panel, additionalFields []string) *DataFrame {
	if len(additionalFields) == 0 {
		return stat.Apply(data, p)
	}

	field := additionalFields[0]
	var result *DataFrame
	levels := Levels(data, field).Elements()
	partition := Partition(data, field, levels)
	for i, part := range partition {
		// Recursion
		part = applyRec(part, stat, p, additionalFields[1:])

		// Re-add the field which was stripped during partitioning.
		af := data.Columns[field].Const(levels[i], part.N)
		part.Columns[field] = af

		// Combine results.
		if i == 0 {
			result = part
		} else {
			result.Append(part)
		}
	}
	return result
}

// -------------------------------------------------------------------------
// Step 4: Wiring Result of Stat to Input of Geom

func (p *Panel) WireStatToGeom() {
	fmt.Printf("Panel %q: WireStatToGeoms()\n", p.Name)

	// A stat may return a nil data frame, e.g. if the input to the stat
	// itself was empty. In this case no wireing is needed and the layer
	// geom can be removed.

	for _, layer := range p.Layers {
		if layer.Data == nil {
			// Data was cleared by stat.
			layer.Geom = nil
			continue
		}
		layer.WireStatToGeom()
	}
}

func (layer *Layer) WireStatToGeom() {
	// Now we have a new data frame with possible new columns.
	// These may be mapped to plot aestetics by plot.StatMapping.
	// Do this now.
	if len(layer.StatMapping) != 0 {
		fmt.Printf("  Layer %q: Preparing scales with stat mapping %v\n",
			layer.Name, layer.StatMapping)

		// Rename mapped fields to their aestethic name
		for a, f := range layer.StatMapping {
			fmt.Printf("  Layer %q: Renaming %q to %q because of stat mapping.\n",
				layer.Name, f, a)
			layer.Data.Rename(f, a)
		}
		layer.Panel.Plot.PrepareScales(layer.Data, layer.StatMapping)

		for a := range layer.StatMapping {
			scale, ok := layer.Panel.Scales[a]
			if !ok {
				continue
			}
			//fmt.Printf("WireStat: Before training Scale %s on panel %s layer %s: [ %.2f, %.2f ]\n",
			//	a, layer.Panel.Name, layer.Name, scale.DomainMin, scale.DomainMax)
			scale.Train(layer.Data.Columns[a])
			// fmt.Printf("WireStat: After training Scale %s on panel %s layer %s: [ %.2f, %.2f ]\n",
			//	a, layer.Panel.Name, layer.Name, scale.DomainMin, scale.DomainMax)
		}

	}

	// TODO: Geoms should contain aesthetict only as input, so there
	// should not be a need for both, StatMapping and GeomMapping, or?

	// Rename fields produces by statistical transform to names
	// the geom understands.
	// TODO: When to set e.g. color to a certain value?
	for aes, field := range layer.GeomMapping {
		layer.Data.Rename(field, aes)
	}

}

// -------------------------------------------------------------------------
// Step 5: Constructing Geoms

// ConstructGeoms sets up the geoms so that they can be rendered. This includes
// an optional renaming of stat-generated fields to geom-understandable fields,
// applying positional adjustment to same-x geoms and reparametrization to
// fundamental geoms.
//
func (p *Panel) ConstructGeoms() {
	fmt.Printf("Panel %q: ConstructGeoms()\n", p.Name)

	for _, layer := range p.Layers {
		layer.ConstructGeoms()
	}
}

func (layer *Layer) ConstructGeoms() {
	if layer.Geom == nil {
		if layer.Data != nil {
			layer.Panel.Plot.Warnf("No Geom specified in layer %s.", layer.Name)
		}
		return
	}

	// Make sure all needed slots are present in the data frame
	slots := NewStringSetFrom(layer.Geom.NeededSlots())
	dfSlots := NewStringSetFrom(layer.Data.FieldNames())
	slots.Remove(dfSlots)
	if len(slots) > 0 {
		layer.Panel.Plot.Warnf("Missing slots in geom %s in layer %s: %v",
			layer.Geom.Name(), layer.Name, slots.Elements())
		layer.Geom = nil
		return
	}

	fmt.Printf("  Layer %q geom %q construction from %d data\n",
		layer.Name, layer.Geom.Name(), layer.Data.N)
	layer.Fundamentals = layer.Geom.Construct(layer.Data, layer.Panel)
}

// -------------------------------------------------------------------------
// Step 6: Prepare Scales
func (p *Panel) FinalizeScales() {
	fmt.Printf("Panel %q: FinalizeScales()\n", p.Name)
	for _, scale := range p.Scales {
		scale.Finalize(p.Plot.Pool)
	}
}

// -------------------------------------------------------------------------
// Step 7: Render fundamental Geoms

func (p *Panel) RenderGeoms() {
	fmt.Printf("Panel %q: RenderGeoms()\n", p.Name)
	for _, layer := range p.Layers {
		if len(layer.Fundamentals) == 0 {
			continue
		}
		for _, fund := range layer.Fundamentals {
			data := fund.Data
			aes := fund.Geom.Aes(p.Plot)
			grobs := fund.Geom.Render(p, data, aes)
			layer.Grobs = append(layer.Grobs, grobs...)
		}
	}
}

// -------------------------------------------------------------------------
// Step 8: Render remaining parts of plot

func (plot *Plot) RenderVisuals() {
	// Title, X-Label and Y-Label.
	if plot.Title != "" {
		style := MergeStyles(plot.Theme.Title, DefaultTheme.Title)
		size := String2Float(style["size"], 0, 100)
		g := GrobText{x: 0.5, y: 0.5, vjust: 0.5, hjust: 0.5,
			text: plot.Title, size: size}
		plot.Grobs["Title"] = g
		_, h := g.BoundingBox()
		plot.renderInfo["Title.Height"] = h
	}
	if name := plot.Scales["x"].Name; name != "" {
		style := MergeStyles(plot.Theme.Label, DefaultTheme.Label)
		size := String2Float(style["size"], 0, 100)
		g := GrobText{x: 0.5, y: 0.5, vjust: 0.5, hjust: 0.5,
			text: name, size: size}
		plot.Grobs["X-Label"] = g
		_, h := g.BoundingBox()
		plot.renderInfo["X-Label.Height"] = h
	}
	if name := plot.Scales["y"].Name; name != "" {
		style := MergeStyles(plot.Theme.Label, DefaultTheme.Label)
		size := String2Float(style["size"], 0, 100)
		g := GrobText{x: 0.5, y: 0.5, vjust: 0.5, hjust: 0.5, text: name,
			angle: math.Pi / 2, size: size}
		w, _ := g.BoundingBox()
		plot.renderInfo["Y-Label.Width"] = w
		plot.Grobs["Y-Label"] = g
	}

	// Strips for facetted plots.
	strip := MergeStyles(plot.Theme.Strip, DefaultTheme.Strip)
	stripBG := String2Color(strip["fill"])
	stripCol := String2Color(strip["color"])
	stripSize := String2Float(strip["size"], 4, 100)
	if len(plot.Faceting.RowStrips) > 0 {
		ncols := len(plot.Panels[0])
		maxWidth := vg.Length(0)
		for r := range plot.Panels {
			strip := plot.Faceting.RowStrips[r]
			// TDOO: Add border.
			rect := GrobRect{xmin: 0, ymin: 0, xmax: 1, ymax: 1, fill: stripBG}
			text := GrobText{x: 0.5, y: 0.5, vjust: 0.5, hjust: 0.5,
				text: strip, angle: math.Pi / 2,
				size: stripSize, color: stripCol}
			plot.Panels[r][ncols-1].Rgr = []Grob{rect, text}
			w, _ := text.BoundingBox()
			if w > maxWidth {
				maxWidth = w
			}
			fmt.Printf("Strip %q %.1f %.1f\n", strip, w, maxWidth)
		}
		plot.renderInfo["Row-Strip.Width"] = maxWidth
	}
	if len(plot.Faceting.ColStrips) > 0 {
		nrows := len(plot.Panels)
		maxHeight := vg.Length(0)
		for c := range plot.Panels[0] {
			strip := plot.Faceting.ColStrips[c]
			// TDOO: Add border.
			rect := GrobRect{xmin: 0, ymin: 0, xmax: 1, ymax: 1, fill: stripBG}
			text := GrobText{x: 0.5, y: 0.5, vjust: 0.5, hjust: 0.5,
				text: strip, angle: 0,
				size: stripSize, color: stripCol}
			plot.Panels[nrows-1][c].Tgr = []Grob{rect, text}
			_, h := text.BoundingBox()
			if h > maxHeight {
				maxHeight = h
			}
		}
		plot.renderInfo["Col-Strip.Height"] = maxHeight
	}
	plot.RenderGuides()
}

func (plot *Plot) RenderGuides() {
	maxWidth := vg.Length(0)
	yCum := vg.Length(0)
	ySep := vg.Length(5) // TODO; make configurable
	guides := GrobGroup{x0: 0, y0: 0}
	for aes, scale := range plot.Scales {
		if aes == "x" || aes == "y" {
			// X and y axes are draw on a per-panel base.
			continue
		}

		fmt.Printf("%s\n", scale.String())

		grobs, width, height := scale.Render()
		if width > maxWidth {
			maxWidth = width
		}
		gg := grobs.(GrobGroup)
		gg.y0 = float64(yCum)
		guides.elements = append(guides.elements, gg)
		yCum += height + ySep
	}
	plot.Grobs["Guides"] = guides
	plot.renderInfo["Guides.Width"] = maxWidth
}

// -------------------------------------------------------------------------
// Panel creation

// CreatePanels populates p.Panels, governed by p.Faceting.
//
// Not only p.Data is facetted but p.Layers also (if they contain own data).
func (plot *Plot) CreatePanels() {
	if plot.Faceting.Columns == "" && plot.Faceting.Rows == "" {
		plot.createSinglePanel()
	} else {
		plot.createGridPanels()
	}
}

func (plot *Plot) createSinglePanel() {
	// println("createSinglePanel()")
	panel := &Panel{
		Plot:   plot,
		Data:   plot.Data,
		Aes:    plot.Aes.Copy(),
		Layers: make([]*Layer, len(plot.Layers)),
		Scales: make(map[string]*Scale),
	}

	plot.Panels = [][]*Panel{[]*Panel{panel}}
	for i, layer := range plot.Layers {
		plot.Panels[0][0].Layers[i] = layer
		plot.Panels[0][0].Layers[i].Panel = panel
	}

	fmt.Printf("After createSinglePanel plot.Panels = %+v\n", plot.Panels)
}

func (p *Plot) createGridPanels() {
	// println("createGridPanel()")
	// Process faceting: How many facets are there, how are they named
	rows, cols := 1, 1
	var cunq []float64
	var runq []float64

	// Make sure facetting can be done and determine number of
	// rows and columns.
	if p.Faceting.Columns != "" {
		f := p.Data.Columns[p.Faceting.Columns]
		if !f.Discrete() {
			panic(fmt.Sprintf("Cannot facet over %s (type %s)",
				p.Faceting.Columns, f.Type.String()))
		}
		cunq = Levels(p.Data, p.Faceting.Columns).Elements()
		cols = len(cunq)
		p.Faceting.ColStrips = make([]string, cols)

		for c := 0; c < cols; c++ {
			p.Faceting.ColStrips[c] = f.String(cunq[c])
		}
	}
	if p.Faceting.Rows != "" {
		f := p.Data.Columns[p.Faceting.Rows]
		if !f.Discrete() {
			panic(fmt.Sprintf("Cannot facet over %s (type %s)",
				p.Faceting.Columns, f.Type.String()))
		}
		runq = Levels(p.Data, p.Faceting.Rows).Elements()
		rows = len(runq)
		p.Faceting.RowStrips = make([]string, rows)
		for r := 0; r < rows; r++ {
			p.Faceting.RowStrips[r] = f.String(runq[r])
		}
	}

	p.Panels = make([][]*Panel, rows, rows+1)
	for r := 0; r < rows; r++ {
		p.Panels[r] = make([]*Panel, cols, cols+1)
		rowData := Filter(p.Data, p.Faceting.Rows, runq[r])
		for c := 0; c < cols; c++ {
			panel := &Panel{
				Name:   fmt.Sprintf("%d/%d %s/%s", r, c, p.Faceting.RowStrips[r], p.Faceting.ColStrips[c]),
				Plot:   p,
				Scales: make(map[string]*Scale),
				Data:   Filter(rowData, p.Faceting.Columns, cunq[c]),
			}
			for _, orig := range p.Layers {
				// Copy plot layers to panel, make sure layer data is filtered.
				layer := &Layer{
					Panel:       panel,
					Name:        orig.Name,
					Stat:        orig.Stat,
					Geom:        orig.Geom,
					DataMapping: orig.DataMapping,
					StatMapping: orig.StatMapping,
					GeomMapping: orig.GeomMapping,
				}
				if orig.Data != nil {
					layer.Data = Filter(orig.Data, p.Faceting.Rows, runq[r])
					layer.Data = Filter(layer.Data, p.Faceting.Columns, cunq[c])
				}
				panel.Layers = append(panel.Layers, layer)
			}
			p.Panels[r][c] = panel

			if p.Faceting.Totals {
				// Add a total columns containing all data of this row.
				panel := &Panel{
					Name: fmt.Sprintf("%d/%d %s/-all-",
						r, c+1, p.Faceting.RowStrips[r]),
					Plot:   p,
					Data:   rowData,
					Scales: make(map[string]*Scale),
				}
				for _, layer := range p.Layers {
					if layer.Data != nil {
						layer.Data = Filter(layer.Data, p.Faceting.Rows, runq[r])
					}
					layer.Panel = panel
					panel.Layers = append(panel.Layers, layer)
				}
				p.Panels[r] = append(p.Panels[r], panel)
			}
		}
	}
	if p.Faceting.Totals {
		// Add a total row containing all column data.
		p.Panels = append(p.Panels, make([]*Panel, cols+1))
		for c := 0; c < cols; c++ {
			colData := Filter(p.Data, p.Faceting.Columns, cunq[c])
			panel := &Panel{
				Name: fmt.Sprintf("%d/%d -all-/%s",
					rows, c, p.Faceting.ColStrips[c]),
				Plot:   p,
				Data:   colData,
				Scales: make(map[string]*Scale),
			}
			for _, layer := range p.Layers {
				if layer.Data != nil {
					layer.Data = Filter(layer.Data, p.Faceting.Columns, cunq[c])
				}
				layer.Panel = panel
				panel.Layers = append(panel.Layers, layer)
			}
			p.Panels[rows][c] = panel
		}
		panel := &Panel{
			Name:   fmt.Sprintf("%d/%d -all-/-all-", rows, cols),
			Plot:   p,
			Data:   p.Data,
			Scales: make(map[string]*Scale),
		}
		for _, layer := range p.Layers {
			layer.Panel = panel
			panel.Layers = append(panel.Layers, layer)
		}
		p.Panels[rows][cols] = panel
		cols++
		rows++
	}

}

// -------------------------------------------------------------------------
// Layouting

/*

Layout and names of the viewports:

  +----------------------------------------------------------------------+
  |                                 Title                                |
  +----+---+------------+--+------------+--+------------+---+------------+
  |    |   |    Tvp     |  |    Tvp     |  |    Tvp     |   |            |
  |    +---+------------+  +------------+  +------------+---+            |
  |    | L |            |  |            |  |            | R |  Guides    |
  | Y  | v | Panel-0,1  |  | Panel-1,1  |  | Panel-2,1  | v |            |
  | -  | p |            |  |            |  |            | p |            |
  | L  +---+------------+  +------------+  +------------+---+            |
  | a  |                                                    |            |
  | b  +---+------------+  +------------+  +------------+---+            |
  | e  | L |            |  |            |  |            | R |            |
  | l  | v | Panel-0,0  |  | Panel-1,0  |  | Panel-2,0  | v |            |
  |    | p |            |  |            |  |            | p |            |
  |    +---+------------+  +------------+  +------------+---+            |
  |    |   |    Bvp     |  |    Bvp     |  |    Bvp     |   |            |
  +----+---+------------+--+------------+--+------------+---+------------+
  |    |                       X-Label                      |            |
  +----+----------------------------------------------------+------------+


*/

// Layout computes suitable viewports for the different components.
func (plot *Plot) Layout(canvas vg.Canvas, width, height vg.Length) {
	plot.Viewports = make(map[string]Viewport)

	// The basic elements: Title, axis labels and guides.

	var titleh vg.Length
	if _, ok := plot.Grobs["Title"]; ok {
		titleh = plot.renderInfo["Title.Height"]
		titleh += 2 * vg.Millimeter // TODO: make configurable
	}

	var ylabelw vg.Length
	if _, ok := plot.Grobs["Y-Label"]; ok {
		ylabelw = plot.renderInfo["Y-Label.Width"]
		ylabelw += 2 * vg.Millimeter // TODO: make configurable
	}

	var xlabelh vg.Length
	if _, ok := plot.Grobs["X-Label"]; ok {
		xlabelh = plot.renderInfo["X-Label.Height"]
		xlabelh += 2 * vg.Millimeter // TODO: make configurable
	}

	guidesSep := 2 * vg.Millimeter // TODO: make configurable
	guidesw := plot.renderInfo["Guides.Width"] + 2*guidesSep

	plot.Viewports["Title"] = Viewport{
		Canvas: canvas,
		X0:     0, Y0: height - titleh,
		Width: width, Height: titleh,
	}
	plot.Viewports["Y-Label"] = Viewport{
		Canvas: canvas,
		X0:     0, Y0: xlabelh,
		Width: ylabelw, Height: height - titleh - xlabelh,
	}
	plot.Viewports["X-Label"] = Viewport{
		Canvas: canvas,
		X0:     ylabelw, Y0: 0,
		Width: width - ylabelw - guidesw, Height: xlabelh,
	}

	plot.Viewports["Guides"] = Viewport{
		Canvas: canvas,
		X0:     width - guidesw + guidesSep, Y0: xlabelh,
		Width: guidesw, Height: height - titleh - xlabelh,
		Direct: true,
	}

	// Col- and Row-Labels, X- and Y-Tics
	var xticsh, yticsw vg.Length
	var collabh, rowlabw vg.Length
	yticsw, xticsh = plot.ticsExtents()
	collabh = plot.renderInfo["Col-Strip.Height"] + 2*vg.Millimeter // TODO: make configurabel
	rowlabw = plot.renderInfo["Row-Strip.Width"] + 2*vg.Millimeter  // TODO: make configurabel

	sepx := 2 * vg.Millimeter // TODO: make configurabel
	sepy := 2 * vg.Millimeter // TODO: make configurabel
	nrows := len(plot.Panels)
	ncols := len(plot.Panels[0])
	tw := width - ylabelw - guidesw - yticsw - rowlabw
	th := height - titleh - xlabelh - collabh - xticsh
	pwidth := (tw - sepx*vg.Length(ncols-1)) / vg.Length(ncols)
	pheight := (th - sepy*vg.Length(nrows-1)) / vg.Length(nrows)
	x0, y0 := ylabelw+yticsw, xlabelh+xticsh

	// Viewports for the panels themself.
	for r := 0; r < nrows; r++ {
		for c := 0; c < ncols; c++ {
			panelId := fmt.Sprintf("Panel-%d,%d", r, c)
			x := x0 + vg.Length(c)*(pwidth+sepx)
			y := y0 + vg.Length(r)*(pheight+sepy)
			// fmt.Printf("  %s r=%d c=%d:  x=%.2f  y=%.2f\n", panelId, r, c, x, y)
			plot.Viewports[panelId] = Viewport{
				Canvas: canvas,
				X0:     x, Y0: y,
				Width: pwidth, Height: pheight,
			}
		}
	}

	// Viewports for y-tics and row-strips.
	for r := 0; r < nrows; r++ {
		y := y0 + vg.Length(r)*(pheight+sepy)
		plot.Panels[r][0].Lvp = Viewport{
			Canvas: canvas,
			X0:     ylabelw, Y0: y,
			Width: yticsw, Height: pheight,
		}
		plot.Panels[r][ncols-1].Rvp = Viewport{
			Canvas: canvas,
			X0:     width - guidesw - rowlabw, Y0: y,
			Width: rowlabw, Height: pheight,
		}
	}

	// Viewports for x-tics and col-strips.
	for c := 0; c < ncols; c++ {
		x := x0 + vg.Length(c)*(pwidth+sepx)
		plot.Panels[0][c].Bvp = Viewport{
			Canvas: canvas,
			X0:     x, Y0: xlabelh,
			Width: pwidth, Height: xticsh,
		}
		plot.Panels[nrows-1][c].Tvp = Viewport{
			Canvas: canvas,
			X0:     x, Y0: height - titleh - collabh,
			Width: pwidth, Height: collabh,
		}
	}

}

// ticsExtents computes the width of the y-tics and the height of the x-tics
// needed to display the tics.
func (plot *Plot) ticsExtents() (ywidth, xheight vg.Length) {
	label := MergeStyles(plot.Theme.TicLabel, DefaultTheme.TicLabel)
	size := String2Float(label["size"], 4, 36)
	angle := String2Float(label["angle"], 0, 2*math.Pi) // TODO: Should be different for x and y.
	sep := vg.Length(String2Float(label["sep"], 0, 100))
	tic := MergeStyles(plot.Theme.Tic, DefaultTheme.Tic)
	length := vg.Length(String2Float(tic["length"], 0, 100))

	// Look for longest y label.
	for r := range plot.Panels {
		sy := plot.Panels[r][0].Scales["y"]
		for _, label := range sy.Labels {
			w, _ := GrobText{text: label, size: size, angle: angle}.BoundingBox()
			if w > ywidth {
				ywidth = w
			}
		}
	}
	ywidth += length + sep

	// Look for highest x label.
	for c := range plot.Panels[0] {
		sx := plot.Panels[0][c].Scales["x"]
		for _, label := range sx.Labels {
			_, h := GrobText{text: label, size: size, angle: angle}.BoundingBox()
			if h > xheight {
				xheight = h
			}
		}
	}
	xheight += length + sep

	return ywidth, xheight
}

// Draw the whole content of this panel to vp.
// show{X,Y} are used to control display of X and Y scale.
func (panel *Panel) Draw(vp Viewport, showX, showY bool) {

	// Draw strips first.
	for _, grob := range panel.Tgr {
		grob.Draw(panel.Tvp)
	}
	for _, grob := range panel.Rgr {
		fmt.Printf("Right: %s\n", grob.String())
		grob.Draw(panel.Rvp)
	}

	// Draw the panel background second.
	panelBG := MergeStyles(panel.Plot.Theme.PanelBG, DefaultTheme.PanelBG)
	// TODO: Decide how to _not_ draw something, e.g. the background?
	GrobRect{
		xmin: 0, ymin: 0,
		xmax: 1, ymax: 1,
		fill: String2Color(panelBG["fill"])}.Draw(vp)
	points := []struct{ x, y float64 }{{0, 0}, {1, 0}, {1, 1}, {0, 1}, {0, 0}}
	GrobPath{points: points,
		linetype: String2LineType(panelBG["linetype"]),
		size:     String2Float(panelBG["size"], 0, 20),
		color:    String2Color(panelBG["color"])}.Draw(vp)

	// Draw grid lines.
	sx := panel.Scales["x"]
	sy := panel.Scales["y"]
	if showX {
		fmt.Printf("\nX-Scale for panel %q:\n%s\n", panel.Name, sx.String())
	}
	if showY {
		fmt.Printf("\nY-Scale for panel %q:\n%s\n", panel.Name, sy.String())
	}

	major := MergeStyles(panel.Plot.Theme.GridMajor, DefaultTheme.GridMajor)
	majorLT := String2LineType(major["linetype"])
	fmt.Println("!!!!!!!!!!!!!!!", major["linetype"], majorLT)
	majorSize := String2Float(major["size"], 0, 20)
	majorCol := String2Color(major["color"])

	tic := MergeStyles(panel.Plot.Theme.Tic, DefaultTheme.Tic)
	ticLT := String2LineType(tic["linetype"])
	ticCol := String2Color(tic["color"])
	ticLen := vg.Length(String2Float(tic["length"], 0, 1000))
	ticSize := String2Float(tic["size"], 0, 100)

	label := MergeStyles(panel.Plot.Theme.TicLabel, DefaultTheme.TicLabel)
	labelAngle := String2Float(label["angle"], 0, 2*math.Pi)
	labelCol := String2Color(label["color"])
	labelSep := vg.Length(String2Float(label["sep"], 0, 1000))
	labelSize := String2Float(label["size"], 0, 100)

	// TODO minor := MergeStyles(panel.Plot.Theme.GridMinor, DefaultTheme.GridMinor)
	for i, x := range sx.Breaks {
		xv := sx.Pos(x)
		GrobLine{x0: xv, y0: 0, x1: xv, y1: 1,
			linetype: majorLT, size: majorSize, color: majorCol}.Draw(vp)
		if !showX {
			continue
		}
		h, sep := vp.YI(vg.Length(ticLen)), vp.YI(labelSep)
		GrobLine{x0: xv, y0: 0, x1: xv, y1: -h,
			linetype: ticLT, size: ticSize, color: ticCol}.Draw(vp)
		GrobText{x: xv, y: -h - sep, hjust: 0.5, vjust: 1,
			text: sx.Labels[i], size: labelSize, angle: labelAngle,
			color: labelCol}.Draw(vp)
	}
	for i, y := range sy.Breaks {
		yv := sy.Pos(y)
		GrobLine{x0: 0, y0: yv, x1: 1, y1: yv,
			linetype: majorLT, size: majorSize, color: majorCol}.Draw(vp)
		if !showY {
			continue
		}
		w, sep := vp.XI(vg.Length(ticLen)), vp.XI(labelSep)
		GrobLine{x0: 0, y0: yv, x1: -w, y1: yv,
			linetype: ticLT, size: ticSize, color: ticCol}.Draw(vp)
		GrobText{x: -w - sep, y: yv, hjust: 1, vjust: 0.5, text: sy.Labels[i],
			size: labelSize, color: labelCol}.Draw(vp)
	}

	// Draw the layers.
	for _, layer := range panel.Layers {
		for _, g := range layer.Grobs {
			// fmt.Printf("Drawing on layer %s: %d %s\n", layer.Name, gi, g.String())
			g.Draw(vp)
		}
	}
}

// -------------------------------------------------------------------------
// Aesthetic Mapping

// AesMapping controlls the mapping of fields of a data frame to aesthetics.
// The zero value of AesMapping is the identity mapping.
//
// The following formats are used:
//     "<fieldname>"        map aesthetic to this field
//     "fixed: <value>"     set aesthetics to the given value
type AesMapping map[string]string

func (m AesMapping) Used(includeAll bool) (aes, names []string) {
	for a, n := range m {
		aes = append(aes, a)
		if includeAll || strings.Index(n, ":") == -1 {
			names = append(names, n)
		}
	}
	sort.Strings(aes)
	sort.Strings(names)
	return aes, names
}

func (m AesMapping) Copy() AesMapping {
	c := make(AesMapping, len(m))
	for a, n := range m {
		c[a] = n
	}
	return c
}

// Merge merges set values in all the ams into m and returns the merged mapping.
func MergeAes(ams ...AesMapping) AesMapping {
	merged := MergeStyles(ams...)
	for k, v := range merged {
		if v == "" {
			delete(merged, k)
		}
	}
	return merged
}

// Combine merges set values in all the ams into m and returns the merged mapping.
// Later values in ams overwrite earlier ones or values in m.
func (m AesMapping) Combine(ams ...AesMapping) AesMapping {
	merged := m.Copy()
	for _, am := range ams {
		for aes, fname := range am {
			merged[aes] = fname
		}
	}
	return merged
}

func (plot *Plot) Check() {
	fmt.Printf("=============== Check of plot ================\n")
	for r := range plot.Panels {
		for c := range plot.Panels[r] {
			panel := plot.Panels[r][c]
			if panel.Plot != plot {
				fmt.Printf("Panel %d,%d %q: panel.Plot=%p but plot=%p\n",
					r, c, panel.Name, panel, plot)
			}
			for i := range panel.Layers {
				layer := panel.Layers[i]
				if layer.Panel != panel {
					fmt.Printf("Panel %d,%d %q, Layer %d %q: layer.Panel=%p but panel=%p\n",
						r, c, panel.Name, i, layer.Name, layer.Panel, panel)

				}
			}
		}
	}
	fmt.Printf("---------------- check done ------------------\n")
}
