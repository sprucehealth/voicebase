package encoding

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrInvalidVersionFormat is returned by ParseVersion
var ErrInvalidVersionFormat = errors.New("encoding: Invalid version format. Should be of form X or X.Y or X.Y.Z")

// VersionComponent is a token signifying a part of a version
type VersionComponent string

// Available version components
const (
	InvalidVersionComponent VersionComponent = ""
	Major                   VersionComponent = "MAJOR"
	Minor                   VersionComponent = "MINOR"
	Patch                   VersionComponent = "PATCH"
)

// Version represents the version of something (e.g. an app)
type Version struct {
	Major int
	Minor int
	Patch int
}

// UnmarshalText implements encoding.TextUnmarshaler
func (v *Version) UnmarshalText(text []byte) error {
	ver, err := ParseVersion(string(text))
	if err != nil {
		return err
	}
	*v = *ver
	return nil
}

// MarshalText implements encoding.TextMarshaler
func (v Version) MarshalText() ([]byte, error) {
	return []byte(v.String()), nil
}

// UnmarshalJSON implements json.Unmarshaler
func (v *Version) UnmarshalJSON(data []byte) error {
	strData := string(data)
	var err error
	var ver *Version
	if len(strData) > 2 && strData[0] == '"' && strData[len(strData)-1] == '"' {
		ver, err = ParseVersion(strData[1 : len(strData)-1])
	} else {
		ver, err = ParseVersion(strData)
	}

	if err != nil {
		return err
	}
	*v = *ver

	return nil
}

// MarshalJSON implements json.Marshaler
func (v Version) MarshalJSON() ([]byte, error) {
	return []byte(`"` + v.String() + `"`), nil
}

// ParseVersion attempts to parse a string as a version returning
// ErrInvalidVersionFormat if the string does not represent a valid version.
func ParseVersion(v string) (*Version, error) {
	version := &Version{}
	var err error

	// identify the seperator
	var sep rune
	var index int
	if index = strings.IndexByte(v, '.'); index >= 0 {
		sep = '.'
	} else if index = strings.IndexByte(v, '-'); index >= 0 {
		sep = '-'
	}

	// if there is no valid seperator then the whole string should be a number
	if index < 0 {
		version.Major, err = strconv.Atoi(v)
		if err != nil {
			return nil, ErrInvalidVersionFormat
		}
		return version, nil
	}

	version.Major, err = strconv.Atoi(v[:index])
	if err != nil {
		return nil, ErrInvalidVersionFormat
	}

	if index+1 >= len(v) {
		return nil, ErrInvalidVersionFormat
	}

	// check if there is another valid seperator
	v = v[index+1:]
	index = strings.IndexByte(v, byte(sep))
	if index > 0 {
		version.Minor, err = strconv.Atoi(v[:index])
		if err != nil {
			return nil, ErrInvalidVersionFormat
		}

		if index+1 >= len(v) {
			return nil, ErrInvalidVersionFormat
		}

		version.Patch, err = strconv.Atoi(v[index+1:])
		if err != nil {
			return nil, ErrInvalidVersionFormat
		}
	} else {
		version.Minor, err = strconv.Atoi(v[index+1:])
		if err != nil {
			return nil, ErrInvalidVersionFormat
		}
	}

	return version, nil
}

// String implements fmt.Stringer
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// IsZero returns true iff all parts of the version are 0
func (v Version) IsZero() bool {
	return v.Major == 0 && v.Minor == 0 && v.Patch == 0
}

// LessThan returns true iff the version is less than other. If either
// version is nil then it returns false.
func (v *Version) LessThan(other *Version) bool {
	if v == nil || other == nil {
		return false
	}

	if v.Major < other.Major {
		return true
	} else if v.Major > other.Major {
		return false
	}

	if v.Minor < other.Minor {
		return true
	} else if v.Minor > other.Minor {
		return false
	}

	if v.Patch < other.Patch {
		return true
	}

	return false
}

// GreaterThanOrEqualTo returns true iff the version is greater than or equal to other.
// If either version is nil then it returns true.
func (v *Version) GreaterThanOrEqualTo(other *Version) bool {
	return !v.LessThan(other)
}

// Equals returns true iff the versions are equal. If either version is nil then
// it returns false.
func (v *Version) Equals(other *Version) bool {
	if v == nil || other == nil {
		return false
	}

	return v.Major == other.Major &&
		v.Minor == other.Minor &&
		v.Patch == other.Patch
}

// VersionRange is the minimum and maximum versions of an app that supports a feature.
// Minimum is include, maximum is exclusive.
type VersionRange struct {
	MinVersion *Version
	MaxVersion *Version
}

// Contains returns true iff the provided version is within the range [min, max)
func (vr VersionRange) Contains(v *Version) bool {
	if vr.MinVersion != nil && !v.GreaterThanOrEqualTo(vr.MinVersion) {
		return false
	}
	if vr.MaxVersion != nil && !v.LessThan(vr.MaxVersion) {
		return false
	}
	return true
}
