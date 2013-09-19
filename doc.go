// Plot provides sientific plots in the style of R's ggplot2.
//
//
// Data Representation: Data Frames
//
// Data can be represented in two different ways: Either as "slice of
// measurements" or as "collection of slices".
//
// "Slice of measurements" are of the following style
//      var DataSOM []Measurement
//      type Measurement struct {
//          Height float64
//          Weigth float64
//          Age    int
//      }
//
// "Collection of slices" are structured like this:
//      var DataCOS Measurements
//      type Measurements struct {
//          Height[] float64
//          Weigth[] float64
//          Age[]    int
//      }
//
// TODO: function types
//
//
// Types of Data Elements
//
// Internaly plot uses the following Go types:
//     float64    for continous data
//     int64      for discrete data
//     string     for discrete data
//     time.Time  for time data
// Other types may be used in data frames but will be converted to one
// of those above. These may lead to overflow when uint64 is used.
// Runes are converted to string.
//
//
// Calculated Values
//
// Your data frame need not contain all data you want to plot as a field.
// By providing appropiate methods on your data frame you can have values
// computed. For slice of measuremennts style data frames just provide
// a method without parameters; in the collection of slices style the
// method takes the index as parameter:
//    func(m Measurement) BMI() float64 { return m.Weight / (m.Height * m.Height) }
//    func(m Measurements) BMI(i int) float64 { return m[i].Weight / (m[i].Height * m[i].Height) }
//
//
//
//
package plot
