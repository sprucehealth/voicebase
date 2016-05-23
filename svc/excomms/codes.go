package excomms

import gcode "google.golang.org/grpc/codes"

const (
	// ErrorCodeMessageLengthExceeded indicates that the message cannot be delivered
	// due to the length exceeding the limit.
	ErrorCodeMessageLengthExceeded gcode.Code = 100

	// ErrorCodeSMSIncapableFromPhoneNumber indicates that the from number is not
	// capable of sending SMS.
	ErrorCodeSMSIncapableFromPhoneNumber gcode.Code = 101

	// ErrorCodeMessageDeliveryFailed indicates that the message cannot be delivered
	// to the destination.
	ErrorCodeMessageDeliveryFailed gcode.Code = 102
)
