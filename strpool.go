package plot

import "sync"

type StringPool struct {
	sync.Mutex
	pool []string
}

func NewStringPool() *StringPool {
	return &StringPool{pool: make([]string, 0, 100)}
}

func (sp *StringPool) Add(s string) int {
	sp.Lock()
	defer sp.Unlock()
	if i := sp.Find(s); i != -1 {
		return i
	}
	sp.pool = append(sp.pool, s)
	return len(sp.pool) - 1
}

func (sp *StringPool) Find(s string) int {
	for i, t := range sp.pool {
		if t == s {
			return i
		}
	}
	return -1
}

func (sp *StringPool) Get(i int) string {
	if i<0 || i>=len(sp.pool) {
		return "--NA--"
	}

	return sp.pool[i]
}
