package test

import (
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/auth/internal/dal"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

type mockDAL struct {
	*mock.Expector
}

// NewDAL returns an initialized instance of mockDAL. This returns the interface for a build time check that this mock always matches.
func NewDAL() dal.DAL {
	return &mockDAL{}
}

// NewMockDAL returns an initialized instance of mockDAL
func NewMockDAL(t *testing.T) *mockDAL {
	return &mockDAL{&mock.Expector{T: t}}
}

func (dl *mockDAL) InsertAccount(model *dal.Account) (dal.AccountID, error) {
	rets := dl.Record(model)
	if len(rets) == 0 {
		return dal.AccountID{}, nil
	}
	return rets[0].(dal.AccountID), mock.SafeError(rets[1])
}

func (dl *mockDAL) Account(id dal.AccountID) (*dal.Account, error) {
	rets := dl.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.Account), mock.SafeError(rets[1])
}

func (dl *mockDAL) AccountForEmail(email string) (*dal.Account, error) {
	rets := dl.Record(email)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.Account), mock.SafeError(rets[1])
}

func (dl *mockDAL) UpdateAccount(id dal.AccountID, update *dal.AccountUpdate) (int64, error) {
	rets := dl.Record(id, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *mockDAL) DeleteAccount(id dal.AccountID) (int64, error) {
	rets := dl.Record(id)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *mockDAL) InsertAuthToken(model *dal.AuthToken) error {
	rets := dl.Record(model)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *mockDAL) ActiveAuthTokenForAccount(accountID dal.AccountID) (*dal.AuthToken, error) {
	rets := dl.Record(accountID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.AuthToken), mock.SafeError(rets[1])
}

func (dl *mockDAL) AuthToken(token string, expiresAfter time.Time, forUpdate bool) (*dal.AuthToken, error) {
	rets := dl.Record(token, expiresAfter, forUpdate)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.AuthToken), mock.SafeError(rets[1])
}

func (dl *mockDAL) DeleteAuthTokens(accountID dal.AccountID) (int64, error) {
	rets := dl.Record(accountID)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *mockDAL) DeleteAuthTokensWithSuffix(accountID dal.AccountID, suffix string) (int64, error) {
	rets := dl.Record(accountID, suffix)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *mockDAL) DeleteAuthToken(token string) (int64, error) {
	rets := dl.Record(token)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *mockDAL) UpdateAuthToken(token string, update *dal.AuthTokenUpdate) (int64, error) {
	rets := dl.Record(token, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *mockDAL) InsertAccountEvent(model *dal.AccountEvent) (dal.AccountEventID, error) {
	rets := dl.Record(model)
	if len(rets) == 0 {
		return dal.AccountEventID{}, nil
	}
	return rets[0].(dal.AccountEventID), mock.SafeError(rets[1])
}

func (dl *mockDAL) AccountEvent(id dal.AccountEventID) (*dal.AccountEvent, error) {
	rets := dl.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.AccountEvent), mock.SafeError(rets[1])
}

func (dl *mockDAL) DeleteAccountEvent(id dal.AccountEventID) (int64, error) {
	rets := dl.Record(id)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *mockDAL) InsertAccountPhone(model *dal.AccountPhone) (dal.AccountPhoneID, error) {
	rets := dl.Record(model)
	if len(rets) == 0 {
		return dal.AccountPhoneID{}, nil
	}
	return rets[0].(dal.AccountPhoneID), mock.SafeError(rets[1])
}

func (dl *mockDAL) AccountPhone(id dal.AccountPhoneID) (*dal.AccountPhone, error) {
	rets := dl.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.AccountPhone), mock.SafeError(rets[1])
}

func (dl *mockDAL) UpdateAccountPhone(id dal.AccountPhoneID, update *dal.AccountPhoneUpdate) (int64, error) {
	rets := dl.Record(id, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *mockDAL) DeleteAccountPhone(id dal.AccountPhoneID) (int64, error) {
	rets := dl.Record(id)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *mockDAL) InsertAccountEmail(model *dal.AccountEmail) (dal.AccountEmailID, error) {
	rets := dl.Record(model)
	if len(rets) == 0 {
		return dal.AccountEmailID{}, nil
	}
	return rets[0].(dal.AccountEmailID), mock.SafeError(rets[1])
}

func (dl *mockDAL) AccountEmail(id dal.AccountEmailID) (*dal.AccountEmail, error) {
	rets := dl.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.AccountEmail), mock.SafeError(rets[1])
}

func (dl *mockDAL) UpdateAccountEmail(id dal.AccountEmailID, update *dal.AccountEmailUpdate) (int64, error) {
	rets := dl.Record(id, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *mockDAL) DeleteAccountEmail(id dal.AccountEmailID) (int64, error) {
	rets := dl.Record(id)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *mockDAL) InsertVerificationCode(model *dal.VerificationCode) error {
	rets := dl.Record(model)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *mockDAL) UpdateVerificationCode(token string, update *dal.VerificationCodeUpdate) (int64, error) {
	rets := dl.Record(token, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *mockDAL) VerificationCode(token string) (*dal.VerificationCode, error) {
	rets := dl.Record(token)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.VerificationCode), mock.SafeError(rets[1])
}

func (dl *mockDAL) DeleteVerificationCode(token string) (int64, error) {
	rets := dl.Record(token)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *mockDAL) TwoFactorLogin(accountID dal.AccountID, deviceID string) (*dal.TwoFactorLogin, error) {
	rets := dl.Record(accountID, deviceID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.TwoFactorLogin), mock.SafeError(rets[1])
}

func (dl *mockDAL) UpsertTwoFactorLogin(accountID dal.AccountID, deviceID string, loginTime time.Time) error {
	rets := dl.Record(accountID, deviceID, loginTime)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *mockDAL) Transact(trans func(dal dal.DAL) error) (err error) {
	if err := trans(dl); err != nil {
		return errors.Trace(err)
	}
	return nil
}
