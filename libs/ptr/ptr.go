package ptr

import "time"

func Bool(b bool) *bool {
	return &b
}

func Int64(i int64) *int64 {
	return &i
}

func String(s string) *string {
	return &s
}

func Time(t time.Time) *time.Time {
	return &t
}
