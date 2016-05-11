package care

import gcode "google.golang.org/grpc/codes"

const (
	// ErrorInvalidAnswer indicates that an answer attempted to be submitted is invalid.
	ErrorInvalidAnswer gcode.Code = 100
)
