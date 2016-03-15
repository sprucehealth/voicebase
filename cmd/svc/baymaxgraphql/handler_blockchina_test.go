package main

import (
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestCIDRRange(t *testing.T) {
	inChina := []string{
		"171.83.247.199",
		"27.16.106.62",
		"111.183.20.200",
		"121.62.232.154",
		"171.44.226.61",
	}

	for _, s := range inChina {
		test.Equals(t, true, ipAddressFromChina(s))
	}

	notInChina := []string{
		"71.198.128.74",
		"96.243.228.192",
		"99.127.47.62",
	}

	for _, s := range notInChina {
		test.Equals(t, false, ipAddressFromChina(s))
	}
}

func BenchmarkIPCheck(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ipAddressFromChina("171.83.247.199")
	}
}
