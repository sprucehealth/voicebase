package features

import "sync"

// Set is a feature flag set
type Set interface {
	// Has returns true iff the feature is in the set
	Has(feature string) bool
	// Enumerate returns a list of all the features in the set
	Enumerate() []string
}

// MapSet is a set of features flags stored as a map
type MapSet map[string]struct{}

// Has implements Set.Has
func (ms MapSet) Has(feature string) bool {
	_, ok := ms[feature]
	return ok
}

// Enumerate implements Set.Enumerate
func (ms MapSet) Enumerate() []string {
	sl := make([]string, 0, len(ms))
	for n := range ms {
		sl = append(sl, n)
	}
	return sl
}

type nullSet struct{}

func (nullSet) Has(string) bool     { return false }
func (nullSet) Enumerate() []string { return nil }

type lazySet struct {
	o sync.Once
	f func() Set
	s Set
}

// LazySet returns a set that can lazily create another set
func LazySet(fn func() Set) Set {
	return &lazySet{
		f: fn,
	}
}

func (ls *lazySet) Has(feature string) bool {
	return ls.set().Has(feature)
}

func (ls *lazySet) Enumerate() []string {
	return ls.set().Enumerate()
}

func (ls *lazySet) set() Set {
	ls.o.Do(func() {
		ls.s = ls.f()
	})
	return ls.s
}
