package models

import (
	"database/sql/driver"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/sprucehealth/backend/libs/errors"
)

const base32EncodedUint64len = 13 // length of 8-byte as base32 with '=' stripped

var errInvalidID = errors.New("invalid ID")

// objectID is used for the (un)marshalling of data models 64-bit IDs
type objectID struct {
	prefix  string
	value   uint64
	isValid bool
}

// MarshalText implements encoding.TextMarshaler
func (id objectID) MarshalText() ([]byte, error) {
	if !id.isValid {
		return nil, nil
	}
	b := make([]byte, len(id.prefix)+base32EncodedUint64len+3+8) // +3 for padding, +8 for scratch area
	ib := b[len(b)-8:]
	b = b[:len(b)-8]
	copy(b, id.prefix)
	binary.BigEndian.PutUint64(ib, id.value)
	base32.HexEncoding.Encode(b[len(id.prefix):], ib)
	return b[:len(b)-3], nil // -3 remove padding
}

// UnmarshalText implements encoding.TextUnmarshaler
func (id *objectID) UnmarshalText(text []byte) error {
	id.value = 0
	id.isValid = false
	if len(text) == 0 {
		return nil
	}
	if len(text) != len(id.prefix)+base32EncodedUint64len {
		return errors.Trace(errInvalidID)
	}
	s := string(text)
	if s[:len(id.prefix)] != id.prefix {
		return errors.Trace(errInvalidID)
	}
	s = s[len(id.prefix):]
	b, err := base32.HexEncoding.DecodeString(s + "===") // repad for decoding
	if err != nil {
		return errors.Trace(errInvalidID)
	}
	id.value = binary.BigEndian.Uint64(b)
	id.isValid = true
	return nil
}

// Scan implements sql.Scanner and expects src to be nil or of type uint64, int64, or string
func (id *objectID) Scan(src interface{}) error {
	id.value = 0
	id.isValid = false
	if src == nil {
		return nil
	}
	switch v := src.(type) {
	case uint64:
		id.value = v
	case int64:
		id.value = uint64(v)
	case []byte:
		var err error
		id.value, err = strconv.ParseUint(string(v), 10, 64)
		if err != nil {
			return errors.Trace(fmt.Errorf("failed to scan objectID string '%s': %s", v, err))
		}
	case string:
		var err error
		id.value, err = strconv.ParseUint(v, 10, 64)
		if err != nil {
			return errors.Trace(fmt.Errorf("failed to scan objectID string '%s': %s", v, err))
		}
	default:
		return errors.Trace(fmt.Errorf("unsupported type for objectID.Scan: %T", src))
	}
	id.isValid = true
	return nil
}

// Value implements sql/driver.Valuer to allow an ObjectID to be used in an sql query
func (id objectID) Value() (driver.Value, error) {
	if !id.isValid {
		return nil, nil
	}
	// int64 because uint64 isn't supported by the sql/driver.Valuer interface
	return int64(id.value), nil
}

// String implements fmt.Stringer to provide a string representation of the ID
func (id objectID) String() string {
	b, _ := id.MarshalText()
	return string(b)
}
