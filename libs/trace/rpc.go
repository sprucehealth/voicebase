package trace

import (
	"strconv"

	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/idgen"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
)

const (
	TraceID = "traceID"
	SpanID  = "spanID"
)

// RPCClientContext builds a context that contains metadata for
// trace and span ID to be sent to a server.
func RPCClientContext(ctx context.Context) context.Context {
	t, _ := FromContext(ctx)

	// Generate a new span ID for the call
	spanID, err := idgen.NewID()
	if err != nil {
		golog.Errorf("Failed to generate new span ID: %s", err)
	}

	return metadata.NewContext(ctx, metadata.Pairs(
		"traceID", strconv.FormatUint(t.TraceID(), 10),
		"spanID", strconv.FormatUint(spanID, 10),
	))
}

// RPCServerContext builds a context and trace based on metadata received
// from a client.
func RPCServerContext(ctx context.Context, family, name string) (context.Context, *Trace) {
	md, _ := metadata.FromContext(ctx)
	traceID, _ := strconv.ParseUint(md["traceID"][0], 10, 64)
	spanID, _ := strconv.ParseUint(md["spanID"][0], 10, 64)
	t := New(family, name, traceID, spanID)
	return TraceContext(ctx, t), t
}
