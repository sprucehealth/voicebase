package twilio

import (
	"fmt"
)

const (
	// ErrorCodeInvalidAreaCode is returned whhen trying to get a phone number in an
	// invalid arae code. https://www.twilio.com/docs/errors/21451
	ErrorCodeInvalidAreaCode = 21451
	// ErrorCodeNoPhoneNumberInAreaCode is returned when there's no available phone numbers
	// in the requested area code. https://www.twilio.com/docs/api/errors/21452
	ErrorCodeNoPhoneNumberInAreaCode = 21452
	// ErrorCodeResourceNotFound is returned when the requested resource was not found
	// https://www.twilio.com/docs/api/errors/20404
	ErrorCodeResourceNotFound = 20404

	// ErrorCodeInvalidToPhoneNumber is returned when there was an attempt to initiate an outbound call
	// or send a message to an invalid phone number.
	// https://www.twilio.com/docs/api/errors/21211
	ErrorCodeInvalidToPhoneNumber = 21211

	// ErrorCodeNotMessageCapableFromPhoneNumber is returned when there is an attempt to send a
	// message from a phone number that is not a valid, message-capable phone number.
	// https://www.twilio.com/docs/api/errors/21606
	ErrorCodeNotMessageCapableFromPhoneNumber = 21606

	// ErrorCodeMessageLengthExceeded is returned when the concatenated message body exceeds
	// the 1600 character limit.
	// https://www.twilio.com/docs/api/errors/21617
	ErrorCodeMessageLengthExceeded = 21617
)

// Exception holds information about error response returned by Twilio API
type Exception struct {
	Status   int    `json:"status"`
	Message  string `json:"message"`
	Code     int    `json:"code"`
	MoreInfo string `json:"more_info"`
}

// Exception implements Error interface
func (e *Exception) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}
