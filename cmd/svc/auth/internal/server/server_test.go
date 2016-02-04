package server

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"golang.org/x/net/context"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/cmd/svc/auth/internal/dal"
	mock_dal "github.com/sprucehealth/backend/cmd/svc/auth/internal/dal/test"
	authSetting "github.com/sprucehealth/backend/cmd/svc/auth/internal/settings"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/hash"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/svc/settings"
	mock_settings "github.com/sprucehealth/backend/svc/settings/mock"
	"github.com/sprucehealth/backend/test"
)

const (
	clientEncryptionSecret = "test-seekrit"
)

func init() {
	conc.Testing = true
}

func TestGetAccount(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	aID1, err := dal.NewAccountID()
	test.OK(t, err)
	fn, ln := "bat", "man"
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Account, aID1), &dal.Account{
		ID:        aID1,
		FirstName: fn,
		LastName:  ln,
	}, nil))
	resp, err := s.GetAccount(context.Background(), &auth.GetAccountRequest{AccountID: aID1.String()})
	test.OK(t, err)

	test.AssertNotNil(t, resp.Account)
	test.Equals(t, aID1.String(), resp.Account.ID)
	test.Equals(t, fn, resp.Account.FirstName)
	test.Equals(t, ln, resp.Account.LastName)
}

func TestGetAccountNotFound(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	aID1, err := dal.NewAccountID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Account, aID1), (*dal.Account)(nil), api.ErrNotFound("not found")))
	_, err = s.GetAccount(context.Background(), &auth.GetAccountRequest{AccountID: aID1.String()})
	test.Assert(t, err != nil, "Expected an error")
	test.Equals(t, codes.NotFound, grpc.Code(err))
}

func TestAuthenticateLogin2FA(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	hasher := hash.NewBcryptHasher(bCryptHashCost)
	email := "test@email.com"
	password := "password"
	hashedPassword, err := hasher.GenerateFromPassword([]byte(password))
	test.OK(t, err)
	aID1, err := dal.NewAccountID()
	test.OK(t, err)
	apID1, err := dal.NewAccountPhoneID()
	test.OK(t, err)
	phoneNumber := "+1234567890"
	dl.Expect(mock.NewExpectation(dl.AccountForEmail, email).WithReturns(&dal.Account{ID: aID1, Password: hashedPassword, PrimaryAccountPhoneID: apID1}, nil))
	dl.Expect(mock.NewExpectation(dl.AccountPhone, apID1).WithReturns(&dal.AccountPhone{PhoneNumber: phoneNumber}, nil))

	settingsMock.Expect(mock.NewExpectation(settingsMock.GetValues, &settings.GetValuesRequest{
		NodeID: aID1.String(),
		Keys: []*settings.ConfigKey{
			{
				Key: authSetting.ConfigKey2FAEnabled,
			},
		},
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Key: &settings.ConfigKey{
					Key: authSetting.ConfigKey2FAEnabled,
				},
				Value: &settings.Value_Boolean{
					Boolean: &settings.BooleanValue{
						Value: true,
					},
				},
			},
		},
	}, nil))

	resp, err := s.AuthenticateLogin(context.Background(), &auth.AuthenticateLoginRequest{
		Email:           email,
		Password:        password,
		TokenAttributes: map[string]string{"test": "attribute"},
	})
	test.OK(t, err)

	test.AssertNil(t, resp.Token)
	test.AssertNotNil(t, resp.Account)
	test.Assert(t, resp.TwoFactorRequired, "Expected two factor to be required")
	test.Equals(t, resp.TwoFactorPhoneNumber, phoneNumber)
}

func TestAuthenticateLogin2FA_Disabled(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	mclock := clock.NewManaged(time.Now())
	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	svr, _ := s.(*server)
	svr.clk = mclock

	// token := "123abc"
	// expires := mclock.Now().Add(defaultTokenExpiration)

	hasher := hash.NewBcryptHasher(bCryptHashCost)
	email := "test@email.com"
	password := "password"
	hashedPassword, err := hasher.GenerateFromPassword([]byte(password))
	test.OK(t, err)
	aID1, err := dal.NewAccountID()
	test.OK(t, err)
	apID1, err := dal.NewAccountPhoneID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.AccountForEmail, email).WithReturns(&dal.Account{ID: aID1, Password: hashedPassword, PrimaryAccountPhoneID: apID1}, nil))
	dl.Expect(mock.NewExpectationFn(dl.InsertAuthToken, func(p ...interface{}) {
		test.Assert(t, len(p) == 1, "Expected only 1 param to be provided")
		authToken, ok := p[0].(*dal.AuthToken)
		test.Assert(t, ok, "Expected *dal.AuthToken")
		test.Assert(t, len(authToken.Token) != 0, "Expected non empty token")
		test.Assert(t, authToken.Expires.Unix() > time.Now().Unix(), "Expected token to expire in the future")
		// token = string(authToken.Token)
		// expires = uint64(authToken.Expires.Unix())
	}).WithReturns(int64(1), nil))
	settingsMock.Expect(mock.NewExpectation(settingsMock.GetValues, &settings.GetValuesRequest{
		NodeID: aID1.String(),
		Keys: []*settings.ConfigKey{
			{
				Key: authSetting.ConfigKey2FAEnabled,
			},
		},
	}).WithReturns(&settings.GetValuesResponse{
		Values: []*settings.Value{
			{
				Key: &settings.ConfigKey{
					Key: authSetting.ConfigKey2FAEnabled,
				},
				Value: &settings.Value_Boolean{
					Boolean: &settings.BooleanValue{
						Value: false,
					},
				},
			},
		},
	}, nil))

	resp, err := s.AuthenticateLogin(context.Background(), &auth.AuthenticateLoginRequest{
		Email:           email,
		Password:        password,
		TokenAttributes: map[string]string{"test": "attribute"},
	})
	test.OK(t, err)

	test.AssertNotNil(t, resp.Token)
	test.AssertNotNil(t, resp.Account)
	test.Assert(t, !resp.TwoFactorRequired, "Expected two factor to be required")
	test.Equals(t, "", resp.TwoFactorPhoneNumber)
}

func TestAuthenticateLoginWithCode(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	mClock := clock.NewManaged(time.Now())
	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	svr, ok := s.(*server)
	test.Assert(t, ok, "Expected a *server")
	svr.clk = mClock
	s = svr
	token := "123abc"
	code := "123456"
	aID1, err := dal.NewAccountID()
	test.OK(t, err)
	var expires uint64
	signer, err := sig.NewSigner([][]byte{[]byte(clientEncryptionSecret)}, sha256.New)
	test.OK(t, err)
	dl.Expect(mock.NewExpectation(dl.VerificationCode, token).WithReturns(&dal.VerificationCode{
		Token:            token,
		Code:             code,
		Expires:          time.Unix(mClock.Now().Unix()+1000, 0),
		VerifiedValue:    aID1.String(),
		VerificationType: dal.VerificationCodeTypeAccount2fa,
	}, nil))
	dl.Expect(mock.NewExpectation(dl.UpdateVerificationCode,
		token, &dal.VerificationCodeUpdate{Consumed: ptr.Bool(true)}).WithReturns(int64(1), nil))
	dl.Expect(mock.NewExpectationFn(dl.InsertAuthToken, func(p ...interface{}) {
		test.Assert(t, len(p) == 1, "Expected only 1 param to be provided")
		authToken, ok := p[0].(*dal.AuthToken)
		test.Assert(t, ok, "Expected *dal.AuthToken")
		test.Assert(t, len(authToken.Token) != 0, "Expected non empty token")
		test.Assert(t, authToken.Expires.Unix() > time.Now().Unix(), "Expected token to expire in the future")
		token = string(authToken.Token)
		expires = uint64(authToken.Expires.Unix())
	}).WithReturns(int64(1), nil))
	dl.Expect(mock.NewExpectation(dl.Account, aID1).WithReturns(&dal.Account{
		ID:        aID1,
		FirstName: "Bat",
		LastName:  "Wayne",
	}, nil))
	resp, err := s.AuthenticateLoginWithCode(context.Background(), &auth.AuthenticateLoginWithCodeRequest{
		Token: token,
		Code:  code,
	})
	key, err := signer.Sign([]byte(token))
	test.OK(t, err)

	test.OK(t, err)
	test.AssertNotNil(t, resp.Token)
	test.Equals(t, &auth.AuthToken{
		Value:               token,
		ExpirationEpoch:     expires,
		ClientEncryptionKey: base64.StdEncoding.EncodeToString(key),
	}, resp.Token)
	test.AssertNotNil(t, resp.Account)
	test.Equals(t, resp.Account.ID, aID1.String())
	test.Equals(t, resp.Account.FirstName, "Bat")
	test.Equals(t, resp.Account.LastName, "Wayne")
}

func TestAuthenticateLoginWithCodeNot2FA(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	mClock := clock.NewManaged(time.Now())
	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	svr, ok := s.(*server)
	test.Assert(t, ok, "Expected a *server")
	svr.clk = mClock
	s = svr
	token := "123abc"
	code := "123456"
	aID1, err := dal.NewAccountID()
	test.OK(t, err)
	dl.Expect(mock.NewExpectation(dl.VerificationCode, token).WithReturns(&dal.VerificationCode{
		Token:            token,
		Code:             code,
		Expires:          time.Unix(mClock.Now().Unix()+1000, 0),
		VerifiedValue:    aID1.String(),
		VerificationType: dal.VerificationCodeTypePhone,
	}, nil))

	resp, err := s.AuthenticateLoginWithCode(context.Background(), &auth.AuthenticateLoginWithCodeRequest{
		Token: token,
		Code:  code,
	})
	test.AssertNil(t, resp)
	test.Equals(t, codes.NotFound, grpc.Code(err))
}

func TestAuthenticateLoginWithCodeBadCode(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	mClock := clock.NewManaged(time.Now())
	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	svr, ok := s.(*server)
	test.Assert(t, ok, "Expected a *server")
	svr.clk = mClock
	s = svr
	token := "123abc"
	code := "123456"
	aID1, err := dal.NewAccountID()
	test.OK(t, err)
	dl.Expect(mock.NewExpectation(dl.VerificationCode, token).WithReturns(&dal.VerificationCode{
		Token:            token,
		Code:             code + "1",
		Expires:          time.Unix(mClock.Now().Unix()+1000, 0),
		VerifiedValue:    aID1.String(),
		VerificationType: dal.VerificationCodeTypeAccount2fa,
	}, nil))

	resp, err := s.AuthenticateLoginWithCode(context.Background(), &auth.AuthenticateLoginWithCodeRequest{
		Token: token,
		Code:  code,
	})
	test.AssertNil(t, resp)
	test.Equals(t, auth.BadVerificationCode, grpc.Code(err))
}

func TestAuthenticateLoginNoEmail(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	email := "test@email.com"
	password := "password"
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.AccountForEmail, email), (*dal.Account)(nil), api.ErrNotFound("not found")))
	_, err = s.AuthenticateLogin(context.Background(), &auth.AuthenticateLoginRequest{
		Email:           email,
		Password:        password,
		TokenAttributes: map[string]string{"test": "attribute"},
	})
	test.Assert(t, err != nil, "Expected an error")

	test.Equals(t, auth.EmailNotFound, grpc.Code(err))
}

func TestAuthenticateBadPassword(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	email := "test@email.com"
	password := "password"
	aID1, err := dal.NewAccountID()
	test.OK(t, err)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.AccountForEmail, email), &dal.Account{ID: aID1, Password: []byte("notpassword")}, nil))
	_, err = s.AuthenticateLogin(context.Background(), &auth.AuthenticateLoginRequest{
		Email:           email,
		Password:        password,
		TokenAttributes: map[string]string{"test": "attribute"},
	})
	test.Assert(t, err != nil, "Expected an error")

	test.Equals(t, auth.BadPassword, grpc.Code(err))
}

func TestCheckAuthentication(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	mClock := clock.NewManaged(time.Now())
	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	svr, ok := s.(*server)
	test.Assert(t, ok, "Expected a *server")
	svr.clk = mClock
	s = svr
	token := "123abc"
	tokenAttributes := map[string]string{"token": "attribute"}
	aID1, err := dal.NewAccountID()
	test.OK(t, err)
	expires := mClock.Now().Add(defaultTokenExpiration)
	signer, err := sig.NewSigner([][]byte{[]byte(clientEncryptionSecret)}, sha256.New)
	test.OK(t, err)
	dl.Expect(mock.NewExpectation(dl.AuthToken, token+":tokenattribute", mClock.Now()).WithReturns(&dal.AuthToken{
		Token:     []byte(token + ":tokenattribute"),
		AccountID: aID1,
		Expires:   expires,
	}, nil))
	dl.Expect(mock.NewExpectation(dl.Account, aID1).WithReturns(&dal.Account{
		ID:        aID1,
		FirstName: "bat",
		LastName:  "man",
	}, nil))
	resp, err := s.CheckAuthentication(context.Background(), &auth.CheckAuthenticationRequest{
		Token:           token,
		TokenAttributes: tokenAttributes,
	})
	test.OK(t, err)
	key, err := signer.Sign([]byte(token))
	test.OK(t, err)

	test.Assert(t, resp.IsAuthenticated, "Expected authentication")
	test.AssertNotNil(t, resp.Account)
	test.AssertNotNil(t, resp.Token)
	test.Equals(t, &auth.Account{
		ID:        aID1.String(),
		FirstName: "bat",
		LastName:  "man",
	}, resp.Account)
	test.Equals(t, &auth.AuthToken{
		Value:               token,
		ExpirationEpoch:     uint64(expires.Unix()),
		ClientEncryptionKey: base64.StdEncoding.EncodeToString(key),
	}, resp.Token)
}

func TestCheckVerificationTokenBadCode(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	mClock := clock.NewManaged(time.Now())
	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	svr, ok := s.(*server)
	test.Assert(t, ok, "Expected a *server")
	svr.clk = mClock
	s = svr
	token := "123abc"
	code := "123456"

	dl.Expect(mock.NewExpectation(dl.VerificationCode, token).WithReturns(&dal.VerificationCode{
		Token: token,
		Code:  code + "1",
	}, nil))

	resp, err := s.CheckVerificationCode(context.Background(), &auth.CheckVerificationCodeRequest{
		Token: token,
		Code:  code,
	})
	test.AssertNil(t, resp)
	test.Assert(t, grpc.Code(err) == auth.BadVerificationCode, "Expected BadVerificationCode error")
}

func TestCheckVerificationTokenExpired(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	mClock := clock.NewManaged(time.Now())
	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	svr, ok := s.(*server)
	test.Assert(t, ok, "Expected a *server")
	svr.clk = mClock
	s = svr
	token := "123abc"
	code := "123456"

	dl.Expect(mock.NewExpectation(dl.VerificationCode, token).WithReturns(&dal.VerificationCode{
		Token:   token,
		Code:    code,
		Expires: mClock.Now(),
	}, nil))
	mClock.WarpForward(10 * time.Minute)
	resp, err := s.CheckVerificationCode(context.Background(), &auth.CheckVerificationCodeRequest{
		Token: token,
		Code:  code,
	})
	test.AssertNil(t, resp)
	test.Assert(t, grpc.Code(err) == auth.VerificationCodeExpired, "Expected VerificationCodeExpired error")
}

func TestCheckVerificationPhone(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	mClock := clock.NewManaged(time.Now())
	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	svr, ok := s.(*server)
	test.Assert(t, ok, "Expected a *server")
	svr.clk = mClock
	s = svr
	token := "123abc"
	code := "123456"
	value := "+1234567890"

	dl.Expect(mock.NewExpectation(dl.VerificationCode, token).WithReturns(&dal.VerificationCode{
		Token:            token,
		Code:             code,
		Expires:          time.Unix(mClock.Now().Unix()+1000, 0),
		VerifiedValue:    value,
		VerificationType: dal.VerificationCodeTypePhone,
	}, nil))
	dl.Expect(mock.NewExpectation(dl.UpdateVerificationCode,
		token, &dal.VerificationCodeUpdate{Consumed: ptr.Bool(true)}).WithReturns(int64(1), nil))

	resp, err := s.CheckVerificationCode(context.Background(), &auth.CheckVerificationCodeRequest{
		Token: token,
		Code:  code,
	})
	test.OK(t, err)
	test.AssertNil(t, resp.Account)
	test.Equals(t, resp.Value, value)
}

func TestCheckVerificationAccount2FA(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	mClock := clock.NewManaged(time.Now())
	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	svr, ok := s.(*server)
	test.Assert(t, ok, "Expected a *server")
	svr.clk = mClock
	s = svr
	token := "123abc"
	code := "123456"
	aID1, err := dal.NewAccountID()
	test.OK(t, err)
	dl.Expect(mock.NewExpectation(dl.VerificationCode, token).WithReturns(&dal.VerificationCode{
		Token:            token,
		Code:             code,
		Expires:          time.Unix(mClock.Now().Unix()+1000, 0),
		VerifiedValue:    aID1.String(),
		VerificationType: dal.VerificationCodeTypeAccount2fa,
	}, nil))
	dl.Expect(mock.NewExpectation(dl.UpdateVerificationCode,
		token, &dal.VerificationCodeUpdate{Consumed: ptr.Bool(true)}).WithReturns(int64(1), nil))
	dl.Expect(mock.NewExpectation(dl.Account, aID1).WithReturns(&dal.Account{
		ID:        aID1,
		FirstName: "Bat",
		LastName:  "Wayne",
	}, nil))

	resp, err := s.CheckVerificationCode(context.Background(), &auth.CheckVerificationCodeRequest{
		Token: token,
		Code:  code,
	})
	test.OK(t, err)
	test.AssertNotNil(t, resp.Account)
	test.Equals(t, resp.Account.ID, aID1.String())
	test.Equals(t, resp.Account.FirstName, "Bat")
	test.Equals(t, resp.Account.LastName, "Wayne")
	test.Equals(t, resp.Value, aID1.String())
}

func TestCheckAuthenticationRefresh(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	mClock := clock.NewManaged(time.Now())
	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	svr, ok := s.(*server)
	test.Assert(t, ok, "Expected a *server")
	svr.clk = mClock
	s = svr
	token := "123abc"
	tokenAttributes := map[string]string{"token": "attribute"}
	aID1, err := dal.NewAccountID()
	test.OK(t, err)
	signer, err := sig.NewSigner([][]byte{[]byte(clientEncryptionSecret)}, sha256.New)
	test.OK(t, err)
	expires := mClock.Now().Add(defaultTokenExpiration)
	var refreshedExpiration time.Time
	dl.Expect(mock.NewExpectation(dl.AuthToken, token+":tokenattribute", mClock.Now()).WithReturns(&dal.AuthToken{
		Token:     []byte(token + ":tokenattribute"),
		AccountID: aID1,
		Expires:   expires,
	}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectationFn(dl.InsertAuthToken, func(p ...interface{}) {
		test.Equals(t, 1, len(p))
		at, ok := p[0].(*dal.AuthToken)
		test.Assert(t, ok, "Expected *dal.AuthToken")
		test.Assert(t, strings.HasSuffix(string(at.Token), ":tokenattribute"), "Expected auth token to have attribute suffix %s, got: %s", ":tokenattribute", at.Token)
		test.Assert(t, at.Expires.Unix() >= time.Now().Unix(), "Expected expiration token to be in the future but was %v", at.Expires)
		test.Assert(t, at.AccountID.String() == aID1.String(), "Expected auth token to map to account id %s, but got %s", aID1.String(), at.AccountID.String())
		token = strings.Split(string(at.Token), ":")[0]
		refreshedExpiration = at.Expires
	}), nil))
	dl.Expect(mock.NewExpectation(dl.DeleteAuthToken, "123abc:tokenattribute").WithReturns(int64(1), nil))
	dl.Expect(mock.NewExpectation(dl.Account, aID1).WithReturns(&dal.Account{
		ID:        aID1,
		FirstName: "bat",
		LastName:  "man",
	}, nil))
	resp, err := s.CheckAuthentication(context.Background(), &auth.CheckAuthenticationRequest{
		Token:           token,
		TokenAttributes: tokenAttributes,
		Refresh:         true,
	})
	test.OK(t, err)
	key, err := signer.Sign([]byte(token))
	test.OK(t, err)

	test.Assert(t, resp.IsAuthenticated, "Expected authentication")
	test.AssertNotNil(t, resp.Account)
	test.AssertNotNil(t, resp.Token)
	test.Equals(t, &auth.Account{
		ID:        aID1.String(),
		FirstName: "bat",
		LastName:  "man",
	}, resp.Account)
	test.Equals(t, &auth.AuthToken{
		Value:               token,
		ExpirationEpoch:     uint64(refreshedExpiration.Unix()),
		ClientEncryptionKey: base64.StdEncoding.EncodeToString(key),
	}, resp.Token)
}

func TestCheckAuthenticationNoToken(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	mClock := clock.NewManaged(time.Now())
	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	svr, ok := s.(*server)
	test.Assert(t, ok, "Expected a *server")
	svr.clk = mClock
	s = svr
	token := "123abc"
	tokenAttributes := map[string]string{"token": "attribute"}
	dl.Expect(mock.NewExpectation(dl.AuthToken, token+":tokenattribute", mClock.Now()).WithReturns((*dal.AuthToken)(nil), api.ErrNotFound("not found")))
	resp, err := s.CheckAuthentication(context.Background(), &auth.CheckAuthenticationRequest{
		Token:           token,
		TokenAttributes: tokenAttributes,
	})
	test.OK(t, err)

	test.Equals(t, false, resp.IsAuthenticated)
	test.AssertNil(t, resp.Account)
	test.AssertNil(t, resp.Token)
}

func TestCreateAccount(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	fn := "bat"
	ln := "man"
	email := "bat@man.com"
	phoneNumber := "+12345678910"
	password := "password"
	hasher := hash.NewBcryptHasher(bCryptHashCost)
	aID1, err := dal.NewAccountID()
	test.OK(t, err)
	aEID1, err := dal.NewAccountEmailID()
	test.OK(t, err)
	aPID1, err := dal.NewAccountPhoneID()
	test.OK(t, err)
	signer, err := sig.NewSigner([][]byte{[]byte(clientEncryptionSecret)}, sha256.New)
	test.OK(t, err)
	dl.Expect(mock.NewExpectationFn(dl.InsertAccount, func(p ...interface{}) {
		test.Equals(t, 1, len(p))
		account, ok := p[0].(*dal.Account)
		test.Assert(t, ok, "Expected *dal.Account but got %T", p[0])
		test.Equals(t, fn, account.FirstName)
		test.Equals(t, ln, account.LastName)
		test.Equals(t, dal.AccountStatusActive, account.Status)
		test.OK(t, hasher.CompareHashAndPassword(account.Password, []byte(password)))
	}).WithReturns(aID1, nil))
	dl.Expect(mock.NewExpectation(dl.InsertAccountEmail, &dal.AccountEmail{
		AccountID: aID1,
		Email:     email,
		Status:    dal.AccountEmailStatusActive,
		Verified:  false,
	}).WithReturns(aEID1, nil))
	dl.Expect(mock.NewExpectation(dl.InsertAccountPhone, &dal.AccountPhone{
		AccountID:   aID1,
		PhoneNumber: phoneNumber,
		Status:      dal.AccountPhoneStatusActive,
		Verified:    false,
	}).WithReturns(aPID1, nil))
	dl.Expect(mock.NewExpectation(dl.UpdateAccount, aID1, &dal.AccountUpdate{
		PrimaryAccountPhoneID: aPID1,
		PrimaryAccountEmailID: aEID1,
	}).WithReturns(int64(1), nil))
	var expiration uint64
	var token string
	dl.Expect(mock.NewExpectationFn(dl.InsertAuthToken, func(p ...interface{}) {
		test.Equals(t, 1, len(p))
		at, ok := p[0].(*dal.AuthToken)
		test.Assert(t, ok, "Expected *dal.AuthToken but got %T", p[0])
		test.Assert(t, len(at.Token) != 0, "Expected a non empty token")
		test.Assert(t, at.Expires.Unix() >= time.Now().Unix(), "Expected expiration token to be in the future but was %v", at.Expires)
		test.Assert(t, at.AccountID.String() == aID1.String(), "Expected auth token to map to account id %s, but got %s", aID1.String(), at.AccountID.String())
		token = string(at.Token)
		expiration = uint64(at.Expires.Unix())
	}))
	dl.Expect(mock.NewExpectation(dl.Account, aID1).WithReturns(&dal.Account{
		ID:        aID1,
		FirstName: fn,
		LastName:  ln,
	}, nil))
	resp, err := s.CreateAccount(context.Background(), &auth.CreateAccountRequest{
		FirstName:   fn,
		LastName:    ln,
		PhoneNumber: phoneNumber,
		Email:       email,
		Password:    password,
	})
	test.OK(t, err)
	key, err := signer.Sign([]byte(token))
	test.OK(t, err)

	test.AssertNotNil(t, resp.Token)
	test.AssertNotNil(t, resp.Account)
	test.Equals(t, &auth.Account{
		ID:        aID1.String(),
		FirstName: "bat",
		LastName:  "man",
	}, resp.Account)
	test.Equals(t, &auth.AuthToken{
		Value:               token,
		ExpirationEpoch:     expiration,
		ClientEncryptionKey: base64.StdEncoding.EncodeToString(key),
	}, resp.Token)
}

func TestCreateAccountMissingData(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	fn := "bat"
	ln := "man"
	email := "bat@man.com"
	phoneNumber := "+12345678910"
	password := "password"
	inputs := []*auth.CreateAccountRequest{
		&auth.CreateAccountRequest{
			FirstName:   "",
			LastName:    ln,
			PhoneNumber: phoneNumber,
			Email:       email,
			Password:    password,
		},
		&auth.CreateAccountRequest{
			FirstName:   fn,
			LastName:    "",
			PhoneNumber: phoneNumber,
			Email:       email,
			Password:    password,
		},
		&auth.CreateAccountRequest{
			FirstName:   fn,
			LastName:    ln,
			PhoneNumber: "",
			Email:       email,
			Password:    password,
		},
		&auth.CreateAccountRequest{
			FirstName:   fn,
			LastName:    ln,
			PhoneNumber: phoneNumber,
			Email:       "",
			Password:    password,
		},
		&auth.CreateAccountRequest{
			FirstName:   fn,
			LastName:    ln,
			PhoneNumber: phoneNumber,
			Email:       email,
			Password:    "",
		},
	}
	for _, r := range inputs {
		_, err := s.CreateAccount(context.Background(), r)
		test.Assert(t, err != nil, "Expected an error")

		test.Equals(t, codes.InvalidArgument, grpc.Code(err))
	}
}

func TestCreateAccountBadEmail(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	fn := "bat"
	ln := "man"
	email := "notarealemail"
	phoneNumber := "+12345678910"
	password := "password"
	_, err = s.CreateAccount(context.Background(), &auth.CreateAccountRequest{
		FirstName:   fn,
		LastName:    ln,
		PhoneNumber: phoneNumber,
		Email:       email,
		Password:    password,
	})
	test.Assert(t, err != nil, "Expected an error")

	test.Equals(t, auth.InvalidEmail, grpc.Code(err))
}

func TestCreateVerificationCode(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	value := "myValue"
	var code string
	var token string
	var expires uint64

	dl.Expect(mock.NewExpectationFn(dl.InsertVerificationCode, func(p ...interface{}) {
		test.Assert(t, len(p) == 1, "Expected only 1 argument")
		vc, ok := p[0].(*dal.VerificationCode)
		test.Assert(t, ok, "Expected *dal.VerificationCode")
		test.Assert(t, vc.Token != "", "Expected a non empty token")
		test.Assert(t, vc.Code != "", "Expected a non empty code")
		test.Assert(t, vc.Expires.Unix() > time.Now().Unix(), "Expected a code that expires in the future")
		test.Equals(t, dal.VerificationCodeTypePhone, vc.VerificationType)
		test.Equals(t, value, vc.VerifiedValue)
		code = vc.Code
		token = vc.Token
		expires = uint64(vc.Expires.Unix())
	}))

	resp, err := s.CreateVerificationCode(context.Background(), &auth.CreateVerificationCodeRequest{
		Type:          auth.VerificationCodeType_PHONE,
		ValueToVerify: value,
	})
	test.OK(t, err)
	test.AssertNotNil(t, resp.VerificationCode)
	test.Equals(t, token, resp.VerificationCode.Token)
	test.Equals(t, code, resp.VerificationCode.Code)
	test.Equals(t, expires, resp.VerificationCode.ExpirationEpoch)
	test.Equals(t, auth.VerificationCodeType_PHONE, resp.VerificationCode.Type)
}

func TestVerifiedValue(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	token := "123abc"
	value := "myValue"

	dl.Expect(mock.NewExpectation(dl.VerificationCode, token).WithReturns(&dal.VerificationCode{
		VerifiedValue: value,
		Consumed:      true,
	}, nil))

	resp, err := s.VerifiedValue(context.Background(), &auth.VerifiedValueRequest{
		Token: token,
	})
	test.OK(t, err)
	test.Equals(t, value, resp.Value)
}

func TestVerifiedValueNotFound(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	token := "123abc"

	dl.Expect(mock.NewExpectation(dl.VerificationCode, token).WithReturns((*dal.VerificationCode)(nil), api.ErrNotFound("foo")))

	resp, err := s.VerifiedValue(context.Background(), &auth.VerifiedValueRequest{
		Token: token,
	})
	test.AssertNil(t, resp)
	test.Equals(t, grpc.Code(err), codes.NotFound)
}

func TestVerifiedValueNotYetVerified(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()

	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()

	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	token := "123abc"

	dl.Expect(mock.NewExpectation(dl.VerificationCode, token).WithReturns(&dal.VerificationCode{
		Consumed: false,
	}, nil))

	resp, err := s.VerifiedValue(context.Background(), &auth.VerifiedValueRequest{
		Token: token,
	})
	test.AssertNil(t, resp)
	test.Equals(t, grpc.Code(err), auth.ValueNotYetVerified)
}

func TestCheckPasswordResetToken(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()
	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	token := "123abc"
	aID1, err := dal.NewAccountID()
	test.OK(t, err)
	apID1, err := dal.NewAccountPhoneID()
	test.OK(t, err)
	aeID1, err := dal.NewAccountEmailID()
	test.OK(t, err)
	phoneNumber := "+1234567890"
	email := "test@test.com"

	dl.Expect(mock.NewExpectation(dl.VerificationCode, token).WithReturns(&dal.VerificationCode{
		Consumed:         false,
		VerificationType: dal.VerificationCodeTypePasswordReset,
		Expires:          time.Unix(time.Now().Unix()+10000, 0),
		VerifiedValue:    aID1.String(),
	}, nil))

	dl.Expect(mock.NewExpectation(dl.Account, aID1).WithReturns(&dal.Account{
		ID: aID1,
		PrimaryAccountPhoneID: apID1,
		PrimaryAccountEmailID: aeID1,
	}, nil))

	dl.Expect(mock.NewExpectation(dl.UpdateVerificationCode,
		token, &dal.VerificationCodeUpdate{Consumed: ptr.Bool(true)}).WithReturns(int64(1), nil))

	dl.Expect(mock.NewExpectation(dl.AccountPhone, apID1).WithReturns(&dal.AccountPhone{PhoneNumber: phoneNumber}, nil))
	dl.Expect(mock.NewExpectation(dl.AccountEmail, aeID1).WithReturns(&dal.AccountEmail{Email: email}, nil))

	resp, err := s.CheckPasswordResetToken(context.Background(), &auth.CheckPasswordResetTokenRequest{
		Token: token,
	})
	test.OK(t, err)
	test.Equals(t, aID1.String(), resp.AccountID)
	test.Equals(t, phoneNumber, resp.AccountPhoneNumber)
	test.Equals(t, email, resp.AccountEmail)
}

func TestCheckPasswordResetTokenNotFound(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()
	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	token := "123abc"

	dl.Expect(mock.NewExpectation(dl.VerificationCode, token).WithReturns((*dal.VerificationCode)(nil), api.ErrNotFound("foo")))

	resp, err := s.CheckPasswordResetToken(context.Background(), &auth.CheckPasswordResetTokenRequest{
		Token: token,
	})
	test.AssertNil(t, resp)
	test.Equals(t, codes.NotFound, grpc.Code(err))
}

func TestCheckPasswordResetTokenWrongType(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()
	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	token := "123abc"
	aID1, err := dal.NewAccountID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.VerificationCode, token).WithReturns(&dal.VerificationCode{
		Consumed:         false,
		VerificationType: dal.VerificationCodeTypePhone,
		Expires:          time.Unix(time.Now().Unix()+10000, 0),
		VerifiedValue:    aID1.String(),
	}, nil))

	resp, err := s.CheckPasswordResetToken(context.Background(), &auth.CheckPasswordResetTokenRequest{
		Token: token,
	})
	test.AssertNil(t, resp)
	test.Equals(t, codes.InvalidArgument, grpc.Code(err))
}

func TestCheckPasswordResetTokenExpired(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()
	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	token := "123abc"
	aID1, err := dal.NewAccountID()
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.VerificationCode, token).WithReturns(&dal.VerificationCode{
		Consumed:         false,
		VerificationType: dal.VerificationCodeTypePasswordReset,
		Expires:          time.Unix(0, 0),
		VerifiedValue:    aID1.String(),
	}, nil))

	resp, err := s.CheckPasswordResetToken(context.Background(), &auth.CheckPasswordResetTokenRequest{
		Token: token,
	})
	test.AssertNil(t, resp)
	test.Equals(t, auth.TokenExpired, grpc.Code(err))
}

func TestCreatePasswordResetToken(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()
	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	aID1, err := dal.NewAccountID()
	test.OK(t, err)
	email := "test@test.com"
	var token string

	dl.Expect(mock.NewExpectation(dl.AccountForEmail, email).WithReturns(&dal.Account{
		ID: aID1,
	}, nil))

	dl.Expect(mock.NewExpectationFn(dl.InsertVerificationCode, func(p ...interface{}) {
		test.Assert(t, len(p) == 1, "Expected only 1 argument")
		vc, ok := p[0].(*dal.VerificationCode)
		test.Assert(t, ok, "Expected *dal.VerificationCode")
		test.Assert(t, vc.Token != "", "Expected a non empty token")
		test.Assert(t, vc.Code != "", "Expected a non empty code")
		test.Assert(t, vc.Expires.Unix() > time.Now().Unix(), "Expected a code that expires in the future")
		test.Equals(t, dal.VerificationCodeTypePasswordReset, vc.VerificationType)
		test.Equals(t, aID1.String(), vc.VerifiedValue)
		token = vc.Token
	}))

	resp, err := s.CreatePasswordResetToken(context.Background(), &auth.CreatePasswordResetTokenRequest{
		Email: email,
	})
	test.OK(t, err)
	test.Equals(t, token, resp.Token)
}

func TestCreatePasswordResetTokenEmailNotFound(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()
	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	email := "test@test.com"

	dl.Expect(mock.NewExpectation(dl.AccountForEmail, email).WithReturns((*dal.Account)(nil), api.ErrNotFound("foo")))

	resp, err := s.CreatePasswordResetToken(context.Background(), &auth.CreatePasswordResetTokenRequest{
		Email: email,
	})
	test.AssertNil(t, resp)
	test.Equals(t, codes.NotFound, grpc.Code(err))
}

// Test the password updating functionality
func TestUpdatePassword(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()
	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	aID1, err := dal.NewAccountID()
	test.OK(t, err)
	token := "123abc"
	code := "123456"
	newPassword := "newPassword"
	test.OK(t, err)
	hasher := hash.NewBcryptHasher(bCryptHashCost)

	dl.Expect(mock.NewExpectation(dl.VerificationCode, token).WithReturns(&dal.VerificationCode{
		Code:          code,
		Expires:       time.Unix(time.Now().Unix()+10000, 0),
		VerifiedValue: aID1.String(),
	}, nil))

	dl.Expect(mock.NewExpectationFn(dl.UpdateAccount, func(p ...interface{}) {
		test.Assert(t, len(p) == 2, "Expected 2 arguments")
		accID, ok := p[0].(dal.AccountID)
		test.Assert(t, ok, "Expected dal.AccountID")
		test.Equals(t, aID1, accID)
		acc, ok := p[1].(*dal.AccountUpdate)
		test.Assert(t, ok, "Expected *dal.AccountUpdate")
		test.AssertNotNil(t, acc.Password)
		test.OK(t, hasher.CompareHashAndPassword(*acc.Password, []byte(newPassword)))
	}))
	dl.Expect(mock.NewExpectation(dl.UpdateVerificationCode,
		token, &dal.VerificationCodeUpdate{Consumed: ptr.Bool(true)}).WithReturns(int64(1), nil))
	dl.Expect(mock.NewExpectation(dl.DeleteAuthTokens, aID1))

	resp, err := s.UpdatePassword(context.Background(), &auth.UpdatePasswordRequest{
		Token:       token,
		Code:        code,
		NewPassword: newPassword,
	})
	test.OK(t, err)
	test.Equals(t, &auth.UpdatePasswordResponse{}, resp)
}

func TestUpdatePasswordNotFound(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()
	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	token := "123abc"
	code := "123456"
	newPassword := "newPassword"

	dl.Expect(mock.NewExpectation(dl.VerificationCode, token).WithReturns((*dal.VerificationCode)(nil), api.ErrNotFound("foo")))

	resp, err := s.UpdatePassword(context.Background(), &auth.UpdatePasswordRequest{
		Token:       token,
		Code:        code,
		NewPassword: newPassword,
	})
	test.AssertNil(t, resp)
	test.Equals(t, codes.NotFound, grpc.Code(err))
}

// Test the password updating functionality
func TestUpdatePasswordCodeExpired(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()
	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	aID1, err := dal.NewAccountID()
	test.OK(t, err)
	token := "123abc"
	code := "123456"
	newPassword := "newPassword"
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.VerificationCode, token).WithReturns(&dal.VerificationCode{
		Code:          code,
		Expires:       time.Unix(0, 0),
		VerifiedValue: aID1.String(),
	}, nil))

	resp, err := s.UpdatePassword(context.Background(), &auth.UpdatePasswordRequest{
		Token:       token,
		Code:        code,
		NewPassword: newPassword,
	})
	test.AssertNil(t, resp)
	test.Equals(t, auth.VerificationCodeExpired, grpc.Code(err))
}

// Test the password updating functionality
func TestUpdatePasswordBadCode(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	defer dl.Finish()
	settingsMock := mock_settings.New(t)
	defer settingsMock.Finish()
	s, err := New(dl, settingsMock, clientEncryptionSecret)
	test.OK(t, err)
	aID1, err := dal.NewAccountID()
	test.OK(t, err)
	token := "123abc"
	code := "123456"
	newPassword := "newPassword"
	test.OK(t, err)

	dl.Expect(mock.NewExpectation(dl.VerificationCode, token).WithReturns(&dal.VerificationCode{
		Code:          code + "1",
		Expires:       time.Unix(time.Now().Unix()+10000, 0),
		VerifiedValue: aID1.String(),
	}, nil))

	resp, err := s.UpdatePassword(context.Background(), &auth.UpdatePasswordRequest{
		Token:       token,
		Code:        code,
		NewPassword: newPassword,
	})
	test.AssertNil(t, resp)
	test.Equals(t, auth.BadVerificationCode, grpc.Code(err))
}
