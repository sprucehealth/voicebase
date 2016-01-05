package server

import (
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"golang.org/x/net/context"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/cmd/svc/auth/internal/dal"
	mock_dal "github.com/sprucehealth/backend/cmd/svc/auth/internal/dal/test"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/hash"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
	"github.com/sprucehealth/backend/svc/auth"
	"github.com/sprucehealth/backend/test"
)

func TestGetAccount(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	s := New(dl)
	aID1 := dal.NewAccountID(1)
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
	mock.FinishAll(dl)
}

func TestGetAccountNotFound(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	s := New(dl)
	aID1 := dal.NewAccountID(1)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.Account, aID1), (*dal.Account)(nil), api.ErrNotFound("not found")))
	_, err := s.GetAccount(context.Background(), &auth.GetAccountRequest{AccountID: aID1.String()})
	test.Assert(t, err != nil, "Expected an error")
	test.Equals(t, codes.NotFound, grpc.Code(err))
	mock.FinishAll(dl)
}

func TestAuthenticateLogin(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	s := New(dl)
	hasher := hash.NewBcryptHasher(bCryptHashCost)
	email := "test@email.com"
	password := "password"
	hashedPassword, err := hasher.GenerateFromPassword([]byte(password))
	test.OK(t, err)
	aID1 := dal.NewAccountID(1)
	var token string
	var expiration uint64
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.AccountForEmail, email), &dal.Account{ID: aID1, Password: hashedPassword}, nil))
	dl.Expect(mock.WithReturns(mock.NewExpectationFn(dl.InsertAuthToken, func(p ...interface{}) {
		test.Equals(t, 1, len(p))
		at, ok := p[0].(*dal.AuthToken)
		test.Assert(t, ok, "Expected *dal.AuthToken")
		test.Assert(t, strings.HasSuffix(string(at.Token), ":testattribute"), "Expected auth token to have attribute suffix, got: %s", at.Token)
		test.Assert(t, at.Expires.Unix() >= time.Now().Unix(), "Expected expiration token to be in the future but was %v", at.Expires)
		test.Assert(t, at.AccountID.String() == aID1.String(), "Expected auth token to map to account id %s, but got %s", aID1.String(), at.AccountID.String())
		token = strings.Split(string(at.Token), ":")[0]
		expiration = uint64(at.Expires.Unix())
	}), nil))
	resp, err := s.AuthenticateLogin(context.Background(), &auth.AuthenticateLoginRequest{
		Email:           email,
		Password:        password,
		TokenAttributes: map[string]string{"test": "attribute"},
	})
	test.OK(t, err)

	test.AssertNotNil(t, resp.Token)
	test.AssertNotNil(t, resp.Account)
	test.Equals(t, token, resp.Token.Value)
	test.Equals(t, expiration, resp.Token.ExpirationEpoch)
	mock.FinishAll(dl)
}

func TestAuthenticateLoginNoEmail(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	s := New(dl)
	email := "test@email.com"
	password := "password"
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.AccountForEmail, email), (*dal.Account)(nil), api.ErrNotFound("not found")))
	_, err := s.AuthenticateLogin(context.Background(), &auth.AuthenticateLoginRequest{
		Email:           email,
		Password:        password,
		TokenAttributes: map[string]string{"test": "attribute"},
	})
	test.Assert(t, err != nil, "Expected an error")

	test.Equals(t, auth.EmailNotFound, grpc.Code(err))
	mock.FinishAll(dl)
}

func TestAuthenticateBadPassword(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	s := New(dl)
	email := "test@email.com"
	password := "password"
	aID1 := dal.NewAccountID(1)
	dl.Expect(mock.WithReturns(mock.NewExpectation(dl.AccountForEmail, email), &dal.Account{ID: aID1, Password: []byte("notpassword")}, nil))
	_, err := s.AuthenticateLogin(context.Background(), &auth.AuthenticateLoginRequest{
		Email:           email,
		Password:        password,
		TokenAttributes: map[string]string{"test": "attribute"},
	})
	test.Assert(t, err != nil, "Expected an error")

	test.Equals(t, auth.BadPassword, grpc.Code(err))
	mock.FinishAll(dl)
}

func TestCheckAuthentication(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	mClock := clock.NewManaged(time.Now())
	s := New(dl)
	svr, ok := s.(*server)
	test.Assert(t, ok, "Expected a *server")
	svr.clk = mClock
	s = svr
	token := "123abc"
	tokenAttributes := map[string]string{"token": "attribute"}
	aID1 := dal.NewAccountID(1)
	expires := mClock.Now().Add(defaultTokenExpiration)
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

	test.Assert(t, resp.IsAuthenticated, "Expected authentication")
	test.AssertNotNil(t, resp.Account)
	test.AssertNotNil(t, resp.Token)
	test.Equals(t, &auth.Account{
		ID:        aID1.String(),
		FirstName: "bat",
		LastName:  "man",
	}, resp.Account)
	test.Equals(t, &auth.AuthToken{
		Value:           token,
		ExpirationEpoch: uint64(expires.Unix()),
	}, resp.Token)
	mock.FinishAll(dl)
}

func TestCheckAuthenticationRefresh(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	mClock := clock.NewManaged(time.Now())
	s := New(dl)
	svr, ok := s.(*server)
	test.Assert(t, ok, "Expected a *server")
	svr.clk = mClock
	s = svr
	token := "123abc"
	tokenAttributes := map[string]string{"token": "attribute"}
	aID1 := dal.NewAccountID(1)
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

	test.Assert(t, resp.IsAuthenticated, "Expected authentication")
	test.AssertNotNil(t, resp.Account)
	test.AssertNotNil(t, resp.Token)
	test.Equals(t, &auth.Account{
		ID:        aID1.String(),
		FirstName: "bat",
		LastName:  "man",
	}, resp.Account)
	test.Equals(t, &auth.AuthToken{
		Value:           token,
		ExpirationEpoch: uint64(refreshedExpiration.Unix()),
	}, resp.Token)
	mock.FinishAll(dl)
}

func TestCheckAuthenticationNoToken(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	mClock := clock.NewManaged(time.Now())
	s := New(dl)
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
	mock.FinishAll(dl)
}

func TestCreateAccount(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	s := New(dl)
	fn := "bat"
	ln := "man"
	email := "bat@man.com"
	phoneNumber := "+12345678910"
	password := "password"
	hasher := hash.NewBcryptHasher(bCryptHashCost)
	aID1 := dal.NewAccountID(1)
	aEID1 := dal.NewAccountEmailID(2)
	aPID1 := dal.NewAccountPhoneID(3)
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
		PrimaryAccountPhoneID: &aPID1,
		PrimaryAccountEmailID: &aEID1,
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

	test.AssertNotNil(t, resp.Token)
	test.AssertNotNil(t, resp.Account)
	test.Equals(t, &auth.Account{
		ID:        aID1.String(),
		FirstName: "bat",
		LastName:  "man",
	}, resp.Account)
	test.Equals(t, &auth.AuthToken{
		Value:           token,
		ExpirationEpoch: expiration,
	}, resp.Token)
	mock.FinishAll(dl)
}

func TestCreateAccountMissingData(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	s := New(dl)
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
	mock.FinishAll(dl)
}

func TestCreateAccountBadEmail(t *testing.T) {
	dl := mock_dal.NewMockDAL(t)
	s := New(dl)
	fn := "bat"
	ln := "man"
	email := "notarealemail"
	phoneNumber := "+12345678910"
	password := "password"
	_, err := s.CreateAccount(context.Background(), &auth.CreateAccountRequest{
		FirstName:   fn,
		LastName:    ln,
		PhoneNumber: phoneNumber,
		Email:       email,
		Password:    password,
	})
	test.Assert(t, err != nil, "Expected an error")

	test.Equals(t, auth.InvalidEmail, grpc.Code(err))
	mock.FinishAll(dl)
}
