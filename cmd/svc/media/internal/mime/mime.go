package mime

import (
	"errors"
	"strings"
)

// TODO: Perhaps libify this

// Type represents the type information associated with data
type Type struct {
	Type    string
	Subtype string
}

// ErrMalformedType is returned if the type string is not well formed
var ErrMalformedType = errors.New("Type not well formed")

// ParseType parses out the components of a mimetype
func ParseType(t string) (*Type, error) {
	if t == "" {
		return nil, ErrMalformedType
	}
	ts := strings.Split(t, "/")
	if len(ts) != 2 {
		return nil, ErrMalformedType
	}
	return &Type{
		Type:    ts[0],
		Subtype: ts[1],
	}, nil
}

func (t *Type) String() string {
	return t.Type + "/" + t.Subtype
}
