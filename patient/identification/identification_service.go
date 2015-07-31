package identification

import (
	"strings"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/ptr"
)

const (
	// AccountNeedsIDVerificationEmailSuffix is the esuffix to attach to account emails when the account has been flagged for ID verification
	accountNeedsIDVerificationEmailSuffix = "+IDVerificationNeeded"
)

// Service describes the methods required to provide patient identification related services to the back end
type Service interface {
	// MarkForNeedsIDVerification moves the patient's account into an unusable state by modifying the associated email, creating a parked account, and associating a promo code.
	MarkForNeedsIDVerification(patientID int64, promoCode string) error
}

// PatientIdentificationService is a service layer struct that encapsulates logic related to identification related operations.
//  As a service layer struct this should not perform an direct data layer access and should rely on the provided DAL interface
type patientIdentificationService struct {
	dataAPI   api.DataAPI
	authAPI   api.AuthAPI
	analytics analytics.Logger
}

// NewPatientIdentificationService returns an initialized instance of patientIdentificationService
func NewPatientIdentificationService(dataAPI api.DataAPI, authAPI api.AuthAPI, analytics analytics.Logger) Service {
	return &patientIdentificationService{
		dataAPI:   dataAPI,
		authAPI:   authAPI,
		analytics: analytics,
	}
}

// MarkForNeedsIDVerification moves the patient's account into an unusable state by modifying the associated email, creating a parked account, and associating a promo code.
func (s *patientIdentificationService) MarkForNeedsIDVerification(patientID int64, promoCode string) error {
	patient, err := s.dataAPI.Patient(patientID, true)
	if err != nil {
		return errors.Trace(err)
	}

	account, err := s.authAPI.GetAccount(patient.AccountID.Int64())
	if err != nil {
		return errors.Trace(err)
	}

	// Look up the associated promo code and the patient's location
	promotionCode, err := s.dataAPI.LookupPromoCode(promoCode)
	if err != nil {
		return errors.Trace(err)
	}
	_, state, err := s.dataAPI.PatientLocation(patientID)
	if err != nil {
		return errors.Trace(err)
	}

	// Idempotency: If the associated account already appears to be deactivated then interact with the unmunged email
	accountEmail := account.Email
	if strings.HasSuffix(accountEmail, accountNeedsIDVerificationEmailSuffix) {
		golog.Infof("MarkForNeedsIDVerification initiated on account where email already contained suffix %s, proceeding with stripped suffix.", accountNeedsIDVerificationEmailSuffix)
		accountEmail = strings.TrimSuffix(accountEmail, accountNeedsIDVerificationEmailSuffix)
	}

	// Idempotency: ParkedAccount's documentation describes it as performing an INSERT IGNORE so no need for idempotency check here
	id, err := s.dataAPI.CreateParkedAccount(&common.ParkedAccount{
		Email:  accountEmail,
		State:  state,
		CodeID: promotionCode.ID,
	})
	if err != nil {
		return errors.Trace(err)
	}
	if id != 0 {
		golog.Infof("Created parked account %d with promo code id %d", id, promotionCode.ID)
	}

	// Idempotency: Update the email associated with the account if it hasn't already been updated
	if !strings.HasSuffix(account.Email, accountNeedsIDVerificationEmailSuffix) {
		if err := s.authAPI.UpdateAccount(account.ID, ptr.String(account.Email+accountNeedsIDVerificationEmailSuffix), nil); err != nil {
			return errors.Trace(err)
		}
		golog.Infof("Account %d email updated with %s as part of deactivation", account.ID, accountNeedsIDVerificationEmailSuffix)
	}

	if _, err := s.dataAPI.DeactivateScheduledMessagesForPatient(patientID); err != nil {
		return errors.Trace(err)
	}

	if err := s.dataAPI.DeletePushCommunicationPreferenceForAccount(account.ID); err != nil {
		return errors.Trace(err)
	}

	s.analytics.WriteEvents([]analytics.Event{&analytics.ServerEvent{Event: "mark_for_needs_id_verification", PatientID: patient.ID.Int64(), AccountID: account.ID}})
	return nil
}
