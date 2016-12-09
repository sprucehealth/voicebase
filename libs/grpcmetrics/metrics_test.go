package grpcmetrics

import (
	"context"
	"testing"

	"github.com/sprucehealth/backend/libs/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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
	out, err := methods[0].Handler(nil, context.Background(), nil, nil)
	if out != nil {
		t.Fatalf("Expected out to be nil, got %#v", out)
	}
	if err == nil {
		t.Fatal("Expected non-nil error")
	}
}

func TestCodedErrors(t *testing.T) {
	methods := []grpc.MethodDesc{
		{
			MethodName: "PanicTest",
			Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				return nil, errors.Trace(grpc.Errorf(codes.NotFound, "Not found"))
			},
		},
	}
	WrapMethods(methods)
	out, err := methods[0].Handler(nil, context.Background(), nil, nil)
	if out != nil {
		t.Fatalf("Expected out to be nil, got %#v", out)
	}
	if err == nil {
		t.Fatal("Expected non-nil error")
	}
	if c := grpc.Code(err); c != codes.NotFound {
		t.Fatalf("Expected NotFound, got %s", err)
	}
}
