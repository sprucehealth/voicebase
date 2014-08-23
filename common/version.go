package common

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
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

func ParseVersion(v string) (*Version, error) {
	version := &Version{}
	var err error
	invalidVersionFormat := errors.New("Invalid version format. Should be of form X or X.Y or X.Y.Z")

	indexOfFirstPeriod := strings.IndexByte(v, '.')

	// if there are no periods then the whole string should be a number
	if indexOfFirstPeriod < 0 {
		version.Major, err = strconv.Atoi(v)
		if err != nil {
			return nil, invalidVersionFormat
		}
		return version, nil
	}

	version.Major, err = strconv.Atoi(v[:indexOfFirstPeriod])
	if err != nil {
		return nil, invalidVersionFormat
	}

	if indexOfFirstPeriod+1 >= len(v) {
		return nil, invalidVersionFormat
	}

	// check if there is another period
	i2 := strings.IndexByte(v[indexOfFirstPeriod+1:], '.')
	if i2 > 0 {
		indexOfSecondPeriod := indexOfFirstPeriod + i2 + 1
		version.Minor, err = strconv.Atoi(v[indexOfFirstPeriod+1 : indexOfSecondPeriod])
		if err != nil {
			return nil, invalidVersionFormat
		}

		if indexOfSecondPeriod+1 >= len(v) {
			return nil, invalidVersionFormat
		}

		version.Patch, err = strconv.Atoi(v[indexOfSecondPeriod+1:])
		if err != nil {
			return nil, invalidVersionFormat
		}
	} else {
		version.Minor, err = strconv.Atoi(v[indexOfFirstPeriod+1:])
		if err != nil {
			return nil, invalidVersionFormat
		}
	}

	return version, nil
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
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
