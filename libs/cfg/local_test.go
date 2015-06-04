package cfg

import (
	"testing"
	"time"
)

func TestLocalConfig(t *testing.T) {
	lc, err := NewLocalStore([]*ValueDef{
		{Name: "int", Type: ValueTypeInt, Default: 123},
		{Name: "float", Type: ValueTypeFloat, Default: 99.0},
		{Name: "string", Type: ValueTypeString, Default: "abc"},
		{Name: "duration", Type: ValueTypeDuration, Default: time.Duration(time.Second)},
		{Name: "bool", Type: ValueTypeBool, Default: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer lc.Close() // Should be a noop and not panic (about the only thing that could go wrong I guess)

	if n := len(lc.Defs()); n != 5 {
		t.Errorf("Expected 5 defs, got %d", n)
	}

	// Check returns when no values have been set
	s := lc.Snapshot()
	if s.Int("int") != 123 {
		t.Errorf("Expected non existant int to return default")
	}
	if s.Int64("int") != 123 {
		t.Errorf("Expected non existant int to return default")
	}
	if s.Float64("float") != 99 {
		t.Errorf("Expected non existant float64 to return default")
	}
	if s.String("string") != "abc" {
		t.Errorf("Expected non existant string to return default")
	}
	if s.Duration("duration") != time.Second {
		t.Errorf("Expected non existant duration to return default")
	}
	if s.Bool("bool") != true {
		t.Errorf("Expected non existant bool to return default ")
	}

	if err := lc.Update(map[string]interface{}{
		"int":      111,
		"float":    1.11,
		"string":   "aaa",
		"duration": time.Hour,
		"bool":     false,
	}); err != nil {
		t.Fatal(err)
	}

	// Check returns for updated values
	s = lc.Snapshot()
	if n := s.Len(); n != 5 {
		t.Errorf("Expected %d values in snapshot, got %d", 5, n)
	}
	if n := len(s.Values()); n != 5 {
		t.Errorf("Expected %d values, got %d", 5, n)
	}
	if v := s.Int("int"); v != 111 {
		t.Errorf("Expected %d for int, got %d", 111, v)
	}
	if v := s.Int64("int"); v != 111 {
		t.Errorf("Expected %d for int, got %d", 111, v)
	}
	if v := s.Float64("float"); v != 1.11 {
		t.Errorf("Expected %f for float64, got %f", 1.11, v)
	}
	if v := s.String("string"); v != "aaa" {
		t.Errorf("Expected %s for string, got %s", "aaa", v)
	}
	if v := s.Duration("duration"); v != time.Hour {
		t.Errorf("Expected %s for duration, got %s", time.Hour, v)
	}
	if v := s.Bool("bool"); v != false {
		t.Errorf("Expected %t for bool, got %t", false, v)
	}
}

func TestLocalConfigNotRegistered(t *testing.T) {
	lc, err := NewLocalStore(nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := lc.Update(map[string]interface{}{
		"aaa": 111,
	}); err != nil {
		t.Fatal(err)
	}
	if v := lc.Snapshot().Int("aaa"); v != 0 {
		t.Fatalf("Unregistered value should not be set. Expected %d got %d", 111, v)
	}
}

func TestLocalConfigBadType(t *testing.T) {
	lc, err := NewLocalStore(nil)
	lc.Register(&ValueDef{Name: "int", Type: ValueTypeInt, Default: 222})
	if err != nil {
		t.Fatal(err)
	}
	if err := lc.Update(map[string]interface{}{
		"int": 111,
	}); err != nil {
		t.Fatal(err)
	}
	// The updating of a value to the wrong type should be ignored
	if err := lc.Update(map[string]interface{}{
		"int": "abc",
	}); err != nil {
		t.Fatal(err)
	}
	if v := lc.Snapshot().Int("int"); v != 111 {
		t.Fatalf("Expected %d got %d", 111, v)
	}
}
