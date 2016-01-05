package twilio

import (
	"github.com/sprucehealth/backend/svc/excomms"
	"golang.org/x/net/context"
)

// The key type is unexported to prevent collisions with context keys defined in
// other packages.
type key int

// twilioParamsKey is the context key for the twilio request params.
const twilioParamsKey key = 0

// NewContext returns a context with the twilio parameters as a value in the context.
func NewContext(ctx context.Context, params *excomms.TwilioParams) context.Context {
	return context.WithValue(ctx, twilioParamsKey, params)
}

// FromContext extracts the twilio parameters as a value from the context.
func FromContext(ctx context.Context) (*excomms.TwilioParams, bool) {
	// ctx.Value returns nil if ctx has no value for the key;
	// the net.IP type assertion returns ok=false for nil.
	params, ok := ctx.Value(twilioParamsKey).(*excomms.TwilioParams)
	return params, ok
}
