package server

import (
	"strings"
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

func TestEncodeAttachment(t *testing.T) {
	out, err := encodeAttachment(strings.NewReader("foo"))
	test.OK(t, err)
	test.Equals(t, "Zm9v", out)
}
