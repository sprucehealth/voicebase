package mock

import (
	"testing"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// Compile time check to make sure the mock conforms to the interface
var _ auth.AuthClient = &Client{}

// Client is a mock for the directory service client.
type Client struct {
	*mock.Expector
}

// New returns an initialized Client.
func New(t *testing.T) *Client {
	return &Client{&mock.Expector{T: t}}
}

func (c *Client) AuthenticateLogin(ctx context.Context, in *auth.AuthenticateLoginRequest, opts ...grpc.CallOption) (*auth.AuthenticateLoginResponse, error) {
	rets := c.Expector.Record(in)
	return rets[0].(*auth.AuthenticateLoginResponse), mock.SafeError(rets[1])
}

func (c *Client) AuthenticateLoginWithCode(ctx context.Context, in *auth.AuthenticateLoginWithCodeRequest, opts ...grpc.CallOption) (*auth.AuthenticateLoginWithCodeResponse, error) {
	rets := c.Expector.Record(in)
	return rets[0].(*auth.AuthenticateLoginWithCodeResponse), mock.SafeError(rets[1])
}

func (c *Client) CheckAuthentication(ctx context.Context, in *auth.CheckAuthenticationRequest, opts ...grpc.CallOption) (*auth.CheckAuthenticationResponse, error) {
	rets := c.Expector.Record(in)
	return rets[0].(*auth.CheckAuthenticationResponse), mock.SafeError(rets[1])
}

func (c *Client) CheckVerificationCode(ctx context.Context, in *auth.CheckVerificationCodeRequest, opts ...grpc.CallOption) (*auth.CheckVerificationCodeResponse, error) {
	rets := c.Expector.Record(in)
	return rets[0].(*auth.CheckVerificationCodeResponse), mock.SafeError(rets[1])
}

func (c *Client) CreateAccount(ctx context.Context, in *auth.CreateAccountRequest, opts ...grpc.CallOption) (*auth.CreateAccountResponse, error) {
	rets := c.Expector.Record(in)
	return rets[0].(*auth.CreateAccountResponse), mock.SafeError(rets[1])
}

func (c *Client) CreateVerificationCode(ctx context.Context, in *auth.CreateVerificationCodeRequest, opts ...grpc.CallOption) (*auth.CreateVerificationCodeResponse, error) {
	rets := c.Expector.Record(in)
	return rets[0].(*auth.CreateVerificationCodeResponse), mock.SafeError(rets[1])
}

func (c *Client) GetAccount(ctx context.Context, in *auth.GetAccountRequest, opts ...grpc.CallOption) (*auth.GetAccountResponse, error) {
	rets := c.Expector.Record(in)
	return rets[0].(*auth.GetAccountResponse), mock.SafeError(rets[1])
}

func (c *Client) Unauthenticate(ctx context.Context, in *auth.UnauthenticateRequest, opts ...grpc.CallOption) (*auth.UnauthenticateResponse, error) {
	rets := c.Expector.Record(in)
	return rets[0].(*auth.UnauthenticateResponse), mock.SafeError(rets[1])
}

func (c *Client) VerifiedValue(ctx context.Context, in *auth.VerifiedValueRequest, opts ...grpc.CallOption) (*auth.VerifiedValueResponse, error) {
	rets := c.Expector.Record(in)
	return rets[0].(*auth.VerifiedValueResponse), mock.SafeError(rets[1])
}
