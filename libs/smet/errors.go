package smet

import "github.com/sprucehealth/backend/libs/golog"

// Error logs the provided error and increases the keyed metric by 1
func Error(key string, err error) {
	Errorf(key, err.Error())
}

// Errorf logs the provided information and increases the keyed metric by 1
func Errorf(key, sfmt string, args ...interface{}) {
	golog.Context("metric_type", "counter", "metric", key).LogDepthf(1, golog.ERR, sfmt, args...)
	GetCounter(key).Inc()
}
