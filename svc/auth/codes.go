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

	// VerificationCodeExpired indicates that the provided verification code token has expired
	VerificationCodeExpired gcode.Code = 105

	// BadVerificationCode indicates that the provided code did not match the code mapped to the token
	BadVerificationCode gcode.Code = 106

	// ValueNotYetVerified indicates that a verified value was requested that has not yet been verified
	ValueNotYetVerified gcode.Code = 107

	// TokenExpired indicates that the provided token has expired
	TokenExpired gcode.Code = 108

	// AccountBlocked indicates that the account is blocked and the user is forbidden from authenticating in.
	AccountBlocked gcode.Code = 109

	// AccountSuspended indicates that the user account has been temporarily suspended and the user should
	// contact support.
	AccountSuspended gcode.Code = 110
)
