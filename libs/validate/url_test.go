package validate

import (
	"testing"

	"github.com/sprucehealth/backend/test"
)

func TestRemoteHost(t *testing.T) {
	tests := []struct {
		h string
		v bool
	}{
		{h: "google.com", v: true},
		{h: "google.feiwjf", v: false},
		{h: "127.0.0.1", v: false},
		{h: "corp-looker.carefront.net", v: false},
	}
	for _, tc := range tests {
		reason, v := RemoteHost(tc.h, true)
		test.Assert(t, tc.v == v, "Expected %t for '%s' got %t because %s", tc.v, tc.h, v, reason)
	}
}
