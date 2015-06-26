package ptr

import "time"

func Int64Ptr(i int64) *int64 {
	return &i
}

func TimePtr(t time.Time) *time.Time {
	return &t
}
