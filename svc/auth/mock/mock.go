package mock

import (
	"testing"

	"context"

	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"google.golang.org/grpc"
)

// Compile time check to make sure the mock conforms to the interface
var _ auth.AuthClient = &Client{}

// Client is a mock for the directory service client.
type Client struct {
	*mock.Expector
}

// New returns an initialized Client.
func New(t testing.TB) *Client {
	return &Client{&mock.Expector{T: t}}
}

func (c *Client) AuthenticateLogin(ctx context.Context, in *auth.AuthenticateLoginRequest, opts ...grpc.CallOption) (*auth.AuthenticateLoginResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*auth.AuthenticateLoginResponse), mock.SafeError(rets[1])
}

func (c *Client) GetLastLoginInfo(ctx context.Context, in *auth.GetLastLoginInfoRequest, opts ...grpc.CallOption) (*auth.GetLastLoginInfoResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*auth.GetLastLoginInfoResponse), mock.SafeError(rets[1])
}

func (c *Client) AuthenticateLoginWithCode(ctx context.Context, in *auth.AuthenticateLoginWithCodeRequest, opts ...grpc.CallOption) (*auth.AuthenticateLoginWithCodeResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*auth.AuthenticateLoginWithCodeResponse), mock.SafeError(rets[1])
}

func (c *Client) CheckAuthentication(ctx context.Context, in *auth.CheckAuthenticationRequest, opts ...grpc.CallOption) (*auth.CheckAuthenticationResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*auth.CheckAuthenticationResponse), mock.SafeError(rets[1])
}

func (c *Client) CheckPasswordResetToken(ctx context.Context, in *auth.CheckPasswordResetTokenRequest, opts ...grpc.CallOption) (*auth.CheckPasswordResetTokenResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*auth.CheckPasswordResetTokenResponse), mock.SafeError(rets[1])
}

func (c *Client) CheckVerificationCode(ctx context.Context, in *auth.CheckVerificationCodeRequest, opts ...grpc.CallOption) (*auth.CheckVerificationCodeResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*auth.CheckVerificationCodeResponse), mock.SafeError(rets[1])
}

func (c *Client) CreateAccount(ctx context.Context, in *auth.CreateAccountRequest, opts ...grpc.CallOption) (*auth.CreateAccountResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*auth.CreateAccountResponse), mock.SafeError(rets[1])
}

func (c *Client) CreatePasswordResetToken(ctx context.Context, in *auth.CreatePasswordResetTokenRequest, opts ...grpc.CallOption) (*auth.CreatePasswordResetTokenResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*auth.CreatePasswordResetTokenResponse), mock.SafeError(rets[1])
}

func (c *Client) CreateVerificationCode(ctx context.Context, in *auth.CreateVerificationCodeRequest, opts ...grpc.CallOption) (*auth.CreateVerificationCodeResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*auth.CreateVerificationCodeResponse), mock.SafeError(rets[1])
}

func (c *Client) GetAccount(ctx context.Context, in *auth.GetAccountRequest, opts ...grpc.CallOption) (*auth.GetAccountResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*auth.GetAccountResponse), mock.SafeError(rets[1])
}

func (c *Client) Unauthenticate(ctx context.Context, in *auth.UnauthenticateRequest, opts ...grpc.CallOption) (*auth.UnauthenticateResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*auth.UnauthenticateResponse), mock.SafeError(rets[1])
}

func (c *Client) UpdatePassword(ctx context.Context, in *auth.UpdatePasswordRequest, opts ...grpc.CallOption) (*auth.UpdatePasswordResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*auth.UpdatePasswordResponse), mock.SafeError(rets[1])
}

func (c *Client) VerifiedValue(ctx context.Context, in *auth.VerifiedValueRequest, opts ...grpc.CallOption) (*auth.VerifiedValueResponse, error) {
	rets := c.Expector.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*auth.VerifiedValueResponse), mock.SafeError(rets[1])
}

func (c *Client) BlockAccount(ctx context.Context, in *auth.BlockAccountRequest, opts ...grpc.CallOption) (*auth.BlockAccountResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*auth.BlockAccountResponse), mock.SafeError(rets[1])
}

func (c *Client) UpdateAuthToken(ctx context.Context, in *auth.UpdateAuthTokenRequest, opts ...grpc.CallOption) (*auth.UpdateAuthTokenResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*auth.UpdateAuthTokenResponse), mock.SafeError(rets[1])
}

func (c *Client) DeleteAccount(ctx context.Context, in *auth.DeleteAccountRequest, opts ...grpc.CallOption) (*auth.DeleteAccountResponse, error) {
	rets := c.Record(in)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*auth.DeleteAccountResponse), mock.SafeError(rets[1])
}
