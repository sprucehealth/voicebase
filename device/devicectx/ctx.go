package devicectx

import (
	"github.com/sprucehealth/backend/device"
	"golang.org/x/net/context"
)

type ctxKey string

const (
	// CtxSpruceHeaders maps to the spruce headers attached to this context
	CtxSpruceHeaders ctxKey = "DeviceCtxSpruceHeaders"
)

// WithSpruceHeaders attaches the provided spruce headers onto a copy of the provided context
func WithSpruceHeaders(ctx context.Context, sh *device.SpruceHeaders) context.Context {
	return context.WithValue(ctx, CtxSpruceHeaders, sh)
}

// SpruceHeaders returns the spruce headers which may be nil
func SpruceHeaders(ctx context.Context) *device.SpruceHeaders {
	sh, _ := ctx.Value(CtxSpruceHeaders).(*device.SpruceHeaders)
	if sh == nil {
		return &device.SpruceHeaders{}
	}
	return sh
}
