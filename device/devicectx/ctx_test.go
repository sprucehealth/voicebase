package devicectx

import (
	"testing"

	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/libs/test"
	"golang.org/x/net/context"
)

func TestSpruceHeadersRoundTrip(t *testing.T) {
	cases := map[string]struct {
		Context  context.Context
		SHeaders *device.SpruceHeaders
		Expected *device.SpruceHeaders
	}{
		"NilHeaders": {
			Context:  context.Background(),
			Expected: &device.SpruceHeaders{},
		},
		"NonNilHeaders": {
			Context: context.Background(),
			SHeaders: &device.SpruceHeaders{
				AppType:        "type",
				AppEnvironment: "env",
			},
			Expected: &device.SpruceHeaders{
				AppType:        "type",
				AppEnvironment: "env",
			},
		},
	}

	for cn, c := range cases {
		test.EqualsCase(t, cn, c.Expected, SpruceHeaders(WithSpruceHeaders(c.Context, c.SHeaders)))
	}
}
