package features

import (
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestMapSet(t *testing.T) {
	ms := MapSet(nil)
	test.Equals(t, false, ms.Has("foo"))
	ms = MapSet(map[string]struct{}{"foo": struct{}{}})
	test.Equals(t, true, ms.Has("foo"))
	test.Equals(t, false, ms.Has("bar"))
}

func TestLazySet(t *testing.T) {
	ms := MapSet(map[string]struct{}{"foo": struct{}{}})

	x := false
	ls := LazySet(func() Set {
		x = true
		return ms
	})
	test.Equals(t, true, ls.Has("foo"))
	test.Equals(t, true, x)

	x = false
	ls = LazySet(func() Set {
		x = true
		return ms
	})
	test.Equals(t, []string{"foo"}, ls.Enumerate())
	test.Equals(t, true, x)
}
