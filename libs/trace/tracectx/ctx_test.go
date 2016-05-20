package tracectx

import (
	"testing"

	"github.com/sprucehealth/backend/test"
	"golang.org/x/net/context"
)

func TestRequestIDRoundTrip(t *testing.T) {
	cases := map[string]struct {
		Context   context.Context
		RequestID uint64
		Expected  uint64
	}{
		"RequestID": {
			Context:   context.Background(),
			RequestID: 12345,
			Expected:  12345,
		},
	}

	for cn, c := range cases {
		test.EqualsCase(t, cn, c.Expected, RequestID(WithRequestID(c.Context, c.RequestID)))
	}
}
