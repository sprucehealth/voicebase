package cfg

import (
	"encoding/json"
	"time"

	"github.com/sprucehealth/backend/libs/golog"
)

// Snapshot is a read-only point-in-time view of the variables in the config store.
type Snapshot struct {
	values map[string]interface{}
	defs   map[string]*ValueDef
}

// Len returns the number of values in the snapshot.
func (s Snapshot) Len() int {
	return len(s.values)
}

// Bool returns named value from the snapshot if it is a boolean.
// Otherwise it returns the default value previously set in the
// value definition.
func (s Snapshot) Bool(name string) bool {
	if v, ok := s.values[name]; ok {
		v2, ok := v.(bool)
		if ok {
			return v2
		}
		golog.Errorf("config: expected a bool for '%s' got %T", name, v)
	}
	if d := s.defs[name]; d == nil {
		golog.Errorf("config: access of undefined bool '%s'", name)
	} else if d.Default != nil {
		return d.Default.(bool)
	}
	return false
}

// Int returns named value from the snapshot if it is an integer.
// Otherwise it returns the default value previously set in the
// value definition.
func (s Snapshot) Int(name string) int {
	if v, ok := s.values[name]; ok {
		v2, ok := v.(int64)
		if ok {
			return int(v2)
		}
		golog.Errorf("config: expected an int64 for '%s' got %T", name, v)
	}
	if d := s.defs[name]; d == nil {
		golog.Errorf("config: access of undefined int '%s'", name)
	} else if d.Default != nil {
		return int(d.Default.(int64))
	}
	return 0
}

// Int64 returns named value from the snapshot if it is an integer.
// Otherwise it returns the default value previously set in the
// value definition.
func (s Snapshot) Int64(name string) int64 {
	if v, ok := s.values[name]; ok {
		v2, ok := v.(int64)
		if ok {
			return v2
		}
		golog.Errorf("config: expected an int64 for '%s' got %T", name, v)
	}
	if d := s.defs[name]; d == nil {
		golog.Errorf("config: access of undefined int64 '%s'", name)
	} else if d.Default != nil {
		return d.Default.(int64)
	}
	return 0
}

// Float64 returns named value from the snapshot if it is a float.
// Otherwise it returns the default value previously set in the
// value definition.
func (s Snapshot) Float64(name string) float64 {
	if v, ok := s.values[name]; ok {
		v2, ok := v.(float64)
		if ok {
			return v2
		}
		golog.Errorf("config: expected a float64 for '%s' got %T", name, v)
	}
	if d := s.defs[name]; d == nil {
		golog.Errorf("config: access of undefined float64 '%s'", name)
	} else if d.Default != nil {
		return d.Default.(float64)
	}
	return 0.0
}

// String returns named value from the snapshot if it is a string.
// Otherwise it returns the default value previously set in the
// value definition.
func (s Snapshot) String(name string) string {
	if v, ok := s.values[name]; ok {
		v2, ok := v.(string)
		if ok {
			return v2
		}
		golog.Errorf("config: expected a string for '%s' got %T", name, v)
	}
	if d := s.defs[name]; d == nil {
		golog.Errorf("config: access of undefined string '%s'", name)
	} else if d.Default != nil {
		return d.Default.(string)
	}
	return ""
}

// Duration returns named value from the snapshot if it is a duration
// or an int64 (in which case it gets converted to a duration before turning).
// Otherwise it returns the default value previously set in the
// value definition.
func (s Snapshot) Duration(name string) time.Duration {
	if v, ok := s.values[name]; ok {
		switch v2 := v.(type) {
		case int64:
			return time.Duration(v2)
		case time.Duration:
			return v2
		}
		golog.Errorf("config: expected an int64 or time.Duration for '%s' got %T", name, v)
	}
	if d := s.defs[name]; d == nil {
		golog.Errorf("config: access of undefined duration '%s'", name)
	} else if d.Default != nil {
		return d.Default.(time.Duration)
	}
	return 0
}

// Values returns the map of names to values. The returned map should be
// considered read-only.
func (s Snapshot) Values() map[string]interface{} {
	return s.values
}

// MarshalJSON implements the json.Marshaler interface.
func (s Snapshot) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.values)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (s *Snapshot) UnmarshalJSON(b []byte) error {
	v, err := DecodeValues(b)
	if err != nil {
		return err
	}
	*s = Snapshot{
		values: v,
	}
	return nil
}
