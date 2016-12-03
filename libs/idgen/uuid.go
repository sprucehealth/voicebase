package idgen

import (
	"crypto/rand"
	"encoding/base64"
	"io"

	"github.com/sprucehealth/backend/libs/errors"
)

// NewUUID returns a probabilistic unique string using a cryptographic random source
func NewUUID() (string, error) {
	var b [20]byte
	_, err := io.ReadFull(rand.Reader, b[:])
	return base64.RawStdEncoding.EncodeToString(b[:]), errors.Trace(err)
}
