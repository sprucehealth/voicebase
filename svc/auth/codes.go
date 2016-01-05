package auth

import gcode "google.golang.org/grpc/codes"

// The error code space for the auth service is 1XX
const (
	// EmailNotFound indicates that the provided email could not be found
	EmailNotFound gcode.Code = 100

	// DuplicateEmail indicates that the provided email already exists
	DuplicateEmail gcode.Code = 101

	// BadPassword indicates that the provided password was not correct
	BadPassword gcode.Code = 102

	// InvalidPhoneNumber indicates that a provided phone number is invalid
	InvalidPhoneNumber gcode.Code = 103

	// InvalidEmail indicates that a provided email is invalid
	InvalidEmail gcode.Code = 104
)
