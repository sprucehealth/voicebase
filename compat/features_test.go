package compat

import (
	"sort"
	"testing"

	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/test"
)

func TestFeatures(t *testing.T) {
	reg := Features{}
	test.Equals(t, false, reg.Supported("abc", "iOS", &encoding.Version{Major: 5, Minor: 5, Patch: 0}))
	reg.Register([]*Feature{
		{
			Name: "abc",
			AppVersions: map[string]encoding.VersionRange{
				"iOS":     {MinVersion: &encoding.Version{Major: 1, Minor: 0, Patch: 0}},
				"Android": {MinVersion: &encoding.Version{Major: 0, Minor: 9, Patch: 0}},
			},
		},
		{
			Name: "xyz",
			AppVersions: map[string]encoding.VersionRange{
				"iOS":     {MinVersion: &encoding.Version{Major: 1, Minor: 5, Patch: 0}},
				"Android": {MinVersion: &encoding.Version{Major: 2, Minor: 0, Patch: 0}},
			},
		},
	})
	test.Equals(t, false, reg.Supported("abc", "iOS", &encoding.Version{Major: 0, Minor: 5, Patch: 0}))
	test.Equals(t, true, reg.Supported("abc", "iOS", &encoding.Version{Major: 1, Minor: 0, Patch: 0}))
	test.Equals(t, true, reg.Supported("xyz", "Android", &encoding.Version{Major: 2, Minor: 0, Patch: 1}))
	test.Equals(t, []string{}, reg.Set("iOS", &encoding.Version{Major: 0, Minor: 5, Patch: 0}).Enumerate())
	test.Equals(t, []string{"abc"}, reg.Set("iOS", &encoding.Version{Major: 1, Minor: 2, Patch: 0}).Enumerate())
	test.Equals(t, true, reg.Set("iOS", &encoding.Version{Major: 1, Minor: 2, Patch: 0}).Has("abc"))
	s := reg.Set("iOS", &encoding.Version{Major: 3, Minor: 0, Patch: 0})
	sl := s.Enumerate()
	sort.Strings(sl)
	test.Equals(t, []string{"abc", "xyz"}, sl)
	test.Equals(t, true, s.Has("abc"))
	test.Equals(t, true, s.Has("xyz"))
	test.Equals(t, false, s.Has("123"))
}
