package plot

import (
	"math"
	"time"
)

type M interface{}

func Add(a, b M) M {
	switch a.(type) {
	case float64:
		return a.(float64) + b.(float64)
	case int64:
		return a.(int64) + b.(int64)
	case string:
		return a.(string) + b.(string)
	case time.Time:
		d := b.(time.Time).Sub(time.Time{})
		return a.(time.Time).Add(d)
	default:
		panic("Oooops")
	}
}

func RoundDown(a, b M) M {
	switch a.(type) {
	case float64:
		return math.Floor(a.(float64)/b.(float64)) * b.(float64)
	case int64:
		return int64(a.(int64)/b.(int64)) * b.(int64)
	default:
		panic("Ooops")
	}
}
