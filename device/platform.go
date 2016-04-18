package device

import (
	"fmt"
	"strings"
)

type Platform string

const (
	Android Platform = "android"
	IOS     Platform = "iOS"
)

func (p Platform) String() string {
	return string(p)
}

// ParsePlatform matches the provided string against known platform types.
// It returns an error if no matches found.
func ParsePlatform(p string) (Platform, error) {
	switch strings.ToLower(p) {
	case "android":
		return Android, nil
	case "ios":
		return IOS, nil
	}
	return "", fmt.Errorf("Unable to determine platform type from %s", p)
}

// UnmarshalText implements encoding.TextUnmarshaler
func (p *Platform) UnmarshalText(text []byte) error {
	var err error
	*p, err = ParsePlatform(string(text))
	return err
}

// Scan implements sql.Scanner
func (p *Platform) Scan(src interface{}) error {
	str, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("Cannot scan type %T into Platform when string expected", src)
	}

	var err error
	*p, err = ParsePlatform(string(str))
	return err
}
