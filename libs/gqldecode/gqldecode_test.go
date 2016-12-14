package gqldecode

import (
	"reflect"
	"testing"
)

func TestSimple(t *testing.T) {
	input := map[string]interface{}{
		"name":         "Gob",
		"age":          45,
		"person":       true,
		"keywords":     []interface{}{"blacklisted", "magician"},
		"optStringSet": "foo",
	}
	type simpleStruct struct {
		Name            string   `gql:"name"`
		Age             int      `gql:"age"`
		Person          bool     `gql:"person"`
		Keywords        []string `gql:"keywords"`
		OptStringSet    *string  `gql:"optStringSet"`
		OptStringNotSet *string  `gql:"optStringNotSet"`
	}
	var st simpleStruct
	if err := Decode(input, &st); err != nil {
		t.Fatal(err)
	}
	foo := "foo"
	exp := simpleStruct{
		Name:            "Gob",
		Age:             45,
		Person:          true,
		Keywords:        []string{"blacklisted", "magician"},
		OptStringSet:    &foo,
		OptStringNotSet: nil,
	}
	if !reflect.DeepEqual(exp, st) {
		t.Fatalf("Expected %+v got %+v", exp, st)
	}
}

func TestSubStruct(t *testing.T) {
	input := map[string]interface{}{
		"ptr":         map[string]interface{}{"foo": "bar"},
		"nonptr":      map[string]interface{}{"foo": "123"},
		"ptrslice":    []interface{}{map[string]interface{}{"foo": "aaa"}},
		"nonptrslice": []interface{}{map[string]interface{}{"foo": "zzz"}},
	}
	type subStruct struct {
		Foo string `gql:"foo"`
	}
	type withSubStruct struct {
		Ptr         *subStruct   `gql:"ptr"`
		NonPtr      subStruct    `gql:"nonptr"`
		PtrSlice    []*subStruct `gql:"ptrslice"`
		NonPtrSlice []subStruct  `gql:"nonptrslice"`
	}
	var st withSubStruct
	if err := Decode(input, &st); err != nil {
		t.Fatal(err)
	}
	exp := withSubStruct{
		Ptr:         &subStruct{Foo: "bar"},
		NonPtr:      subStruct{Foo: "123"},
		PtrSlice:    []*subStruct{{Foo: "aaa"}},
		NonPtrSlice: []subStruct{{Foo: "zzz"}},
	}
	if !reflect.DeepEqual(exp, st) {
		t.Fatalf("Expected %+v got %+v", exp, st)
	}
}

func TestPlane0Validation(t *testing.T) {
	// Allow plane0

	input := map[string]interface{}{
		"name": "Foo",
	}
	type plane0Struct struct {
		Name string `gql:"name,plane0"`
	}
	st := &plane0Struct{}
	if err := Decode(input, st); err != nil {
		t.Fatal(err)
	}
	exp := &plane0Struct{
		Name: "Foo",
	}
	if !reflect.DeepEqual(exp, st) {
		t.Fatalf("Expected %+v got %+v", exp, st)
	}

	// Non plane0 should fail

	input = map[string]interface{}{
		"name": "Foo👀",
	}
	if err, ok := Decode(input, st).(ErrValidationFailed); !ok {
		t.Fatal("Expected ErrValidationFailed")
	} else if err.Field != "name" {
		t.Fatalf("Expected field 'name' got '%s'", err.Field)
	}
}

func BenchmarkDecode(b *testing.B) {
	input := map[string]interface{}{
		"string":     "vvvvv",
		"int":        123123,
		"stringList": []interface{}{"abc", "foo"},
		"struct": map[string]interface{}{
			"field": "string",
		},
	}
	type benchmarkSubStruct struct {
		Field string `gql:"field"`
	}
	type benchmarkStruct struct {
		String     string              `gql:"string"`
		Int        int                 `gql:"int"`
		StringList []string            `gql:"stringList"`
		Struct     *benchmarkSubStruct `gql:"struct"`
	}
	b.ReportAllocs()
	b.ResetTimer()
	var st benchmarkStruct
	for i := 0; i < b.N; i++ {
		if err := Decode(input, &st); err != nil {
			b.Fatal(err)
		}
	}
}
