package server

import (
	"errors"
	"fmt"

	"github.com/sprucehealth/backend/cmd/svc/payments/internal/dal"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/svc/payments"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// go vet doesn't like that the first argument to grpcErrorf is not a string so alias the function with a different name :(
var grpcErrf = grpc.Errorf

func grpcErrorf(c codes.Code, format string, a ...interface{}) error {
	if c == codes.Internal {
		golog.LogDepthf(1, golog.ERR, "Payments - Internal GRPC Error: %s", fmt.Sprintf(format, a...))
	}
	return grpcErrf(c, format, a...)
}

func grpcError(err error) error {
	if grpc.Code(err) == codes.Unknown {
		return grpcErrorf(codes.Internal, err.Error())
	}
	return err
}

func grpcIErrorf(fmt string, args ...interface{}) error {
	golog.LogDepthf(1, golog.ERR, fmt, args...)
	return grpcErrorf(codes.Internal, fmt, args...)
}

var (
	// ErrNotImplemented is returned from RPC calls that have yet to be implemented
	ErrNotImplemented = errors.New("Not Implemented")
)

type server struct {
	dal dal.DAL
}

// New returns an initialized instance of server
func New(dl dal.DAL) (payments.PaymentsServer, error) {
	return &server{
		dal: dl,
	}, nil
}
