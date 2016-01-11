package model

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

// ObjectID is used for the (un)marshalling of data models 64-bit IDs
type ObjectID struct {
	Prefix  string
	Val     uint64 // Cannot expose Val since we need that name for the Val() driver method
	IsValid bool
}

// MarshalText implements encoding.TextMarshaler
func (id ObjectID) MarshalText() ([]byte, error) {
	if !id.IsValid {
		return nil, nil
	}
	b := make([]byte, len(id.Prefix)+base32EncodedUint64len+3+8) // +3 for padding, +8 for scratch area
	ib := b[len(b)-8:]
	b = b[:len(b)-8]
	copy(b, id.Prefix)
	binary.BigEndian.PutUint64(ib, id.Val)
	base32.HexEncoding.Encode(b[len(id.Prefix):], ib)
	return b[:len(b)-3], nil // -3 remove padding
}

// UnmarshalText implements encoding.TextUnmarshaler
func (id *ObjectID) UnmarshalText(text []byte) error {
	id.Val = 0
	id.IsValid = false
	if len(text) == 0 {
		return nil
	}
	if len(text) != len(id.Prefix)+base32EncodedUint64len {
		return errors.Trace(errInvalidID)
	}
	s := string(text)
	if s[:len(id.Prefix)] != id.Prefix {
		return errors.Trace(errInvalidID)
	}
	s = s[len(id.Prefix):]
	b, err := base32.HexEncoding.DecodeString(s + "===") // repad for decoding
	if err != nil {
		return errors.Trace(errInvalidID)
	}
	id.Val = binary.BigEndian.Uint64(b)
	id.IsValid = true
	return nil
}

// Scan implements sql.Scanner and expects src to be nil or of type uint64, int64, or string
func (id *ObjectID) Scan(src interface{}) error {
	id.Val = 0
	id.IsValid = false
	if src == nil {
		return nil
	}
	switch v := src.(type) {
	case uint64:
		id.Val = v
	case int64:
		id.Val = uint64(v)
	case []byte:
		var err error
		id.Val, err = strconv.ParseUint(string(v), 10, 64)
		if err != nil {
			return errors.Trace(fmt.Errorf("failed to scan ObjectID string '%s': %s", v, err))
		}
	case string:
		var err error
		id.Val, err = strconv.ParseUint(v, 10, 64)
		if err != nil {
			return errors.Trace(fmt.Errorf("failed to scan ObjectID string '%s': %s", v, err))
		}
	default:
		return errors.Trace(fmt.Errorf("unsupported type for ObjectID.Scan: %T", src))
	}
	id.IsValid = true
	return nil
}

// Val implements sql/driver.Valr to allow an ObjectID to be used in an sql query
func (id ObjectID) Value() (driver.Value, error) {
	if !id.IsValid {
		return nil, nil
	}
	// int64 because uint64 isn't supported by the sql/driver.Valr interface
	return int64(id.Val), nil
}

// String implements fmt.Stringer to provide a string representation of the ID
func (id ObjectID) String() string {
	b, _ := id.MarshalText()
	return string(b)
}
