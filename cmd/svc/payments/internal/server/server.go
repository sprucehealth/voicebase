package server

import (
	"fmt"

	"context"

	"github.com/sprucehealth/backend/cmd/svc/payments/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/payments/internal/oauth"
	"github.com/sprucehealth/backend/libs/errors"
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
	dal         dal.DAL
	stripeOAuth oauth.StripeOAuth
}

// New returns an initialized instance of server
func New(dl dal.DAL, stripeSecretKey string) (payments.PaymentsServer, error) {
	return &server{
		dal:         dl,
		stripeOAuth: oauth.NewStripe(stripeSecretKey, ""),
	}, nil
}

func (s *server) ConnectVendorAccount(ctx context.Context, req *payments.ConnectVendorAccountRequest) (*payments.ConnectVendorAccountResponse, error) {
	if req.EntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "EntityID required")
	}
	var vendorAccount *dal.VendorAccount
	switch req.VendorAccountType {
	case payments.VENDOR_ACCOUNT_TYPE_STRIPE:
		stripeReq := req.GetStripeRequest()
		if stripeReq.Code == "" {
			return nil, grpc.Errorf(codes.InvalidArgument, "Code required")
		}
		accessTokenResponse, err := s.stripeOAuth.RequestStripeAccessToken(stripeReq.Code)
		if err != nil {
			return nil, grpcError(err)
		}
		vendorAccount = &dal.VendorAccount{
			AccessToken:        accessTokenResponse.AccessToken,
			RefreshToken:       accessTokenResponse.RefreshToken,
			PublishableKey:     accessTokenResponse.StripePublishableKey,
			ConnectedAccountID: accessTokenResponse.StripeUserID,
			Scope:              accessTokenResponse.Scope,
			Live:               accessTokenResponse.LiveMode,
			AccountType:        dal.VendorAccountAccountTypeStripe,
		}
	default:
		return nil, grpc.Errorf(codes.InvalidArgument, "Unsupported vendor account type %s", req.VendorAccountType)
	}
	// sanity
	if vendorAccount == nil {
		return nil, grpcErrorf(codes.Internal, "nil vendorAccount, this should never happen")
	}
	vendorAccount.Lifecycle = dal.VendorAccountLifecycleConnected
	vendorAccount.ChangeState = dal.VendorAccountChangeStateNone
	vendorAccount.EntityID = req.EntityID
	if _, err := s.dal.InsertVendorAccount(ctx, vendorAccount); err != nil {
		return nil, grpcError(err)
	}

	// Look up the new set of vendor accounts associated with the entity ID now
	vendorAccounts, err := s.VendorAccounts(ctx, &payments.VendorAccountsRequest{EntityID: req.EntityID})
	if err != nil {
		return nil, err
	}
	return &payments.ConnectVendorAccountResponse{
		VendorAccounts: vendorAccounts.VendorAccounts,
	}, nil
}

func (s *server) DisconnectVendorAccount(ctx context.Context, req *payments.DisconnectVendorAccountRequest) (*payments.DisconnectVendorAccountResponse, error) {
	if req.VendorAccountID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "VendorAccountID required")
	}
	vendorAccountID, err := dal.ParseVendorAccountID(req.VendorAccountID)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
	}
	if err := s.dal.Transact(ctx, func(ctx context.Context, dl dal.DAL) error {
		vendorAccount, err := dl.VendorAccount(ctx, vendorAccountID, dal.ForUpdate)
		if errors.Cause(err) == dal.ErrNotFound {
			return grpcErrorf(codes.NotFound, "Vendor Account %s Not Found", vendorAccountID)
		} else if err != nil {
			return grpcError(err)
		}
		// If we're already disconnected then do nothing
		if vendorAccount.Lifecycle == dal.VendorAccountLifecycleDisconnected {
			return nil
		}
		if err := dl.UpdateVendorAccount(ctx, vendorAccountID, &dal.VendorAccountUpdate{
			Lifecycle:   dal.VendorAccountLifecycleDisconnected,
			ChangeState: dal.VendorAccountChangeStatePending,
		}); err != nil {
			return grpcError(err)
		}
		return nil
	}); err != nil {
		return nil, grpcError(err)
	}
	return &payments.DisconnectVendorAccountResponse{}, nil
}

func (s *server) VendorAccounts(ctx context.Context, req *payments.VendorAccountsRequest) (*payments.VendorAccountsResponse, error) {
	if req.EntityID == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "EntityID required")
	}
	vendorAccounts, err := s.dal.EntityVendorAccounts(ctx, req.EntityID)
	if err != nil {
		return nil, grpcError(err)
	}
	return &payments.VendorAccountsResponse{
		VendorAccounts: transformVendorAccountsToResponse(vendorAccounts),
	}, nil
}
