package settings

import gcode "google.golang.org/grpc/codes"

const (
	// InvalidUserValue indicates that the value entered by the user is incorrect
	// for a certain setting config.
	InvalidUserValue gcode.Code = 100
)
