package grpcmetrics

import (
	"context"
	"testing"

	"google.golang.org/grpc"
)

func TestWrapMetricsPanic(t *testing.T) {
	methods := []grpc.MethodDesc{
		{
			MethodName: "PanicTest",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				panic("BOOM")
			},
		},
	}
	WrapMethods(methods)
	out, err := methods[0].Handler(nil, nil, nil, nil)
	if out != nil {
		t.Fatalf("Expected out to be nil, got %#v", out)
	}
	if err == nil {
		t.Fatal("Expected non-nil error")
	}
}
