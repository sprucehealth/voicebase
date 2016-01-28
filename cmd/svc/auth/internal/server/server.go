package server

import (
	"fmt"
	"sort"
	"time"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/cmd/svc/auth/internal/dal"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/clock"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/hash"
	"github.com/sprucehealth/backend/libs/phone"
	"github.com/sprucehealth/backend/libs/ptr"
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/auth"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var grpcErrorf = grpc.Errorf

var (
	// ErrNotImplemented is returned from RPC calls that have yet to be implemented
	ErrNotImplemented = errors.New("Not Implemented")
)

type server struct {
	dal    dal.DAL
	hasher hash.PasswordHasher
	clk    clock.Clock
}

var bCryptHashCost = 10

// New returns an initialized instance of server
func New(dl dal.DAL) auth.AuthServer {
	return &server{
		dal:    dl,
		hasher: hash.NewBcryptHasher(bCryptHashCost),
		clk:    clock.New(),
	}
}

func (s *server) AuthenticateLogin(ctx context.Context, rd *auth.AuthenticateLoginRequest) (*auth.AuthenticateLoginResponse, error) {
	golog.Debugf("Entering server.server.AuthenticateLogin: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.AuthenticateLogin...") }()
	}
	golog.Debugf("Getting account for email %s...", rd.Email)
	account, err := s.dal.AccountForEmail(rd.Email)
	if api.IsErrNotFound(err) {
		return nil, grpcErrorf(auth.EmailNotFound, "Unknown email: %s", rd.Email)
	} else if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	golog.Debugf("Got account %+v", account)

	golog.Debugf("Comparing password with hash...")
	if err := s.hasher.CompareHashAndPassword(account.Password, []byte(rd.Password)); err != nil {
		golog.Debugf("Error while comparing password: %s", err)
		return nil, grpcErrorf(auth.BadPassword, "The password does not match the provided account email: %s", rd.Email)
	}

	var authToken *auth.AuthToken
	var twoFactorPhone string
	accountRequiresTwoFactor := true
	if accountRequiresTwoFactor {
		// TODO: Make this response and data less phone/sms specific
		accountPhone, err := s.dal.AccountPhone(account.PrimaryAccountPhoneID)
		if err != nil {
			return nil, grpcErrorf(codes.Internal, err.Error())
		}
		twoFactorPhone = accountPhone.PhoneNumber
	} else {
		authToken, err = generateAndInsertToken(s.dal, account.ID, rd.TokenAttributes, s.clk)
		if err != nil {
			return nil, grpcErrorf(codes.Internal, err.Error())
		}
	}

	return &auth.AuthenticateLoginResponse{
		Token: authToken,
		Account: &auth.Account{
			ID:        account.ID.String(),
			FirstName: account.FirstName,
			LastName:  account.LastName,
		},
		TwoFactorRequired:    accountRequiresTwoFactor,
		TwoFactorPhoneNumber: twoFactorPhone,
	}, nil
}

func (s *server) AuthenticateLoginWithCode(ctx context.Context, rd *auth.AuthenticateLoginWithCodeRequest) (*auth.AuthenticateLoginWithCodeResponse, error) {
	golog.Debugf("Entering server.server.AuthenticateLoginForCode: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.AuthenticateLoginForCode...") }()
	}

	if rd.Token == "" {
		return nil, grpcErrorf(codes.NotFound, "No verification code maps to the provided token %q", rd.Token)
	}

	verificationCode, err := s.dal.VerificationCode(rd.Token)
	if api.IsErrNotFound(err) {
		return nil, grpcErrorf(codes.NotFound, "No verification code maps to the provided token %q", rd.Token)
	} else if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	} else if verificationCode.VerificationType != dal.VerificationCodeTypeAccount2fa {
		return nil, grpcErrorf(codes.NotFound, "No 2FA verification code maps to the provided token %q", rd.Token)
	}

	// Check that the code matches the token and it is not expired
	if verificationCode.Code != rd.Code {
		return nil, grpcErrorf(auth.BadVerificationCode, "The code mapped to the provided token does not match %s", rd.Code)
	} else if verificationCode.Expires.Unix() < s.clk.Now().Unix() {
		return nil, grpcErrorf(auth.VerificationCodeExpired, "The code mapped to the provided token has expired.")
	}

	// Since we sucessfully checked the token, mark it as consumed
	_, err = s.dal.UpdateVerificationCode(rd.Token, &dal.VerificationCodeUpdate{
		Consumed: ptr.Bool(true),
	})
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	accountID, err := dal.ParseAccountID(verificationCode.VerifiedValue)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, "ACCOUNT_2FA verification code value %q failed to parse into account id, unable to generate auth token: %s", verificationCode.VerifiedValue, err)
	}

	authToken, err := generateAndInsertToken(s.dal, accountID, rd.TokenAttributes, s.clk)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, "Failed to generate and insert new auth token for ACCOUNT_2FA: %s", err)
	}

	acc, err := s.dal.Account(accountID)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, "Failed to fetch account record for ACCOUNT_2FA: %s", err)
	}

	return &auth.AuthenticateLoginWithCodeResponse{
		Token: authToken,
		Account: &auth.Account{
			ID:        acc.ID.String(),
			FirstName: acc.FirstName,
			LastName:  acc.LastName,
		},
	}, nil
}

func (s *server) CheckAuthentication(ctx context.Context, rd *auth.CheckAuthenticationRequest) (*auth.CheckAuthenticationResponse, error) {
	golog.Debugf("Entering server.server.CheckAuthentication: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.CheckAuthentication...") }()
	}
	attributedToken, err := appendAttributes(rd.Token, rd.TokenAttributes)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	golog.Debugf("Checking authentication of token %s", attributedToken)
	aToken, err := s.dal.AuthToken(attributedToken, s.clk.Now())
	if api.IsErrNotFound(err) {
		return &auth.CheckAuthenticationResponse{
			IsAuthenticated: false,
		}, nil
	} else if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	authToken := &auth.AuthToken{
		Value:           rd.Token,
		ExpirationEpoch: uint64(aToken.Expires.Unix()),
	}
	if rd.Refresh {
		if err := s.dal.Transact(func(dl dal.DAL) error {
			rotatedToken, err := generateAndInsertToken(dl, aToken.AccountID, rd.TokenAttributes, s.clk)
			if err != nil {
				return errors.Trace(err)
			}
			authToken = rotatedToken
			_, err = dl.DeleteAuthToken(attributedToken)
			return errors.Trace(err)
		}); err != nil {
			return nil, grpcErrorf(codes.Internal, err.Error())
		}
	}

	golog.Debugf("Getting account %s", aToken.AccountID)
	account, err := s.dal.Account(aToken.AccountID)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	return &auth.CheckAuthenticationResponse{
		IsAuthenticated: true,
		Token:           authToken,
		Account: &auth.Account{
			ID:        account.ID.String(),
			FirstName: account.FirstName,
			LastName:  account.LastName,
		},
	}, nil
}

func (s *server) CheckVerificationCode(ctx context.Context, rd *auth.CheckVerificationCodeRequest) (*auth.CheckVerificationCodeResponse, error) {
	golog.Debugf("Entering server.server.CheckVerificationCode: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.CheckVerificationCode...") }()
	}
	if rd.Token == "" {
		return nil, grpcErrorf(codes.NotFound, "No verification code maps to the provided token %q", rd.Token)
	}

	verificationCode, err := s.dal.VerificationCode(rd.Token)
	if api.IsErrNotFound(err) {
		return nil, grpcErrorf(codes.NotFound, "No verification code maps to the provided token %q", rd.Token)
	} else if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	// Check that the code matches the token and it is not expired
	if verificationCode.Code != rd.Code {
		return nil, grpcErrorf(auth.BadVerificationCode, "The code mapped to the provided token does not match %s", rd.Code)
	} else if verificationCode.Expires.Unix() < s.clk.Now().Unix() {
		return nil, grpcErrorf(auth.VerificationCodeExpired, "The code mapped to the provided token has expired.")
	}

	// Since we sucessfully checked the token, mark it as consumed
	_, err = s.dal.UpdateVerificationCode(rd.Token, &dal.VerificationCodeUpdate{
		Consumed: ptr.Bool(true),
	})
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	// If this is a ACCOUNT_2FA token return the account object as well
	var account *auth.Account
	if verificationCode.VerificationType == dal.VerificationCodeTypeAccount2fa {
		accountID, err := dal.ParseAccountID(verificationCode.VerifiedValue)
		if err != nil {
			return nil, grpcErrorf(codes.Internal, "ACCOUNT_2FA verification code value %q failed to parse into account id: %s", verificationCode.VerifiedValue, err)
		}

		acc, err := s.dal.Account(accountID)
		if err != nil {
			return nil, grpcErrorf(codes.Internal, "Failed to fetch account record for ACCOUNT_2FA: %s", err)
		}
		account = &auth.Account{
			ID:        acc.ID.String(),
			FirstName: acc.FirstName,
			LastName:  acc.LastName,
		}
	}

	return &auth.CheckVerificationCodeResponse{
		Account: account,
		Value:   verificationCode.VerifiedValue,
	}, nil
}

func (s *server) CreateAccount(ctx context.Context, rd *auth.CreateAccountRequest) (*auth.CreateAccountResponse, error) {
	golog.Debugf("Entering server.server.CreateAccount: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.CreateAccount...") }()
	}
	if errResp := s.validateCreateAccountRequest(rd); errResp != nil {
		return nil, errResp
	}
	pn, err := phone.ParseNumber(rd.PhoneNumber)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "The provided phone number is not valid: %s", rd.PhoneNumber)
	}

	var account *dal.Account
	var authToken *auth.AuthToken
	if err := s.dal.Transact(func(dl dal.DAL) error {
		hashedPassword, err := s.hasher.GenerateFromPassword([]byte(rd.Password))
		if err != nil {
			return errors.Trace(err)
		}

		golog.Debugf("Inserting account")
		accountID, err := dl.InsertAccount(&dal.Account{
			FirstName: rd.FirstName,
			LastName:  rd.LastName,
			Password:  hashedPassword,
			Status:    dal.AccountStatusActive,
		})
		if err != nil {
			return errors.Trace(err)
		}
		golog.Debugf("Account inserted %s", accountID)

		golog.Debugf("Inserting account email")
		accountEmailID, err := dl.InsertAccountEmail(&dal.AccountEmail{
			AccountID: accountID,
			Email:     rd.Email,
			Status:    dal.AccountEmailStatusActive,
			Verified:  false,
		})
		if err != nil {
			return errors.Trace(err)
		}
		golog.Debugf("Account email inserted %s", accountEmailID)

		golog.Debugf("Inserting account phone")
		accountPhoneID, err := dl.InsertAccountPhone(&dal.AccountPhone{
			AccountID:   accountID,
			Verified:    false,
			Status:      dal.AccountPhoneStatusActive,
			PhoneNumber: pn.String(),
		})
		if err != nil {
			return errors.Trace(err)
		}
		golog.Debugf("Account phone inserted %s", accountPhoneID)

		golog.Debugf("Updating account for email and phone")
		aff, err := dl.UpdateAccount(accountID, &dal.AccountUpdate{
			PrimaryAccountPhoneID: accountPhoneID,
			PrimaryAccountEmailID: accountEmailID,
		})
		golog.Debugf("Account updated: %d affected", aff)
		if err != nil {
			return errors.Trace(err)
		} else if aff != 1 {
			return errors.Trace(fmt.Errorf("Expected 1 row to be affected but got %d", aff))
		}

		authToken, err = generateAndInsertToken(dl, accountID, rd.TokenAttributes, s.clk)
		if err != nil {
			return errors.Trace(err)
		}

		golog.Debugf("Getting account %s", accountID)
		account, err = dl.Account(accountID)
		golog.Debugf("Account %+v", account)
		return errors.Trace(err)
	}); err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}

	return &auth.CreateAccountResponse{
		Token: authToken,
		Account: &auth.Account{
			ID:        account.ID.String(),
			FirstName: account.FirstName,
			LastName:  account.LastName,
		},
	}, nil
}

func (s *server) validateCreateAccountRequest(rd *auth.CreateAccountRequest) error {
	var invalidInputMessage string
	if rd.FirstName == "" {
		invalidInputMessage = "FirstName cannot be empty"
	} else if rd.LastName == "" {
		invalidInputMessage = "LastName cannot be empty"
	} else if rd.Email == "" {
		invalidInputMessage = "Email cannot be empty"
	} else if rd.PhoneNumber == "" {
		invalidInputMessage = "PhoneNumber cannot be empty"
	} else if rd.Password == "" {
		invalidInputMessage = "Password cannot be empty"
	}
	if invalidInputMessage != "" {
		return grpcErrorf(codes.InvalidArgument, invalidInputMessage)
	}

	if !validate.Email(rd.Email) {
		return grpcErrorf(auth.InvalidEmail, "The provided email is not valid: %s", rd.Email)
	}
	return nil
}

func (s *server) CreateVerificationCode(ctx context.Context, rd *auth.CreateVerificationCodeRequest) (*auth.CreateVerificationCodeResponse, error) {
	golog.Debugf("Entering server.server.CreateVerificationCode: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.CreateVerificationCode...") }()
	}
	verificationCode, err := generateAndInsertVerificationCode(s.dal, rd.ValueToVerify, rd.Type, s.clk)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	return &auth.CreateVerificationCodeResponse{
		VerificationCode: verificationCode,
	}, nil
}

func (s *server) GetAccount(ctx context.Context, rd *auth.GetAccountRequest) (*auth.GetAccountResponse, error) {
	golog.Debugf("Entering server.server.GetAccount: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.GetAccount...") }()
	}
	id, err := dal.ParseAccountID(rd.AccountID)
	if err != nil {
		return nil, grpcErrorf(codes.InvalidArgument, "Unable to parse provided account ID")
	}
	account, err := s.dal.Account(id)
	if api.IsErrNotFound(err) {
		return nil, grpcErrorf(codes.NotFound, "Account with ID %s not found", rd.AccountID)
	}
	return &auth.GetAccountResponse{
		Account: &auth.Account{
			ID:        account.ID.String(),
			FirstName: account.FirstName,
			LastName:  account.LastName,
		},
	}, nil
}

func (s *server) Unauthenticate(ctx context.Context, rd *auth.UnauthenticateRequest) (*auth.UnauthenticateResponse, error) {
	golog.Debugf("Entering server.server.Unauthenticate: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.Unauthenticate...") }()
	}
	tokenWithAttributes, err := appendAttributes(rd.Token, rd.TokenAttributes)
	if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	}
	golog.Debugf("Deleting auth token %s", tokenWithAttributes)
	if aff, err := s.dal.DeleteAuthToken(tokenWithAttributes); err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	} else if aff != 1 {
		return nil, grpcErrorf(codes.Internal, "Expected 1 row to be affected but got %d", aff)
	}
	return &auth.UnauthenticateResponse{}, nil
}

func (s *server) VerifiedValue(ctx context.Context, rd *auth.VerifiedValueRequest) (*auth.VerifiedValueResponse, error) {
	golog.Debugf("Entering server.server.VerifiedValue: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.VerifiedValue...") }()
	}

	if rd.Token == "" {
		return nil, grpcErrorf(codes.NotFound, "No verification code maps to the provided token %q", rd.Token)
	}

	verificationCode, err := s.dal.VerificationCode(rd.Token)
	if api.IsErrNotFound(err) {
		return nil, grpcErrorf(codes.NotFound, "No verification code maps to the provided token %q", rd.Token)
	} else if err != nil {
		return nil, grpcErrorf(codes.Internal, err.Error())
	} else if !verificationCode.Consumed {
		return nil, grpcErrorf(auth.ValueNotYetVerified, "The value mapped to this token %q has not yet been verified", rd.Token)
	}

	return &auth.VerifiedValueResponse{
		Value: verificationCode.VerifiedValue,
	}, nil
}

const (
	maxTokenSize           = 250
	defaultTokenExpiration = 60 * 60 * 24 * time.Second
)

func appendAttributes(token string, tokenAttributes map[string]string) (string, error) {
	golog.Debugf("Entering server.appendAttributes: Token: %s, TokenAttributes: %+v", token, tokenAttributes)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.appendAttributes...") }()
	}
	if len(tokenAttributes) > 0 {
		token += ":"
		// due to the non deterministic nature of maps we need to sort the keys and always apply in that order
		// note: to get around this for optimization purposes we could require the caller to provide a list instead
		var i int
		keys := make([]string, len(tokenAttributes))
		for k := range tokenAttributes {
			keys[i] = k
			i++
		}
		sort.Strings(keys)
		for _, k := range keys {
			token += (k + tokenAttributes[k])
		}
		if len(token) >= maxTokenSize {
			return "", errors.Trace(fmt.Errorf("Provided client data makes token too long..."))
		}
	}
	return token, nil
}

type authToken struct {
	token      string
	expiration time.Time
}

func generateAndInsertToken(dl dal.DAL, accountID dal.AccountID, tokenAttributes map[string]string, clk clock.Clock) (*auth.AuthToken, error) {
	golog.Debugf("Entering server.generateAndInsertToken: AccountID: %s, TokenAttributes: %+v", accountID, tokenAttributes)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.generateAndInsertToken...") }()
	}
	token, err := common.GenerateToken()
	if err != nil {
		return nil, errors.Trace(err)
	}
	tokenWithAttributes, err := appendAttributes(token, tokenAttributes)
	if err != nil {
		return nil, errors.Trace(err)
	}
	tokenExpiration := clk.Now().Add(defaultTokenExpiration)

	golog.Debugf("Inserting auth token %s - expires %+v for account %s", tokenWithAttributes, tokenExpiration, accountID)
	if err := dl.InsertAuthToken(&dal.AuthToken{
		AccountID: accountID,
		Expires:   tokenExpiration,
		Token:     []byte(tokenWithAttributes),
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &auth.AuthToken{Value: token, ExpirationEpoch: uint64(tokenExpiration.Unix())}, nil
}

const (
	verificationCodeDigitCount        = 6
	verificationCodeMaxValue          = 999999
	defaultVerificationCodeExpiration = 15 * time.Minute
)

func generateAndInsertVerificationCode(dl dal.DAL, valueToVerify string, codeType auth.VerificationCodeType, clk clock.Clock) (*auth.VerificationCode, error) {
	golog.Debugf("Entering server.generateAndInsertVerificationCode: valueToVerify: %s", valueToVerify)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.generateAndInsertVerificationCode...") }()
	}
	token, err := common.GenerateToken()
	if err != nil {
		return nil, errors.Trace(err)
	}
	code, err := common.GenerateRandomNumber(verificationCodeMaxValue, verificationCodeDigitCount)
	if err != nil {
		return nil, errors.Trace(err)
	}
	tokenExpiration := clk.Now().Add(defaultVerificationCodeExpiration)

	vType, err := dal.ParseVerificationCodeType(auth.VerificationCodeType_name[int32(codeType)])
	if err != nil {
		return nil, errors.Trace(err)
	}

	// TODO: Remove logging of the code perhaps?
	golog.Debugf("Inserting verification code %s - with token %s - for value %s - expires %+v.", token, valueToVerify, tokenExpiration)
	if err := dl.InsertVerificationCode(&dal.VerificationCode{
		Token:            token,
		Code:             code,
		Expires:          tokenExpiration,
		VerificationType: vType,
		VerifiedValue:    valueToVerify,
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &auth.VerificationCode{
		Token:           token,
		Type:            codeType,
		Code:            code,
		ExpirationEpoch: uint64(tokenExpiration.Unix()),
	}, nil
}
