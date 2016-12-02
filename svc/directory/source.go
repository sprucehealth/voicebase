package directory

import (
	"strings"

	"github.com/sprucehealth/backend/libs/errors"
)

const sourceSep = ":"

// FlattenEntitySource returns the flattened key/serialized verison of a source
func FlattenEntitySource(s *EntitySource) string {
	if s == nil {
		return ""
	}
	return s.Type.String() + sourceSep + s.Data
}

// ParseEntitySource parses the string represenation of an entity source
func ParseEntitySource(s string) (*EntitySource, error) {
	ss := strings.SplitN(s, sourceSep, 2)
	if len(ss) != 2 {
		return nil, errors.Errorf("Invalid EntitySource %s", s)
	}
	tv, ok := EntitySource_SourceType_value[ss[0]]
	if !ok {
		return nil, errors.Errorf("Invalid EntitySource Type for %s", s)
	}
	return &EntitySource{
		Type: EntitySource_SourceType(tv),
		Data: ss[1],
	}, nil
}
