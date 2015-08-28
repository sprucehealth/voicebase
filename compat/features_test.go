package compat

import (
	"sort"
	"testing"

	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/test"
)

func TestFeatures(t *testing.T) {
	reg := Features{}
	test.Equals(t, false, reg.Supported("abc", "iOS", &encoding.Version{5, 5, 0}))
	reg.Register([]*Feature{
		{
			Name: "abc",
			AppVersions: map[string]encoding.VersionRange{
				"iOS":     {MinVersion: &encoding.Version{1, 0, 0}},
				"Android": {MinVersion: &encoding.Version{0, 9, 0}},
			},
		},
		{
			Name: "xyz",
			AppVersions: map[string]encoding.VersionRange{
				"iOS":     {MinVersion: &encoding.Version{1, 5, 0}},
				"Android": {MinVersion: &encoding.Version{2, 0, 0}},
			},
		},
	})
	test.Equals(t, false, reg.Supported("abc", "iOS", &encoding.Version{0, 5, 0}))
	test.Equals(t, true, reg.Supported("abc", "iOS", &encoding.Version{1, 0, 0}))
	test.Equals(t, true, reg.Supported("xyz", "Android", &encoding.Version{2, 0, 1}))
	test.Equals(t, []string{}, reg.Set("iOS", &encoding.Version{0, 5, 0}).Enumerate())
	test.Equals(t, []string{"abc"}, reg.Set("iOS", &encoding.Version{1, 2, 0}).Enumerate())
	test.Equals(t, true, reg.Set("iOS", &encoding.Version{1, 2, 0}).Has("abc"))
	s := reg.Set("iOS", &encoding.Version{3, 0, 0})
	sl := s.Enumerate()
	sort.Strings(sl)
	test.Equals(t, []string{"abc", "xyz"}, sl)
	test.Equals(t, true, s.Has("abc"))
	test.Equals(t, true, s.Has("xyz"))
	test.Equals(t, false, s.Has("123"))
}
