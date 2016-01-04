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
	"github.com/sprucehealth/backend/libs/validate"
	"github.com/sprucehealth/backend/svc/auth"
	"golang.org/x/net/context"
)

// Server represents the methods required to interact with the auth service
type Server interface {
	AuthenticateLogin(context.Context, *auth.AuthenticateLoginRequest) (*auth.AuthenticateLoginResponse, error)
	CheckAuthentication(context.Context, *auth.CheckAuthenticationRequest) (*auth.CheckAuthenticationResponse, error)
	CreateAccount(context.Context, *auth.CreateAccountRequest) (*auth.CreateAccountResponse, error)
	GetAccount(context.Context, *auth.GetAccountRequest) (*auth.GetAccountResponse, error)
	Unauthenticate(context.Context, *auth.UnauthenticateRequest) (*auth.UnauthenticateResponse, error)
}

// authDAL represents the methods expected to be present on the data access layer mechanisms provided
type authDAL interface {
	InsertAccount(model *dal.Account) (dal.AccountID, error)
	Account(id dal.AccountID) (*dal.Account, error)
	AccountForEmail(email string) (*dal.Account, error)
	UpdateAccount(id dal.AccountID, update *dal.AccountUpdate) (int64, error)
	DeleteAccount(id dal.AccountID) (int64, error)
	InsertAuthToken(model *dal.AuthToken) error
	AuthToken(token string, expiresAfter time.Time) (*dal.AuthToken, error)
	DeleteAuthTokens(accountID dal.AccountID) (int64, error)
	DeleteAuthTokensWithSuffix(accountID dal.AccountID, suffix string) (int64, error)
	DeleteAuthToken(token string) (int64, error)
	UpdateAuthToken(token string, update *dal.AuthTokenUpdate) (int64, error)
	InsertAccountEvent(model *dal.AccountEvent) (dal.AccountEventID, error)
	AccountEvent(id dal.AccountEventID) (*dal.AccountEvent, error)
	DeleteAccountEvent(id dal.AccountEventID) (int64, error)
	InsertAccountPhone(model *dal.AccountPhone) (dal.AccountPhoneID, error)
	AccountPhone(id dal.AccountPhoneID) (*dal.AccountPhone, error)
	UpdateAccountPhone(id dal.AccountPhoneID, update *dal.AccountPhoneUpdate) (int64, error)
	DeleteAccountPhone(id dal.AccountPhoneID) (int64, error)
	InsertAccountEmail(model *dal.AccountEmail) (dal.AccountEmailID, error)
	AccountEmail(id dal.AccountEmailID) (*dal.AccountEmail, error)
	UpdateAccountEmail(id dal.AccountEmailID, update *dal.AccountEmailUpdate) (int64, error)
	DeleteAccountEmail(id dal.AccountEmailID) (int64, error)
	Transact(trans func(dl dal.DAL) error) (err error)
}

var (
	// ErrNotImplemented is returned from RPC calls that have yet to be implemented
	ErrNotImplemented = errors.New("Not Implemented")
)

type server struct {
	dal    authDAL
	hasher hash.PasswordHasher
	clk    clock.Clock
}

var bCryptHashCost = 10

// New returns an initialized instance of server
func New(dal authDAL) Server {
	return &server{
		dal:    dal,
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
		return &auth.AuthenticateLoginResponse{
			Success: false,
			Failure: &auth.AuthenticateLoginResponse_Failure{
				Reason:  auth.AuthenticateLoginResponse_Failure_UNKNOWN_EMAIL,
				Message: fmt.Sprintf("Unknown email: %s", rd.Email),
			},
		}, nil
	} else if err != nil {
		return nil, errors.Trace(err)
	}
	golog.Debugf("Got account %+v", account)

	golog.Debugf("Comparing password with hash...")
	if err := s.hasher.CompareHashAndPassword(account.Password, []byte(rd.Password)); err != nil {
		golog.Debugf("Error while comparing password: %s", err)
		return &auth.AuthenticateLoginResponse{
			Success: false,
			Failure: &auth.AuthenticateLoginResponse_Failure{
				Reason:  auth.AuthenticateLoginResponse_Failure_PASSWORD_MISMATCH,
				Message: fmt.Sprintf("The password does not match the provided account email %s", rd.Email),
			},
		}, nil
	}

	authToken, err := generateAndInsertToken(s.dal, account.ID, rd.TokenAttributes, s.clk)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &auth.AuthenticateLoginResponse{
		Success: true,
		Token: &auth.AuthToken{
			Value:           authToken.token,
			ExpirationEpoch: uint64(authToken.expiration.Unix()),
		},
		Account: &auth.Account{
			ID:        account.ID.String(),
			FirstName: account.FirstName,
			LastName:  account.LastName,
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
		return nil, errors.Trace(err)
	}

	golog.Debugf("Checking authentication of token %s", attributedToken)
	aToken, err := s.dal.AuthToken(attributedToken, s.clk.Now())
	if api.IsErrNotFound(err) {
		return &auth.CheckAuthenticationResponse{
			Success:         true,
			IsAuthenticated: false,
		}, nil
	} else if err != nil {
		return nil, errors.Trace(err)
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
			authToken.Value = rotatedToken.token
			authToken.ExpirationEpoch = uint64(rotatedToken.expiration.Unix())
			_, err = dl.DeleteAuthToken(attributedToken)
			return errors.Trace(err)
		}); err != nil {
			return nil, errors.Trace(err)
		}
	}

	golog.Debugf("Getting account %s", aToken.AccountID)
	account, err := s.dal.Account(aToken.AccountID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &auth.CheckAuthenticationResponse{
		Success:         true,
		IsAuthenticated: true,
		Token:           authToken,
		Account: &auth.Account{
			ID:        account.ID.String(),
			FirstName: account.FirstName,
			LastName:  account.LastName,
		},
	}, nil
}

func (s *server) CreateAccount(ctx context.Context, rd *auth.CreateAccountRequest) (*auth.CreateAccountResponse, error) {
	golog.Debugf("Entering server.server.CreateAccount: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.CreateAccount...") }()
	}
	if errResp := s.validateCreateAccountRequest(rd); errResp != nil {
		return errResp, nil
	}

	var account *dal.Account
	var authToken *authToken
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
			PhoneNumber: rd.PhoneNumber,
		})
		if err != nil {
			return errors.Trace(err)
		}
		golog.Debugf("Account phone inserted %s", accountPhoneID)

		golog.Debugf("Updating account for email and phone")
		aff, err := dl.UpdateAccount(accountID, &dal.AccountUpdate{
			PrimaryAccountPhoneID: &accountPhoneID,
			PrimaryAccountEmailID: &accountEmailID,
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
		return nil, errors.Trace(err)
	}

	return &auth.CreateAccountResponse{
		Success: true,
		Token: &auth.AuthToken{
			Value:           authToken.token,
			ExpirationEpoch: uint64(authToken.expiration.Unix()),
		},
		Account: &auth.Account{
			ID:        account.ID.String(),
			FirstName: account.FirstName,
			LastName:  account.LastName,
		},
	}, nil
}

func (s *server) validateCreateAccountRequest(rd *auth.CreateAccountRequest) *auth.CreateAccountResponse {
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
		return &auth.CreateAccountResponse{
			Success: false,
			Failure: &auth.CreateAccountResponse_Failure{
				Reason:  auth.CreateAccountResponse_Failure_INVALID_INPUT,
				Message: invalidInputMessage,
			},
		}
	}

	if !validate.Email(rd.Email) {
		return &auth.CreateAccountResponse{
			Success: false,
			Failure: &auth.CreateAccountResponse_Failure{
				Reason:  auth.CreateAccountResponse_Failure_EMAIL_NOT_VALID,
				Message: fmt.Sprintf("The provided email is not valid: %s", rd.Email),
			},
		}
	}

	/*
		if _, err := common.ParsePhone(rd.PhoneNumber); err != nil {
			return &auth.CreateAccountResponse{
				Success: false,
				Failure: &auth.CreateAccountResponse_Failure{
					Reason:  auth.CreateAccountResponse_Failure_PHONE_NUMBER_NOT_VALID,
					Message: fmt.Sprintf("The provided phone number is not valid: %s", rd.PhoneNumber),
				},
			}
		}
	*/
	return nil
}

func (s *server) GetAccount(ctx context.Context, rd *auth.GetAccountRequest) (*auth.GetAccountResponse, error) {
	golog.Debugf("Entering server.server.GetAccount: %+v", rd)
	if golog.Default().L(golog.DEBUG) {
		defer func() { golog.Debugf("Leaving server.server.GetAccount...") }()
	}
	account, err := s.dal.Account(dal.ParseAccountID(rd.AccountID))
	if api.IsErrNotFound(err) {
		return &auth.GetAccountResponse{
			Success: false,
			Failure: &auth.GetAccountResponse_Failure{
				Reason:  auth.GetAccountResponse_Failure_NOT_FOUND,
				Message: fmt.Sprintf("Account with ID %s not found", rd.AccountID),
			},
		}, nil
	}
	return &auth.GetAccountResponse{
		Success: true,
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
		return nil, errors.Trace(err)
	}
	golog.Debugf("Deleting auth token %s", tokenWithAttributes)
	if aff, err := s.dal.DeleteAuthToken(tokenWithAttributes); err != nil {
		return nil, errors.Trace(err)
	} else if aff != 1 {
		return nil, errors.Trace(fmt.Errorf("Expected 1 row to be affected but got %d", aff))
	}
	return &auth.UnauthenticateResponse{
		Success: true,
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

func generateAndInsertToken(dl dal.DAL, accountID dal.AccountID, tokenAttributes map[string]string, clk clock.Clock) (*authToken, error) {
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
	golog.Debugf("Auth token inserted")

	golog.Debugf("Deleting old attributed tokens for this account")
	attributes, err := appendAttributes("", tokenAttributes)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if _, err := dl.DeleteAuthTokensWithSuffix(accountID, attributes); err != nil {
		return nil, errors.Trace(err)
	}
	return &authToken{token: token, expiration: tokenExpiration}, nil
}
