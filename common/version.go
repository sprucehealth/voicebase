package common

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type VersionComponent string

const (
	InvalidVersionComponent VersionComponent = ""
	Major                   VersionComponent = "MAJOR"
	Minor                   VersionComponent = "MINOR"
	Patch                   VersionComponent = "PATCH"
)

type Version struct {
	Major int
	Minor int
	Patch int
}

func (v *Version) UnmarshalText(text []byte) error {
	ver, err := ParseVersion(string(text))
	if err != nil {
		return err
	}
	*v = *ver
	return nil
}

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

func (v Version) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, v.String())), nil
}

func ParseVersion(v string) (*Version, error) {
	version := &Version{}
	var err error
	invalidVersionFormat := errors.New("Invalid version format. Should be of form X or X.Y or X.Y.Z")

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
			return nil, invalidVersionFormat
		}
		return version, nil
	}

	version.Major, err = strconv.Atoi(v[:index])
	if err != nil {
		return nil, invalidVersionFormat
	}

	if index+1 >= len(v) {
		return nil, invalidVersionFormat
	}

	// check if there is another valid seperator
	v = v[index+1:]
	index = strings.IndexByte(v, byte(sep))
	if index > 0 {
		version.Minor, err = strconv.Atoi(v[:index])
		if err != nil {
			return nil, invalidVersionFormat
		}

		if index+1 >= len(v) {
			return nil, invalidVersionFormat
		}

		version.Patch, err = strconv.Atoi(v[index+1:])
		if err != nil {
			return nil, invalidVersionFormat
		}
	} else {
		version.Minor, err = strconv.Atoi(v[index+1:])
		if err != nil {
			return nil, invalidVersionFormat
		}
	}

	return version, nil
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v Version) IsZero() bool {
	return v.Major == 0 && v.Minor == 0 && v.Patch == 0
}

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

func (v *Version) GreaterThanOrEqualTo(other *Version) bool {
	return !v.LessThan(other)
}

func (v *Version) Equals(other *Version) bool {
	if v == nil || other == nil {
		return false
	}

	return v.Major == other.Major &&
		v.Minor == other.Minor &&
		v.Patch == other.Patch
}
