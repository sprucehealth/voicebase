package media

import (
	"crypto/rand"
	"fmt"
	"io"
)

// NewID returns an ID that conforms to the media id specifications
func NewID() (string, error) {
	buff := make([]byte, 16)

	_, err := io.ReadFull(rand.Reader, buff)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x-%x-%x-%x-%x-%x", buff[0:4], buff[4:6], buff[6:8], buff[8:10], buff[10:12], buff[12:]), nil
}
