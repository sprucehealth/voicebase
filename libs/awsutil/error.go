package awsutil

// ErrAWS implements the awserr.Error interface
type ErrAWS struct {
	CodeF    string
	MessageF string
	OrigErrF error
}

// Code returns the Code
func (e ErrAWS) Code() string {
	return e.CodeF
}

// Message returns the Message
func (e ErrAWS) Message() string {
	return e.MessageF
}

// OrigErr returns the source error
func (e ErrAWS) OrigErr() error {
	return e.OrigErrF
}

// Error returns the Message
func (e ErrAWS) Error() string {
	return e.MessageF
}
