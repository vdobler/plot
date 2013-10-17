package plot

import (
	"fmt"
	"image/color"
	"math"
	"sort"
	"strings"
	"time"
)

var now = time.Now
var col = color.RGBA{}
var floor = math.Floor

type Plot struct {
	// Data is the data to draw.
	Data *DataFrame

	// Faceting describes the Used Faceting
	Faceting Faceting

	// Mapping describes how fileds in data are mapped to Aesthetics
	Aes AesMapping

	// Layers contains all the layers displayed in the plot.
	Layers []*Layer

	// Panels are the different panels for faceting
	Panels [][]Panel

	Scales map[string]*Scale

	Theme Theme
}

// Layer represents one layer of data
//
type Layer struct {
	Plot *Plot
	Name string

	// A nil Data will use the Data from the plot this Layer belongs to.
	Data        *DataFrame
	DataMapping AesMapping

	// Stat is the statistical transformation used in this layer.
	Stat        Stat
	StatMapping AesMapping

	// Geom is the geom to use for this layer
	Geom        Geom
	GeomMapping AesMapping

	Position PositionAdjust

	Grobs []Grob
}

func (p *Plot) Warnf(f string, args ...interface{}) {
	if !strings.HasSuffix(f, "\n") {
		f = f + "\n"
	}
	fmt.Printf("Warning "+f, args...)
}

func contains(s []string, t string) bool {
	for _, ss := range s {
		if t == ss {
			return true
		}
	}
	return false
}

func same(s []string, t []string) bool {
	if len(s) != len(t) {
		return false
	}
	for _, x := range s {
		if !contains(t, x) {
			return false
		}
	}
	return true
}

// PrepareData is the first step in generating a plot.
// After preparing the data frame the following holds
//   - Layer has a own data frame (maybe a copy of plots data frame)
//   - This data frame has no unused (aka not mapped to aesthetics)
//     columns
//   - The columns name are the aestectics (e.g. x, y, size, color...)
//   - The columns have been transformed according to the
//     ScaleTransform associated with x, y, size, ....
//
// TODO: how about grouping? 69b0d2b contains grouping code.
func (p *Plot) PrepareData() {
	for _, layer := range p.Layers {
		// Set up data and aestetics mapping.
		if layer.Data == nil {
			layer.Data = layer.Plot.Data.Copy()
		}
		aes := MergeAes(layer.DataMapping, layer.Plot.Aes)

		// Drop all unused (unmapped) fields in the data frame.
		_, fields := aes.Used(false)
		for _, f := range layer.Data.FieldNames() {
			if contains(fields, f) {
				continue
			}
			delete(layer.Data.Columns, f)
		}

		// Rename mapped fields to their aestethic name
		for a, f := range aes {
			layer.Data.Rename(f, a)
		}

		layer.Plot.PrepareScales(layer.Data, aes)
	}
}

// PrepareScales makes sure plot contains all sclaes needed for the
// aesthetics in aes, the data is scale transformed if requested by the
// scale and the scales are pre-trained.
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
		println("PrepareScales working on ", a)
		if !scaleable[a] {
			println("Un-scalable scale ", a)
			continue
		}

		scale, ok := plot.Scales[a]

		// Add scale for these aesthetics if not jet set up.
		if !ok {
			// Add appropriate scale.
			scale = NewScale(a, data.Columns[a])
			plot.Scales[a] = scale
			println("Added new scale ", a)
		} else {
			println("Scale ", a, " exists with name ", scale.Type)
		}

		// Transform scales if needed.
		if scale.Transform != nil {
			field := data.Columns[a]
			field.Apply(scale.Transform.Trans)
			println("Transform data on scale ", a)
		}
		// Pre-train scales
		println("Pretraining ", a, " ", scale.Type)
		scale.Train(data.Columns[a])
	}

}

// ComputeStatistics computes the statistical transform. Might be the identity.
func (layer *Layer) ComputeStatistics() {
	if layer.Stat == nil {
		return // The identity statistical transformation.
	}

	// Make sure all needed aesthetics (columns) are present in
	// our data frame.
	needed := layer.Stat.NeededAes()
	for _, aes := range needed {
		if _, ok := layer.Data.Columns[aes]; !ok {
			layer.Plot.Warnf("Stat %s in Layer %s needs column %s",
				layer.Stat.Name(), layer.Name, aes)
			// TODO: more cleanup?
			layer.Geom = nil // Don't draw anything.
			return
		}
	}

	// Handling of excess fields. TODO: Massive refactoring needed.
	usedByStat := NewStringSetFrom(needed)
	usedByStat.Join(NewStringSetFrom(layer.Stat.OptionalAes()))
	fields := NewStringSetFrom(layer.Data.FieldNames())
	fields.Remove(usedByStat)
	handling := layer.Stat.ExtraFieldHandling()
	if len(fields) == 0 || handling == IgnoreExtraFields {
		layer.Data = layer.Stat.Apply(layer.Data, layer.Plot)
	} else {
		if handling == FailOnExtraFields {
			layer.Plot.Warnf("Stat %s in Layer %s cannot cope with excess fields %v",
				layer.Stat.Name(), layer.Name, fields.Elements())
			// TODO: more cleanup?
			layer.Geom = nil // Don't draw anything.
			return
		}

		// Else, make sure all excess fields are discrete.
		for _, f := range fields.Elements() {
			if !layer.Data.Columns[f].Discrete() {
				layer.Plot.Warnf("Stat %s in Layer %s cannot cope with continous excess fields %s",
					layer.Stat.Name(), layer.Name, f)
				// TODO: more cleanup?
				layer.Geom = nil // Don't draw anything.
				return
			}
		}

		if len(fields) > 1 {
			panic("Implement me")
		}
		ef := fields.Elements()[0]
		f := layer.Data.Columns[ef]
		levels := Levels(layer.Data, ef)
		for i, level := range levels.Elements() {
			df := Filter(layer.Data, ef, level)
			delete(df.Columns, ef)
			res := layer.Stat.Apply(df, layer.Plot)
			res.Columns[ef] = f.Const(level, res.N)
			if i == 0 {
				layer.Data = res
			} else {
				layer.Data.Append(res)
			}
		}

	}

	// Now we have a new data frame with possible new columns.
	// These may be mapped to plot aestetics by plot.StatMapping.
	// Do this now.
	if len(layer.StatMapping) == 0 {
		return
		// TODO: this also misses training the scales...
	}

	fmt.Printf("Layer %s: preparing scales with stat mapping %v\n",
		layer.Name, layer.StatMapping)

	// Rename mapped fields to their aestethic name
	for a, f := range layer.StatMapping {
		layer.Data.Rename(f, a)
		println("Renaming ", f, " to ", a, " because of stat mapping.")
	}
	layer.Plot.PrepareScales(layer.Data, layer.StatMapping)
}

func (p *Plot) ComputeStatistics() {
	for _, layer := range p.Layers {
		layer.ComputeStatistics()
	}
}

// ConstructGeoms sets up the geoms so that they can be rendered. This includes
// an optional renaming of stat-generated fields to geom-understandable fields,
// applying positional adjustment to same-x geoms and reparametrization to
// fundamental geoms.
//
// TODO: Should 5a and 5b be exchanged?
func (p *Plot) ConstructGeoms() {
	for _, layer := range p.Layers {
		if layer.Geom == nil {
			layer.Plot.Warnf("No Geom specified in layer %s.", layer.Name)
			return
		}

		// Rename fields produces by statistical transform to names
		// the geom understands. (Step 4b.)
		// TODO: When to set e.g. color to a certain value?
		for aes, field := range layer.GeomMapping {
			layer.Data.Rename(field, aes)
		}

		// Make sure all needed slots are present in the data frame
		slots := NewStringSetFrom(layer.Geom.NeededSlots())
		dfSlots := NewStringSetFrom(layer.Data.FieldNames())
		slots.Remove(dfSlots)
		if len(slots) > 0 {
			layer.Plot.Warnf("Missing slots in geom %s in layer %s: %v",
				layer.Geom.Name(), layer.Name, slots.Elements())
			layer.Geom = nil
			return
		}

		// (Step 5a)
		layer.Geom.AdjustPosition(layer.Data, layer.Position)

		// (Step 5b)
		layer.Geom = layer.Geom.Reparametrize(layer.Data)
	}
}

func (p *Plot) RetrainScales() {
	for aes, scale := range p.Scales {
		for _, layer := range p.Layers {
			if layer.Geom == nil {
				println("Retrain scale: No geom on layer ", layer.Name)
			}

			scale.Retrain(aes, layer.Geom, layer.Data)
		}
		scale.Prepare()
	}
}

func (p *Plot) RenderGeoms() {
	for _, layer := range p.Layers {
		if layer.Geom != nil {
			data := layer.Data
			aes := layer.Geom.Aes(p)
			layer.Grobs = layer.Geom.Render(p, data, aes)
		}
	}
}

// Unfacetted plotting, Layers have no own data.
// TODO: maybe not func on Plot but on Panel
func (p *Plot) Simple() {
	// Make sure all layers know their parent plot / and or panel (TODO)
	for i := range p.Layers {
		p.Layers[i].Plot = p
	}

	// Prepare data: map aestetics, add scales, clean data frame and
	// apply scale transformations. Mapped scales are pre-trained.
	// (Steps 2a and 2b in design.)
	p.PrepareData()

	// The second step: Compute statistics.
	// If a layer has a statistical transform: Apply this transformation
	// to the data frame of this layer.
	// (Step 3 and 4a in design)
	p.ComputeStatistics()

	// Construct geoms
	// (Step 4b, 5a and 5b in design)
	p.ConstructGeoms()

	// Retrain scales. (Step 6)
	p.RetrainScales()

	// Render Geoms to Grobs using scales (Step7).
	p.RenderGeoms()
}

func (p *Plot) Draw() {
	p.CreatePanels()

	// p.DistributeAes()

	// Transform scales
	// Compute statistics
	// p.ComputeStatistics()

	// Map aestetics
	// Train scales
	// Render
}

// CreatePanels populates p.Panels, coverned by p.Faceting.
//
// Not only p.Data is facetted but p.Layers also (if they contain own data).
func (p *Plot) CreatePanels() {
	// Process faceting: How many facets are there, how are they named
	rows, cols := 1, 1
	var cunq []float64
	var runq []float64

	if p.Faceting.Columns != "" {
		if f := p.Data.Columns[p.Faceting.Columns]; !f.Discrete() {
			panic(fmt.Sprintf("Cannot facet over %s (type %s)", p.Faceting.Columns, f.Type.String()))
		}
		cunq = Levels(p.Data, p.Faceting.Columns).Elements()
		cols = len(cunq)
	}

	if p.Faceting.Rows != "" {
		if f := p.Data.Columns[p.Faceting.Rows]; !f.Discrete() {
			panic(fmt.Sprintf("Cannot facet over %s (type %s)", p.Faceting.Columns, f.Type.String()))
		}
		runq = Levels(p.Data, p.Faceting.Rows).Elements()
		rows = len(runq)
	}

	p.Panels = make([][]Panel, rows, rows+1)
	for r := 0; r < rows; r++ {
		p.Panels[r] = make([]Panel, cols, cols+1)
		rdf := Filter(p.Data, p.Faceting.Rows, runq[r])
		for c := 0; c < cols; c++ {
			p.Panels[r][c].Data = Filter(rdf, p.Faceting.Columns, cunq[c])
			for _, layer := range p.Layers {
				if layer.Data != nil {
					layer.Data = Filter(layer.Data, p.Faceting.Rows, runq[r])
					layer.Data = Filter(layer.Data, p.Faceting.Columns, cunq[c])
				}
				p.Panels[r][c].Layers = append(p.Panels[r][c].Layers, layer)
			}

			if p.Faceting.Totals {
				p.Panels[r] = append(p.Panels[r], Panel{Data: rdf})
				for _, layer := range p.Layers {
					if layer.Data != nil {
						layer.Data = Filter(layer.Data, p.Faceting.Rows, runq[r])
					}
					p.Panels[r][c].Layers = append(p.Panels[r][c].Layers, layer)
				}
			}
		}
	}
	if p.Faceting.Totals {
		/*
			p.Panels = append(p.Panels, make([]Panel, cols+1))
			for c := 0; c < cols; c++ {
				cdf := p.Data.Filter(p.Faceting.Columns, cunq[c])
				p.Panels[rows][c] = Panel{Data: cdf}
			}
			p.Panels[rows][cols] = Panel{Data: p.Data}
			for _, layer := range p.Layers {
				if layer.Data != nil {
					layer.Data = layer.Data.Filter(p.Faceting.Columns, cunq[c])
				}
				p.Panels[rows][cols].Layers = append(p.Panels[rows][cols].Layers, layer)
			}
		*/
		cols++
		rows++
	}
}

/*
// merge plot aes into each layer aes
func (p *Plot) DistributeAes() {
	for r := range p.Panels {
		for c := range p.Panels[r] {
			for l := range p.Panels[r][c].Layers {
				p.Panels[r][c].Layers[l].Aes = plot.Aes.Merge(p.Panels[r][c].Layers[l].Aes)
			}
		}
	}
}

func (p *Plot) ComputeStatistics() {
	for r := range p.Panels {
		for c := range p.Panels[r] {
			p.Panels[r][c].Layers = []Layer{}
			for _, layer := range p.Layers {
				if layer.Stat != nil {
					statDF := layer.Stat.Apply(layer.Data, layer.Aes)
				}
			}
		}
	}
}
*/

type Panel struct {
	Data *DataFrame

	RowName string
	ColName string

	// Plot is the plot this panel belongs to
	Plot *Plot

	Layers []*Layer
}

type Faceting struct {
	// Columns and Rows are the faceting specification. Each may be a comma
	// seperated list of fields in the Data. An empty string means no
	// faceting in this dimension.
	Columns, Rows string

	// Totals controlls display of totals.
	Totals bool // TODO: fancier control needed

	FreeScale string // "": fixed, "x": x is free, "y": y is free, "xy": both are free

	FreeSpace string // as FreeScale but for size of panel
}

func UniqueStrings(s []string) (u []string) {
	if len(s) <= 1 {
		return s
	}
	sort.Strings(s)
	t := s[0]
	for i := 1; i <= len(s); i++ {
		if s[i] == t {
			continue
		}
		t = s[i]
		u = append(u, t)
	}
	return u
}

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

type Viewport struct {
	// The underlying image

	// The rectangel of this vp

	// Functions to turn grob coordinates to pixel
}

// -------------------------------------------------------------------------
// Scale Transformations

type ScaleTransform struct {
	Trans   func(float64) float64
	Inverse func(float64) float64
	Format  func(float64, string) string
}

var Log10Scale = ScaleTransform{
	Trans:   func(x float64) float64 { return math.Log10(x) },
	Inverse: func(y float64) float64 { return math.Pow(10, y) },
	Format:  func(y float64, s string) string { return fmt.Sprintf("10^{%s}", s) },
}

var IdentityScale = ScaleTransform{
	Trans:   func(x float64) float64 { return x },
	Inverse: func(y float64) float64 { return y },
	Format:  func(y float64, s string) string { return s },
}

// -------------------------------------------------------------------------
// Position Adjustments

type PositionAdjust int

const (
	PosIdentity PositionAdjust = iota
	PosJitter
	PosStack
	PosFill
	PosDodge
)
