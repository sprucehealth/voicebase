package cfg

import (
	"testing"
	"time"
)

func TestLocalConfig(t *testing.T) {
	lc := NewLocalStore()
	lc.Register(&ValueDef{Name: "int", Type: ValueTypeInt, Default: 123})
	lc.Register(&ValueDef{Name: "float", Type: ValueTypeFloat, Default: 99.0})
	lc.Register(&ValueDef{Name: "string", Type: ValueTypeString, Default: "abc"})
	lc.Register(&ValueDef{Name: "duration", Type: ValueTypeDuration, Default: time.Duration(time.Second)})

	// Check returns when no values have been set
	s := lc.Snapshot()
	if s.Int("int") != 123 {
		t.Errorf("Expected non existant int to return default")
	}
	if s.Float64("float") != 99 {
		t.Errorf("Expected non existant float64 to return default")
	}
	if s.String("string") != "abc" {
		t.Errorf("Expected non existant string to return default")
	}
	if s.Duration("duration") != time.Second {
		t.Errorf("Expected non existant duraiton to return default")
	}

	if err := lc.Update(map[string]interface{}{
		"int":      111,
		"float":    1.11,
		"string":   "aaa",
		"duration": time.Hour,
	}); err != nil {
		t.Fatal(err)
	}

	// Check returns for updated values
	s = lc.Snapshot()
	if v := s.Int("int"); v != 111 {
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
}

func TestLocalConfigNotRegistered(t *testing.T) {
	lc := NewLocalStore()
	if err := lc.Update(map[string]interface{}{
		"int": 111,
	}); err != nil {
		t.Fatal(err)
	}
	if v := lc.Snapshot().Int("int"); v != 0 {
		t.Fatalf("Unregistered value should not be set. Expected %d got %d", 111, v)
	}
}

func TestLocalConfigBadType(t *testing.T) {
	lc := NewLocalStore()
	lc.Register(&ValueDef{Name: "int", Type: ValueTypeInt, Default: 222})
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
