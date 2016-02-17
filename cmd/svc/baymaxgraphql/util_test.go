package main

import (
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestIsValidPlane0Unicode(t *testing.T) {
	test.Equals(t, true, isValidPlane0Unicode(`This is a välid string`))
	test.Equals(t, false, isValidPlane0Unicode(`This is not 😡`))
}
