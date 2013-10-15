package plot

import (
	"testing"
)

func TestFloatSet(t *testing.T) {
	a := NewFloatSet()
	if !a.Equals(nil) {
		t.Errorf("Got a = %v", a)
	}
	a.Add(17)
	a.Add(-2)
	if !a.Equals([]float64{-2, 17}) {
		t.Errorf("Got a = %v", a)
	}

	b := NewFloatSet()
	b.Add(17)
	b.Add(0)
	b.Add(99)
	if !b.Equals([]float64{0, 17, 99}) {
		t.Errorf("Got b = %v", a)
	}
	a.Join(b)
	if !a.Equals([]float64{-2, 0, 17, 99}) {
		t.Errorf("Got a = %v", a)
	}

	c := NewFloatSet()
	c.Add(-10)
	c.Add(0)
	c.Add(17)
	if !c.Equals([]float64{-10, 0, 17}) {
		t.Errorf("Got c = %v", c)
	}

	d := a.Intersect(c)
	if !d.Equals([]float64{0, 17}) || len(d) != 2 {
		t.Errorf("Got d = %v", d)
	}

	if d.Contains(-10) {
		t.Errorf("d contains -10")
	}
	if d.Contains(3) {
		t.Errorf("d contains 3")
	}
	if !d.Contains(0) {
		t.Errorf("d dosn't contains 0")
	}
	if !d.Contains(17) {
		t.Errorf("d dosn't contains 17")
	}

	a.Del(99)
	if !a.Equals([]float64{-2, 0, 17}) {
		t.Errorf("Got a = %v", a)
	}
	a.Del(0)
	if !a.Equals([]float64{-2, 17}) {
		t.Errorf("Got a = %v", a)
	}
	elem := a.Elements()
	if len(elem) != 2 || elem[0] != -2 || elem[1] != 17 {
		t.Errorf("Got elem = %v", elem)
	}

}

func TestStringSet(t *testing.T) {
	a := NewStringSet()
	a.Add("cat")
	a.Add("dog")
	a.Add("fish")
	a.Add("dog")
	if len(a) != 3 || !a.Equals([]string{"cat", "dog", "fish"}) {
		t.Errorf("Got a = %v", a)
	}

	a.Join(a)
	if len(a) != 3 || !a.Equals([]string{"cat", "dog", "fish"}) {
		t.Errorf("Got a = %v", a)
	}
	b := a.Intersect(a)
	if len(b) != 3 || !b.Equals([]string{"cat", "dog", "fish"}) {
		t.Errorf("Got b = %v", b)
	}
	a.Remove(a)
	if len(a) != 0 || len(a.Elements()) != 0 {
		t.Errorf("Got a = %v", a)
	}
}
