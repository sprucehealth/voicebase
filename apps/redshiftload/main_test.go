package main

import (
	"testing"
)

func BenchmarkStruct(b *testing.B) {
	b.ReportAllocs()
	a := 123
	var c interface{}
	for i := 0; i < b.N; i++ {
		c = &struct {
			A int
		}{
			A: a,
		}
	}
	_ = c
}

func BenchmarkMap(b *testing.B) {
	b.ReportAllocs()
	a := 123
	var c interface{}
	for i := 0; i < b.N; i++ {
		c = map[string]interface{}{
			"A": a,
		}
	}
	_ = c
}
