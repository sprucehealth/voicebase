package compat

import (
	"strings"

	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/features"
)

// Features maintains a compatibility set of features based on app version
type Features struct {
	versions map[string]map[string]encoding.VersionRange // App -> Feature -> Minimum Version
}

// Feature is the set of minimum app versions for a feature
type Feature struct {
	Name        string
	AppVersions map[string]encoding.VersionRange
}

type set struct {
	f map[string]encoding.VersionRange
	v *encoding.Version
}

// Register registers a minimum version for a feature on a set of apps.
func (f *Features) Register(feature []*Feature) {
	if f.versions == nil {
		f.versions = make(map[string]map[string]encoding.VersionRange)
	}
	for _, feat := range feature {
		for app, vr := range feat.AppVersions {
			app = strings.ToLower(app)
			appf := f.versions[app]
			if appf == nil {
				appf = make(map[string]encoding.VersionRange)
				f.versions[app] = appf
			}
			appf[feat.Name] = vr
		}
	}
}

// Supported returns true iff the feature is supported by the version of the app
func (f *Features) Supported(feature, app string, ver *encoding.Version) bool {
	appf := f.versions[strings.ToLower(app)]
	vr, ok := appf[feature]
	if !ok {
		// Default to false if feature not registered
		return false
	}
	return vr.Contains(ver)
}

// Set returns the set of available feature flags for the provided app and version
func (f *Features) Set(app string, ver *encoding.Version) features.Set {
	return &set{f: f.versions[strings.ToLower(app)], v: ver}
}

func (s *set) Has(feature string) bool {
	vr, ok := s.f[feature]
	if !ok {
		return false
	}
	return vr.Contains(s.v)
}

func (s *set) Enumerate() []string {
	feat := make([]string, 0, len(s.f))
	for feature, vr := range s.f {
		if vr.Contains(s.v) {
			feat = append(feat, feature)
		}
	}
	return feat
}
