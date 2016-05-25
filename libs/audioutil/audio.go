package audioutil

import (
	"io"
	"time"

	"github.com/nareix/mp4"
	"github.com/tcolgate/mp3"
)

// Duration returns the duration of the provided media. If the media type is not supported then a duration of 0 is returned
// Note: This may progress the provided reader
// TODO: Perhaps figure out a way to determine mimeType from the raw input
func Duration(r io.ReadSeeker, mimeType string) (time.Duration, error) {
	switch mimeType {
	case "audio/mpeg":
		return mp3Duration(r)
	case "audio/mp4":
		return mp4Duration(r)
	}
	return time.Duration(0), nil
}

func mp3Duration(r io.ReadSeeker) (time.Duration, error) {
	dec := mp3.NewDecoder(r)
	var frame mp3.Frame
	var duration time.Duration
	for {
		if err := dec.Decode(&frame); err != nil {
			if err == io.EOF {
				return duration, nil
			}
			return 0, err
		}
		duration += frame.Duration()
	}
}

func mp4Duration(r io.ReadSeeker) (time.Duration, error) {
	demuxer := mp4.Demuxer{R: r}
	if err := demuxer.ReadHeader(); err != nil {
		return 0, err
	}
	// Peserve the ms section of our float value before rounding
	return time.Duration(int64(time.Millisecond) * int64(1000*demuxer.TrackAAC.Duration())), nil
}
