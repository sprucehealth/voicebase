package handlers

import (
	"testing"

	"github.com/sprucehealth/backend/libs/test"
)

func TestNewShortID(t *testing.T) {
	id, err := newShortID()
	test.OK(t, err)
	test.Equals(t, 11, len(id))
}
