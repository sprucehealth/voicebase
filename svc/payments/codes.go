package payments

import gcode "google.golang.org/grpc/codes"

// The error code space for the auth service is 1XX
const (
	// PaymentMethodError indicates that the resulting error is due to a error in the payment method
	PaymentMethodError gcode.Code = 100
)
