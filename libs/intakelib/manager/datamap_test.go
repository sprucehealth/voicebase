package manager

import "testing"

func TestRequiredKeys(t *testing.T) {
	data := dataMap(map[string]interface{}{
		"questions": "aegpajg",
		"answers":   "ewpghjag",
	})

	if err := data.requiredKeys("test", "questions"); err != nil {
		t.Fatal(err)
	}

	if err := data.requiredKeys("test", "questions", "answers"); err != nil {
		t.Fatal(err)
	}

	if err := data.requiredKeys("test", "missing_key"); err == nil {
		t.Fatal("Expected required keys check to fail but it didnt")
	}
}

func TestGetIfX(t *testing.T) {
	data := dataMap(map[string]interface{}{
		"integer":         1,
		"float":           1.234,
		"string":          "hello",
		"interface_slice": []interface{}{"1", "2", "3"},
		"str_slice":       []string{"1", "2", "3"},
	})

	if intVal := data.mustGetInt("integer"); intVal != 1 {
		t.Fatalf("Expected %d got %d", 1, intVal)
	}

	if floatVal := data.mustGetFloat32("float"); floatVal != 1.234 {
		t.Fatalf("Expected %f to be returned for float instead got %f", 1.234, floatVal)
	}

	if strVal := data.mustGetString("string"); strVal != "hello" {
		t.Fatalf("Expected %s got %s", "hello", strVal)
	}

	strSliceVal, err := data.getStringSlice("str_slice")
	if err != nil {
		t.Fatal(err)
	} else if len(strSliceVal) != 3 {
		t.Fatalf("Expected %v got %v", []string{"1", "2", "3"}, strSliceVal)
	}

	iSliceVal, err := data.getStringSlice("interface_slice")
	if err != nil {
		t.Fatal(err)
	} else if len(iSliceVal) != 3 {
		t.Fatalf("Expected %v got %v", []string{"1", "2", "3"}, iSliceVal)
	}
}

func TestDataMapForKey(t *testing.T) {
	data := dataMap(map[string]interface{}{
		"client_data": map[string]interface{}{
			"integerString": "1",
			"float":         1.234,
			"string":        "hello",
			"string_slice":  []string{"1", "2", "3"},
		},
	})

	clientData, err := data.dataMapForKey("client_data")
	if err != nil {
		t.Fatal(err)
	} else if clientData == nil {
		t.Fatalf("Expected clientData object to exist in map but it doesnt")
	}
}

func TestJsonData(t *testing.T) {
	data := dataMap(map[string]interface{}{
		"client_data": map[string]interface{}{
			"integerString": "1",
			"float":         1.234,
			"string":        "hello",
			"string_slice":  []string{"1", "2", "3"},
		},
	})

	jsonData, err := data.getJSONData("client_data")
	if err != nil {
		t.Fatal(err)
	} else if len(jsonData) == 0 {
		t.Fatalf("Expected json data to be returned but got none")
	}
}

func BenchmarkStringAccess(b *testing.B) {
	data := dataMap(map[string]interface{}{
		"string": "hello",
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if strVal := data.mustGetString("string"); strVal != "hello" {
			b.Fatalf("Expected %s got %s", "hello", strVal)
		}
	}
}
