package encoding

import (
	"database/sql/driver"
	"fmt"
	"strconv"

	"github.com/sprucehealth/backend/errors"
)

// ObjectID is used for the (un)marshalling of data models ids, such that
// null values passed from the client can be treated as 0 values.
type ObjectID struct {
	Uint64Value uint64
	IsValid     bool
}

// NewObjectID returns an ObjectID using the provided uint64.
func NewObjectID(id uint64) ObjectID {
	objectID := ObjectID{
		Uint64Value: id,
		IsValid:     true,
	}
	return objectID
}

// DeprecatedNewObjectID returns an ObjectID using the provided int64.
// DEPRECATED: use NewObjectID instead which takes a uint64 rather than int64
func DeprecatedNewObjectID(id int64) ObjectID {
	objectID := ObjectID{
		Uint64Value: uint64(id),
		IsValid:     true,
	}
	return objectID
}

// UnmarshalJSON implements json.Unmarshaler
func (id *ObjectID) UnmarshalJSON(data []byte) error {
	strData := string(data)
	// only treating the case of an empty string or a null value
	// as value being 0.
	// otherwise relying on integer parser
	if len(strData) < 2 || strData == "null" || strData == `""` {
		*id = ObjectID{
			Uint64Value: 0,
			IsValid:     false,
		}
		return nil
	}
	intID, err := strconv.ParseUint(strData[1:len(strData)-1], 10, 64)
	*id = ObjectID{
		Uint64Value: intID,
		IsValid:     true,
	}
	return err
}

// MarshalJSON implements json.Marshaler
func (id ObjectID) MarshalJSON() ([]byte, error) {
	// don't marshal anything if value is not valid
	if !id.IsValid {
		return []byte(`null`), nil
	}

	return []byte(fmt.Sprintf(`"%d"`, id.Uint64Value)), nil
}

// MarshalText implements encoding.TextMarshaler
func (id ObjectID) MarshalText() ([]byte, error) {
	if !id.IsValid {
		return []byte{}, nil
	}
	return strconv.AppendUint(nil, id.Uint64Value, 10), nil
}

// UnmarshalText implements encoding.TextUnmarshaler
func (id *ObjectID) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		*id = ObjectID{
			Uint64Value: 0,
			IsValid:     false,
		}
		return nil
	}
	var err error
	id.Uint64Value, err = strconv.ParseUint(string(text), 10, 64)
	id.IsValid = err == nil
	return errors.Trace(err)
}

// Int64 returns an int64 version of the ID.
// DEPRECATED: kept for compatibility but should be removed
func (id ObjectID) Int64() int64 {
	return int64(id.Uint64Value)
}

// Int64Ptr returns a pointer to an int64 version of the ID or nil if it's invalid
// DEPRECATED: kept for compatibility but should be removed
func (id ObjectID) Int64Ptr() *int64 {
	if !id.IsValid {
		return nil
	}
	i := id.Int64()
	return &i
}

// Uint64 returns a uint64 version of the ID
func (id ObjectID) Uint64() uint64 {
	return id.Uint64Value
}

// Uint64Ptr returns a pointer to a uint64 version of the ID or nil if it's invalid
func (id ObjectID) Uint64Ptr() *uint64 {
	if !id.IsValid {
		return nil
	}
	return &id.Uint64Value
}

// Scan implements sql.Scanner and expects src to be nil or of type uint64, int64, or string
func (id *ObjectID) Scan(src interface{}) error {
	if src == nil {
		*id = ObjectID{
			Uint64Value: 0,
			IsValid:     false,
		}
		return nil
	}

	var i uint64
	switch v := src.(type) {
	case uint64:
		i = v
	case int64:
		i = uint64(v)
	case []byte:
		var err error
		i, err = strconv.ParseUint(string(v), 10, 64)
		if err != nil {
			*id = ObjectID{
				Uint64Value: 0,
				IsValid:     false,
			}
			return errors.Trace(fmt.Errorf("encoding: failed to scan ObjectID string '%s': %s", v, err))
		}
	case string:
		var err error
		i, err = strconv.ParseUint(v, 10, 64)
		if err != nil {
			*id = ObjectID{
				Uint64Value: 0,
				IsValid:     false,
			}
			return errors.Trace(fmt.Errorf("encoding: failed to scan ObjectID string '%s': %s", v, err))
		}
	default:
		return errors.Trace(fmt.Errorf("encoding: unsupported type for ObjectID.Scan: %T", src))
	}
	*id = ObjectID{
		Uint64Value: i,
		IsValid:     true,
	}

	return nil
}

// Value implements sql/driver.Valuer to allow an ObjectID to be used in an sql query
func (id ObjectID) Value() (driver.Value, error) {
	if !id.IsValid {
		return nil, nil
	}
	return int64(id.Uint64Value), nil
}

// String implements fmt.Stringer to provide a string representation of the ID
func (id ObjectID) String() string {
	return strconv.FormatUint(id.Uint64Value, 10)
}
