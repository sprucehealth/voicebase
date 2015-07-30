package identification

import (
	"testing"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/encoding"
	"github.com/sprucehealth/backend/test"
)

type mockIdentificationServiceDataAPI struct {
	api.DataAPI
	patientParam                                     int64
	patientErr                                       error
	patient                                          *common.Patient
	lookupPromoCodeParam                             string
	lookupPromoCodeErr                               error
	lookupPromoCode                                  *common.PromoCode
	patientLocationParam                             int64
	patientLocationErr                               error
	patientLocationZip                               string
	patientLocationState                             string
	createParkedAccountParam                         *common.ParkedAccount
	createParkedAccountErr                           error
	createParkedAccount                              int64
	deactivateScheduledMessagesForPatientParam       int64
	deactivateScheduledMessagesForPatientErr         error
	deactivateScheduledMessagesForPatient            int64
	deletePushCommunicationPreferenceForAccountParam int64
	deletePushCommunicationPreferenceForAccountErr   error
}

func (m *mockIdentificationServiceDataAPI) Patient(id int64, basicInfoOnly bool) (*common.Patient, error) {
	m.patientParam = id
	return m.patient, m.patientErr
}

func (m *mockIdentificationServiceDataAPI) LookupPromoCode(code string) (*common.PromoCode, error) {
	m.lookupPromoCodeParam = code
	return m.lookupPromoCode, m.lookupPromoCodeErr
}

func (m *mockIdentificationServiceDataAPI) PatientLocation(patientID int64) (zipcode string, state string, err error) {
	m.patientLocationParam = patientID
	return m.patientLocationZip, m.patientLocationState, m.patientLocationErr
}

func (m *mockIdentificationServiceDataAPI) CreateParkedAccount(parkedAccount *common.ParkedAccount) (int64, error) {
	m.createParkedAccountParam = parkedAccount
	return m.createParkedAccount, m.createParkedAccountErr
}

func (m *mockIdentificationServiceDataAPI) DeactivateScheduledMessagesForPatient(patientID int64) (int64, error) {
	m.deactivateScheduledMessagesForPatientParam = patientID
	return m.deactivateScheduledMessagesForPatient, m.deactivateScheduledMessagesForPatientErr
}

func (m *mockIdentificationServiceDataAPI) DeletePushCommunicationPreferenceForAccount(accountID int64) error {
	m.deletePushCommunicationPreferenceForAccountParam = accountID
	return m.deletePushCommunicationPreferenceForAccountErr
}

type mockIdentificationServiceAuthAPI struct {
	api.AuthAPI
	getAccountParam             int64
	getAccountErr               error
	getAccount                  *common.Account
	updateAccountAccountIDParam int64
	updateAccountEmailParam     *string
	updateAccountTwoFactorParam *bool
	updateAccountErr            error
}

func (m *mockIdentificationServiceAuthAPI) GetAccount(id int64) (*common.Account, error) {
	m.getAccountParam = id
	return m.getAccount, m.getAccountErr
}

func (m *mockIdentificationServiceAuthAPI) UpdateAccount(accountID int64, email *string, twoFactorEnabled *bool) error {
	m.updateAccountAccountIDParam = accountID
	m.updateAccountEmailParam = email
	m.updateAccountTwoFactorParam = twoFactorEnabled
	return m.updateAccountErr
}

// Begin MarkForNeedsIDVerification tests
func TestMarkForNeedsIDVerificationHappyCase(t *testing.T) {
	var patientID int64 = 1
	var accountID int64 = 2
	var promoCodeID int64 = 3
	var parkedAccountID int64 = 5
	var nilTwoFactor *bool
	state := "CA"
	promoCode := "Foo"
	email := "not@verified.com"
	dataAPI := &mockIdentificationServiceDataAPI{
		patient:              &common.Patient{AccountID: encoding.NewObjectID(accountID)},
		lookupPromoCode:      &common.PromoCode{ID: promoCodeID},
		patientLocationState: state,
		createParkedAccount:  parkedAccountID,
	}
	authAPI := &mockIdentificationServiceAuthAPI{
		getAccount: &common.Account{ID: accountID, Email: email},
	}
	analytics := analytics.NullLogger{}
	service := NewPatientIdentificationService(dataAPI, authAPI, analytics)
	test.OK(t, service.MarkForNeedsIDVerification(patientID, promoCode))
	test.Equals(t, patientID, dataAPI.patientParam)
	test.Equals(t, accountID, authAPI.getAccountParam)
	test.Equals(t, promoCode, dataAPI.lookupPromoCodeParam)
	test.Equals(t, patientID, dataAPI.patientLocationParam)
	test.Equals(t, &common.ParkedAccount{
		Email:  email,
		State:  state,
		CodeID: promoCodeID,
	}, dataAPI.createParkedAccountParam)
	test.Equals(t, accountID, authAPI.updateAccountAccountIDParam)
	test.Equals(t, email+accountNeedsIDVerificationEmailSuffix, *authAPI.updateAccountEmailParam)
	test.Equals(t, nilTwoFactor, authAPI.updateAccountTwoFactorParam)
	test.Equals(t, patientID, dataAPI.deactivateScheduledMessagesForPatientParam)
	test.Equals(t, accountID, dataAPI.deletePushCommunicationPreferenceForAccountParam)
}

func TestMarkForNeedsIDVerificationIdempotent(t *testing.T) {
	var patientID int64 = 1
	var accountID int64 = 2
	var promoCodeID int64 = 3
	var parkedAccountID int64 = 5
	var nilTwoFactor *bool
	var nilEmail *string
	state := "CA"
	promoCode := "Foo"
	email := "not@verified.com"
	dataAPI := &mockIdentificationServiceDataAPI{
		patient:              &common.Patient{AccountID: encoding.NewObjectID(accountID)},
		lookupPromoCode:      &common.PromoCode{ID: promoCodeID},
		patientLocationState: state,
		createParkedAccount:  parkedAccountID,
	}
	authAPI := &mockIdentificationServiceAuthAPI{
		getAccount: &common.Account{ID: accountID, Email: email + accountNeedsIDVerificationEmailSuffix},
	}
	analytics := analytics.NullLogger{}
	service := NewPatientIdentificationService(dataAPI, authAPI, analytics)
	test.OK(t, service.MarkForNeedsIDVerification(patientID, promoCode))
	test.Equals(t, patientID, dataAPI.patientParam)
	test.Equals(t, accountID, authAPI.getAccountParam)
	test.Equals(t, promoCode, dataAPI.lookupPromoCodeParam)
	test.Equals(t, patientID, dataAPI.patientLocationParam)
	test.Equals(t, &common.ParkedAccount{
		Email:  email,
		State:  state,
		CodeID: promoCodeID,
	}, dataAPI.createParkedAccountParam)
	test.Equals(t, int64(0), authAPI.updateAccountAccountIDParam)
	test.Equals(t, nilEmail, authAPI.updateAccountEmailParam)
	test.Equals(t, nilTwoFactor, authAPI.updateAccountTwoFactorParam)
	test.Equals(t, patientID, dataAPI.deactivateScheduledMessagesForPatientParam)
	test.Equals(t, accountID, dataAPI.deletePushCommunicationPreferenceForAccountParam)
}

// End MarkForNeedsIDVerification tests
