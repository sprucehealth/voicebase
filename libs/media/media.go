package media

import "errors"

// ErrNotFound is returned when the requested media was not found
var (
	ErrNotFound = errors.New("media: media not found")
	ErrTooLarge = errors.New("media: media too large")
)

const (
	mimeTypeHeader      = "Content-Type"
	contentLengthHeader = "Content-Length"
	widthHeader         = "x-amz-meta-width"
	heightHeader        = "x-amz-meta-height"
	durationHeader      = "duration"
)