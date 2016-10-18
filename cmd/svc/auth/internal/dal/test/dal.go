package test

import (
	"context"
	"testing"
	"time"

	"github.com/sprucehealth/backend/cmd/svc/auth/internal/dal"
	"github.com/sprucehealth/backend/device"
	"github.com/sprucehealth/backend/libs/errors"
	"github.com/sprucehealth/backend/libs/testhelpers/mock"
)

var _ dal.DAL = &MockDAL{}

type MockDAL struct {
	*mock.Expector
}

// NewMockDAL returns an initialized instance of MockDAL
func NewMockDAL(t *testing.T) *MockDAL {
	return &MockDAL{&mock.Expector{T: t}}
}

func (dl *MockDAL) InsertAccount(ctx context.Context, model *dal.Account) (dal.AccountID, error) {
	rets := dl.Record(model)
	if len(rets) == 0 {
		return dal.AccountID{}, nil
	}
	return rets[0].(dal.AccountID), mock.SafeError(rets[1])
}

func (dl *MockDAL) Account(ctx context.Context, id dal.AccountID) (*dal.Account, error) {
	rets := dl.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.Account), mock.SafeError(rets[1])
}

func (dl *MockDAL) AccountForEmail(ctx context.Context, email string) (*dal.Account, error) {
	rets := dl.Record(email)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.Account), mock.SafeError(rets[1])
}

func (dl *MockDAL) UpdateAccount(ctx context.Context, id dal.AccountID, update *dal.AccountUpdate) (int64, error) {
	rets := dl.Record(id, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) DeleteAccount(ctx context.Context, id dal.AccountID) (int64, error) {
	rets := dl.Record(id)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) InsertAuthToken(ctx context.Context, model *dal.AuthToken) error {
	rets := dl.Record(model)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *MockDAL) ActiveAuthTokenForAccount(ctx context.Context, accountID dal.AccountID, deviceID string, duration dal.AuthTokenDurationType) (*dal.AuthToken, error) {
	rets := dl.Record(accountID, deviceID, duration)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.AuthToken), mock.SafeError(rets[1])
}

func (dl *MockDAL) AuthToken(ctx context.Context, token string, expiresAfter time.Time, forUpdate bool) (*dal.AuthToken, error) {
	rets := dl.Record(token, expiresAfter, forUpdate)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.AuthToken), mock.SafeError(rets[1])
}

func (dl *MockDAL) DeleteAuthTokens(ctx context.Context, accountID dal.AccountID) (int64, error) {
	rets := dl.Record(accountID)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) DeleteAuthTokensWithSuffix(ctx context.Context, accountID dal.AccountID, suffix string) (int64, error) {
	rets := dl.Record(accountID, suffix)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) DeleteExpiredAuthTokens(ctx context.Context, expiredBefore time.Time) (int64, error) {
	rets := dl.Record(expiredBefore)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) DeleteAuthToken(ctx context.Context, token string) (int64, error) {
	rets := dl.Record(token)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) UpdateAuthToken(ctx context.Context, token string, update *dal.AuthTokenUpdate) (int64, error) {
	rets := dl.Record(token, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) InsertAccountEvent(ctx context.Context, model *dal.AccountEvent) (dal.AccountEventID, error) {
	rets := dl.Record(model)
	if len(rets) == 0 {
		return dal.AccountEventID{}, nil
	}
	return rets[0].(dal.AccountEventID), mock.SafeError(rets[1])
}

func (dl *MockDAL) AccountEvent(ctx context.Context, id dal.AccountEventID) (*dal.AccountEvent, error) {
	rets := dl.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.AccountEvent), mock.SafeError(rets[1])
}

func (dl *MockDAL) DeleteAccountEvent(ctx context.Context, id dal.AccountEventID) (int64, error) {
	rets := dl.Record(id)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) InsertAccountPhone(ctx context.Context, model *dal.AccountPhone) (dal.AccountPhoneID, error) {
	rets := dl.Record(model)
	if len(rets) == 0 {
		return dal.AccountPhoneID{}, nil
	}
	return rets[0].(dal.AccountPhoneID), mock.SafeError(rets[1])
}

func (dl *MockDAL) AccountPhone(ctx context.Context, id dal.AccountPhoneID) (*dal.AccountPhone, error) {
	rets := dl.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.AccountPhone), mock.SafeError(rets[1])
}

func (dl *MockDAL) UpdateAccountPhone(ctx context.Context, id dal.AccountPhoneID, update *dal.AccountPhoneUpdate) (int64, error) {
	rets := dl.Record(id, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) DeleteAccountPhone(ctx context.Context, id dal.AccountPhoneID) (int64, error) {
	rets := dl.Record(id)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) InsertAccountEmail(ctx context.Context, model *dal.AccountEmail) (dal.AccountEmailID, error) {
	rets := dl.Record(model)
	if len(rets) == 0 {
		return dal.AccountEmailID{}, nil
	}
	return rets[0].(dal.AccountEmailID), mock.SafeError(rets[1])
}

func (dl *MockDAL) AccountEmail(ctx context.Context, id dal.AccountEmailID) (*dal.AccountEmail, error) {
	rets := dl.Record(id)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.AccountEmail), mock.SafeError(rets[1])
}

func (dl *MockDAL) UpdateAccountEmail(ctx context.Context, id dal.AccountEmailID, update *dal.AccountEmailUpdate) (int64, error) {
	rets := dl.Record(id, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) DeleteAccountEmail(ctx context.Context, id dal.AccountEmailID) (int64, error) {
	rets := dl.Record(id)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) InsertVerificationCode(ctx context.Context, model *dal.VerificationCode) error {
	rets := dl.Record(model)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *MockDAL) UpdateVerificationCode(ctx context.Context, token string, update *dal.VerificationCodeUpdate) (int64, error) {
	rets := dl.Record(token, update)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) VerificationCode(ctx context.Context, token string) (*dal.VerificationCode, error) {
	rets := dl.Record(token)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.VerificationCode), mock.SafeError(rets[1])
}

func (dl *MockDAL) VerificationCodesByValue(ctx context.Context, codeType dal.VerificationCodeType, value string) ([]*dal.VerificationCode, error) {
	rets := dl.Record(codeType, value)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].([]*dal.VerificationCode), mock.SafeError(rets[1])
}

func (dl *MockDAL) DeleteVerificationCode(ctx context.Context, token string) (int64, error) {
	rets := dl.Record(token)
	if len(rets) == 0 {
		return 0, nil
	}
	return rets[0].(int64), mock.SafeError(rets[1])
}

func (dl *MockDAL) TwoFactorLogin(ctx context.Context, accountID dal.AccountID, deviceID string) (*dal.TwoFactorLogin, error) {
	rets := dl.Record(accountID, deviceID)
	if len(rets) == 0 {
		return nil, nil
	}
	return rets[0].(*dal.TwoFactorLogin), mock.SafeError(rets[1])
}

func (dl *MockDAL) UpsertTwoFactorLogin(ctx context.Context, accountID dal.AccountID, deviceID string, loginTime time.Time) error {
	rets := dl.Record(accountID, deviceID, loginTime)
	if len(rets) == 0 {
		return nil
	}
	return mock.SafeError(rets[0])
}

func (dl *MockDAL) TrackLogin(ctx context.Context, accountID dal.AccountID, platform device.Platform, deviceID string) error {
	rets := dl.Record(accountID, platform, deviceID)
	if len(rets) == 0 {
		return nil
	}

	return mock.SafeError(rets[0])
}

func (dl *MockDAL) LastLogin(ctx context.Context, accountID dal.AccountID) (*dal.LoginInfo, error) {
	rets := dl.Record(accountID)
	if len(rets) == 0 {
		return nil, nil
	}

	return rets[0].(*dal.LoginInfo), mock.SafeError(rets[1])
}

func (dl *MockDAL) Transact(ctx context.Context, trans func(ctx context.Context, dal dal.DAL) error) (err error) {
	if err := trans(ctx, dl); err != nil {
		return errors.Trace(err)
	}
	return nil
}
