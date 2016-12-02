// Package ptr provides helpers to generating pointers to inline values.
package ptr

import "time"

// Bool returns a pointer to the provided value.
func Bool(b bool) *bool {
	return &b
}

// Int returns a pointer to the provided value.
func Int(i int) *int {
	return &i
}

// Int64 returns a pointer to the provided value.
func Int64(i int64) *int64 {
	return &i
}

// Uint64 returns a pointer to the provided value.
func Uint64(i uint64) *uint64 {
	return &i
}

// Int64NilZero returns nil if the value is zero otherwise it returns a pointer to the value.
func Int64NilZero(i int64) *int64 {
	if i == 0 {
		return nil
	}
	return &i
}

// String returns a pointer to the provided value.
func String(s string) *string {
	return &s
}

// Time returns a pointer to the provided value.
func Time(t time.Time) *time.Time {
	return &t
}

// Strings returns a list of pointers to the provided values.
func Strings(ss []string) []*string {
	pss := make([]*string, len(ss))
	for i, s := range ss {
		cs := s
		pss[i] = &cs
	}
	return pss
}
