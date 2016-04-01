package dal

import (
	"testing"
)

type nullScanner struct{}

func (nullScanner) Scan(v ...interface{}) error {
	return nil
}

func BenchmarkScanEntity(b *testing.B) {
	row := nullScanner{}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e, err := scanEntity(row)
		if err != nil {
			b.Fatal(err)
		}
		e.Recycle()
	}
}

func BenchmarkScanEntityContact(b *testing.B) {
	row := nullScanner{}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e, err := scanEntityContact(row)
		if err != nil {
			b.Fatal(err)
		}
		e.Recycle()
	}
}
