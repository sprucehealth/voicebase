package cfg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type ValueType string

const (
	ValueTypeBool     ValueType = "bool"
	ValueTypeInt      ValueType = "int"   // 64-bit
	ValueTypeFloat    ValueType = "float" // 64-bit
	ValueTypeString   ValueType = "string"
	ValueTypeDuration ValueType = "duration"
)

// ValueDef is the definition for a value to store.
type ValueDef struct {
	Name        string        `json:"name"`
	Type        ValueType     `json:"type"`
	Description string        `json:"description"`
	Default     interface{}   `json:"default"`
	Choices     []interface{} `json:"choices"`
	Multi       bool          `json:"multi"`
}

// Store is the interface implemented by configuration storage backends.
type Store interface {
	Register(def *ValueDef)
	Defs() map[string]*ValueDef
	Snapshot() Snapshot
	Update(map[string]interface{}) error
	Close() error
}

// Valid returns true if the type is one of the defined constants
func (vt ValueType) Valid() bool {
	switch vt {
	case ValueTypeBool, ValueTypeFloat, ValueTypeInt, ValueTypeString, ValueTypeDuration:
		return true
	}
	return false
}

// Valid returns true if all fields of the definition are valid
func (d *ValueDef) Valid() bool {
	if d.Name == "" {
		return false
	}
	if !d.Type.Valid() {
		return false
	}
	if d.Default == nil {
		return false
	}
	var ok bool
	d.Default, ok = normalizeType(d.Default, d.Type, false)
	if !ok {
		return false
	}
	if len(d.Choices) != 0 {
		v := d.Choices[0]
		if _, ok := normalizeType(v, d.Type, false); !ok {
			return false
		}
	}
	return true
}

// normalizeType makes sure the value at the interfaces matches the
// expected type and returns a normalized version (e.g. int64 for int).
// If coerce is true then try to convert the type safely (anything that
// does not result in loss of precision).
func normalizeType(v interface{}, t ValueType, coerce bool) (interface{}, bool) {
	switch v := v.(type) {
	case int:
		if t != ValueTypeInt {
			return v, false
		}
		return int64(v), true
	case int64:
		if t == ValueTypeDuration {
			return time.Duration(v), true
		}
		return v, t == ValueTypeInt
	case string:
		if t == ValueTypeString {
			return v, true
		}
		if !coerce {
			return v, false
		}
		switch t {
		case ValueTypeDuration:
			ns, err := strconv.ParseInt(v, 10, 64)
			if err == nil {
				return time.Duration(ns), true
			}
			d, err := time.ParseDuration(v)
			return d, err == nil
		case ValueTypeInt:
			i, err := strconv.ParseInt(v, 10, 64)
			return i, err == nil
		case ValueTypeFloat:
			f, err := strconv.ParseFloat(v, 64)
			return f, err == nil
		}
		return v, false
	case float64:
		return v, t == ValueTypeFloat
	case time.Duration:
		return v, t == ValueTypeDuration
	}
	return v, false
}

// jsonValue allows decoding of JSON so as to preserve integer values.
// By default the JSON decoder when unmarshaling to an interface{}
// converts all numbers to float64.
type jsonValue struct {
	v interface{}
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (v *jsonValue) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		v.v = nil
		return nil
	}
	// Use the standard decoder for all non-numbers
	c := byte(b[0])
	if (c < '0' || c > '9') && c != '-' && c != '+' {
		return json.Unmarshal(b, &v.v)
	}
	// Check for a decimal or exponent to detect float
	if bytes.IndexByte(b, '.') >= 0 || bytes.IndexByte(b, 'e') >= 0 {
		var err error
		v.v, err = strconv.ParseFloat(string(b), 64)
		return err

	}
	var err error
	v.v, err = strconv.ParseInt(string(b), 10, 64)
	return err
}

func DecodeValues(b []byte) (map[string]interface{}, error) {
	var val map[string]jsonValue
	if err := json.Unmarshal(b, &val); err != nil {
		return nil, err
	}
	snap := make(map[string]interface{}, len(val))
	for n, v := range val {
		snap[n] = v.v
	}
	return snap, nil
}

func CoerceValues(def map[string]*ValueDef, val map[string]interface{}) error {
	for name, d := range def {
		if v, ok := val[name]; ok {
			v, ok := normalizeType(v, d.Type, true)
			if !ok {
				return fmt.Errorf("cfg: invalid type for %s", name)
			}
			val[name] = v
		}
	}
	return nil
}
