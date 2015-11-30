package trace

import (
	"testing"

	"golang.org/x/net/context"
)

func TestRPC(t *testing.T) {
	tr := New("client", "test", 0, 0)
	ctx := TraceContext(context.Background(), tr)

	rpcCtx := RPCClientContext(ctx)
	_, tr2 := RPCServerContext(rpcCtx, "server", "test")
	if tr2.TraceID() != tr.TraceID() {
		t.Fatal("Trace ID does not match")
	}
}
