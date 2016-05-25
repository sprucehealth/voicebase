package videoutil

import (
	"io"
	"time"

	"github.com/nareix/mp4"
)

// Duration returns the duration of the provided media. If the media type is not supported then a duration of 0 is returned
// Note: This may progress the provided reader
// TODO: Perhaps figure out a way to determine mimeType from the raw input
func Duration(r io.ReadSeeker, mimeType string) (time.Duration, error) {
	switch mimeType {
	case "video/mp4":
		return mp4Duration(r)
	}
	return time.Duration(0), nil
}

func mp4Duration(r io.ReadSeeker) (time.Duration, error) {
	demuxer := mp4.Demuxer{R: r}
	if err := demuxer.ReadHeader(); err != nil {
		return 0, err
	}
	// Peserve the ms section of our float value before rounding
	return time.Duration(int64(time.Millisecond) * int64(1000*demuxer.TrackH264.Duration())), nil
}
