package slice

import (
	"reflect"
	"testing"
)

// Contains asserts that the desired element is present in the provided slice
func Contains(t *testing.T, s []interface{}, ele interface{}) {
	for _, v := range s {
		if reflect.DeepEqual(v, ele) {
			return
		}
	}
	t.Fatalf("Unable to locate element %+v in the provided slice", ele)
}

// AsISlice converts any provided slice into a slice of interfaces
func AsISlice(sli interface{}) []interface{} {
	s := reflect.ValueOf(sli)
	if s.Kind() != reflect.Slice {
		panic("The provided argument is not a slice")
	}

	ret := make([]interface{}, s.Len())
	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}

	return ret
}
