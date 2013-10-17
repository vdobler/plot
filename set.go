package plot

import (
	"fmt"
	"sort"
)

// -------------------------------------------------------------------------
// Float Set

// Float set is a set of float64 values.
type FloatSet map[float64]struct{}

func NewFloatSet() FloatSet {
	return make(FloatSet)
}

func (s FloatSet) String() string {
	var t = "[ "
	for x, _ := range s {
		t += fmt.Sprintf("%.1f ", x)
	}
	return t + "]"
}

// Add adds x to s.
func (s FloatSet) Add(x float64) {
	s[x] = struct{}{}
}

// Del removes x from s.
func (s FloatSet) Del(x float64) {
	delete(s, x)
}

// Contains reports membership of x in s.
func (s FloatSet) Contains(x float64) bool {
	_, ok := s[x]
	return ok
}

// Join adds all elements of t to s.
func (s FloatSet) Join(t FloatSet) {
	for x := range t {
		s[x] = struct{}{}
	}
}

// Intersect returns the intersection of s and t.
func (s FloatSet) Intersect(t FloatSet) FloatSet {
	intersection := NewFloatSet()
	for x := range s {
		if t.Contains(x) {
			intersection.Add(x)
		}
	}
	return intersection
}

// Remove removes all elements of t from s. (Set difference)
func (s FloatSet) Remove(t FloatSet) {
	for x := range t {
		delete(s, x)
	}
}

// Equals compares s to a slice t.
func (s FloatSet) Equals(t []float64) bool {
	if len(s) != len(t) {
		return false
	}
	for _, x := range t {
		if _, ok := s[x]; !ok {
			return false
		}
	}
	return true
}

func (s FloatSet) Elements() []float64 {
	elems := make([]float64, len(s))
	i := 0
	for x := range s {
		elems[i] = x
		i++
	}
	sort.Float64s(elems)
	return elems
}

// -------------------------------------------------------------------------
// String Set

// String set is a set of string values.
type StringSet map[string]struct{}

func NewStringSet() StringSet {
	return make(StringSet)
}

func NewStringSetFrom(init []string) StringSet {
	s := NewStringSet()
	for _, v := range init {
		s.Add(v)
	}
	return s
}

// Add adds x to s.
func (s StringSet) Add(x string) {
	s[x] = struct{}{}
}

// Del removes x from s.
func (s StringSet) Del(x string) {
	delete(s, x)
}

// Contains reports membership of x in s.
func (s StringSet) Contains(x string) bool {
	_, ok := s[x]
	return ok
}

// Join adds all elements of t to s.
func (s StringSet) Join(t StringSet) {
	for x := range t {
		s[x] = struct{}{}
	}
}

// Intersect returns the intersection of s and t.
func (s StringSet) Intersect(t StringSet) StringSet {
	intersection := NewStringSet()
	for x := range s {
		if t.Contains(x) {
			intersection.Add(x)
		}
	}
	return intersection
}

// Remove removes all elements of t from s. (Set difference)
func (s StringSet) Remove(t StringSet) {
	for x := range t {
		delete(s, x)
	}
}

// Equals compares s to a slice t.
func (s StringSet) Equals(t []string) bool {
	if len(s) != len(t) {
		return false
	}
	for _, x := range t {
		if _, ok := s[x]; !ok {
			return false
		}
	}
	return true
}

func (s StringSet) Elements() []string {
	elems := make([]string, len(s))
	i := 0
	for x := range s {
		elems[i] = x
		i++
	}
	sort.Strings(elems)
	return elems
}
