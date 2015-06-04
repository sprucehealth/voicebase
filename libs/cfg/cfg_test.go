package cfg

import "testing"

func TestValueDef(t *testing.T) {
	validDef := &ValueDef{
		Name:    "Var",
		Type:    ValueTypeBool,
		Default: true,
		Choices: []interface{}{true, false},
	}
	if err := validDef.Validate(); err != nil {
		t.Fatalf("Expected valid def got error %s", err)
	}

	def := &ValueDef{}
	if err := def.Validate(); err == nil {
		t.Errorf("Expected empty ValueDef to be invalid")
	}

	*def = *validDef
	def.Name = ""
	if err := def.Validate(); err == nil {
		t.Errorf("Expected missing Name to be invalid")
	}

	*def = *validDef
	def.Type = ""
	if err := def.Validate(); err == nil {
		t.Errorf("Expected missing Type to be invalid")
	}

	*def = *validDef
	def.Default = nil
	if err := def.Validate(); err == nil {
		t.Errorf("Expected missing Default to be invalid")
	}

	*def = *validDef
	def.Default = "not a bool"
	if err := def.Validate(); err == nil {
		t.Errorf("Expected bad Default type to be invalid")
	}

	*def = *validDef
	def.Choices = []interface{}{"not a bool"}
	if err := def.Validate(); err == nil {
		t.Errorf("Expected bad Choice type to be invalid")
	}
}

func TestNormalizeType(t *testing.T) {
	// Bool no coerce
	if v, ok := normalizeType(true, ValueTypeBool, false); !ok {
		t.Error("Couldn't normalize bool to bool")
	} else if v, ok := v.(bool); !ok {
		t.Errorf("Expected bool, got %T", v)
	} else if v != true {
		t.Errorf("Expected true, got %t", v)
	}
	// Bool coerce string
	if v, ok := normalizeType("true", ValueTypeBool, true); !ok {
		t.Error("String 'true' should coerce to bool")
	} else if v, ok := v.(bool); !ok {
		t.Errorf("Expected bool, got %T", v)
	} else if !v {
		t.Errorf("Expected true, got false")
	}
	// Int64 no coerce
	if v, ok := normalizeType(int64(123), ValueTypeInt, false); !ok {
		t.Error("Couldn't normalize int64 to in64")
	} else if v, ok := v.(int64); !ok {
		t.Errorf("Expected int64, got %T", v)
	} else if v != 123 {
		t.Errorf("Expected 123, got %d", v)
	}
	// Int64 coerce string
	if v, ok := normalizeType("123", ValueTypeInt, true); !ok {
		t.Error("String '123' should coerce to int64")
	} else if v, ok := v.(int64); !ok {
		t.Errorf("Expected int64, got %T", v)
	} else if v != 123 {
		t.Errorf("Expected 123, got %d", v)
	}
}

func TestJSONValue(t *testing.T) {
	var v jsonValue
	v.v = 123
	if err := v.UnmarshalJSON(nil); err != nil {
		t.Fatal(err)
	} else if v.v != nil {
		t.Error("Unmarshal of nil (or empty slice) should clear value")
	}

	// jsonValue should parse integers as int64 rather than float64
	if err := v.UnmarshalJSON([]byte(`-1234`)); err != nil {
		t.Fatal(err)
	}
	if i, ok := v.v.(int64); !ok {
		t.Errorf("Expected integer to be decoded to int rather than %T", v.v)
	} else if i != -1234 {
		t.Errorf("Expected -1234, got %d", i)
	}

	// should parse float (any number with a decimal or exponent) as float64
	if err := v.UnmarshalJSON([]byte(`1234.123`)); err != nil {
		t.Fatal(err)
	}
	if f, ok := v.v.(float64); !ok {
		t.Errorf("Expected float to be decoded to float64 rather than %T", v.v)
	} else if f != 1234.123 {
		t.Errorf("Expected 1234.123, got %f", f)
	}

	// other types should be decoded as usual for encoding/json
	if err := v.UnmarshalJSON([]byte(`"abc"`)); err != nil {
		t.Fatal(err)
	}
	if s, ok := v.v.(string); !ok {
		t.Errorf("Expected string to be decoded to string rather than %T", v.v)
	} else if s != "abc" {
		t.Errorf("Expected 'abc', got '%s'", s)
	}
}

func TestCoerceValues(t *testing.T) {
	values := map[string]interface{}{
		"int":  int64(111),
		"bool": "true",
	}
	err := CoerceValues(map[string]*ValueDef{
		"int": {
			Name:    "int",
			Type:    ValueTypeInt,
			Default: 222,
		},
		"bool": {
			Name:    "bool",
			Type:    ValueTypeBool,
			Default: false,
		},
	}, values)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := values["int"].(int64); !ok {
		t.Errorf("Expected int64, got %T", values["int"])
	} else if v != 111 {
		t.Errorf("Expected 111, got %d", v)
	}
	if v, ok := values["bool"].(bool); !ok {
		t.Errorf("Expected bool, got %T", values["bool"])
	} else if !v {
		t.Errorf("Expected true, got false")
	}
}
