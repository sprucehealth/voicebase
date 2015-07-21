package promotions

import (
	"fmt"
	"time"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/conc"
	"github.com/sprucehealth/backend/libs/golog"
)

// CreateReferralProgramForDoctor creates a route-to-doctor referral program for the doctor,
// if one doesn't already exist. The code that is generated for this program is unique to each doctor
// and deterministic in that its either the doctor's last name if its not already used as one of the promo codes
// or the lastName with a single digit code appended to the end
func CreateReferralProgramForDoctor(doctor *common.Doctor, dataAPI api.DataAPI, apiDomain string) error {

	// check if the referral program for the doctor exists
	_, err := dataAPI.ActiveReferralProgramForAccount(doctor.AccountID.Int64(), common.PromotionTypes)
	if err == nil {
		return nil
	} else if !api.IsErrNotFound(err) {
		return err
	}

	// if not, create one of the doctor

	displayMsg := fmt.Sprintf("Complete a Spruce Visit with %s", doctor.LongDisplayName)
	shortMsg := fmt.Sprintf("Visit with %s", doctor.ShortDisplayName)
	successMsg := fmt.Sprintf("You will be seen by %s.", doctor.LongDisplayName)

	promotion, err := NewRouteDoctorPromotion(
		doctor.ID.Int64(),
		doctor.LongDisplayName,
		doctor.ShortDisplayName,
		app_url.ThumbnailURL(apiDomain, api.RoleDoctor, doctor.ID.Int64()),
		"new_user",
		displayMsg, shortMsg, successMsg, 0, USDUnit)
	if err != nil {
		return err
	}

	rp := NewDoctorReferralProgram(
		doctor.AccountID.Int64(),
		displayMsg,
		fmt.Sprintf("Share this code to see patients on Spruce"),
		"new_user",
		promotion.(*routeDoctorPromotion),
	)

	referralCode, err := generateReferralCodeForDoctor(dataAPI, doctor)
	if err != nil {
		return err
	}

	if err := dataAPI.CreateReferralProgram(&common.ReferralProgram{
		AccountID: doctor.AccountID.Int64(),
		Code:      referralCode,
		Data:      rp,
		Status:    common.RSActive,
	}); err != nil {
		return err
	}
	return nil
}

// PromotionDisplayInfo represents the information the client should use to display a given promotion
type PromotionDisplayInfo struct {
	Title    string
	ImageURL string
}

// LookupPromoCode returns the display information for the provided code if the code exists
// as a promotion, and is not expired.
func LookupPromoCode(code string, dataAPI api.DataAPI, analyticsLogger analytics.Logger) (*PromotionDisplayInfo, error) {
	promoCode, err := dataAPI.LookupPromoCode(code)
	if api.IsErrNotFound(err) {
		return nil, ErrInvalidCode
	} else if err != nil {
		return nil, err
	}

	var promotion *common.Promotion
	if promoCode.IsReferral {
		rp, err := dataAPI.ReferralProgram(promoCode.ID, common.PromotionTypes)
		if err != nil {
			return nil, err
		}
		promotion = rp.Data.(ReferralProgram).PromotionForReferredAccount(promoCode.Code)
	} else {
		promotion, err = dataAPI.Promotion(promoCode.ID, common.PromotionTypes)
		if err != nil {
			return nil, err
		}
	}

	// ensure that the promotion has not expired
	if promotion.Expires != nil && promotion.Expires.Before(time.Now()) {
		return nil, ErrPromotionExpired
	}

	p := promotion.Data.(Promotion)

	go func() {
		analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Event:     "promo_code_lookup",
				Timestamp: analytics.Time(time.Now()),
				ExtraJSON: analytics.JSONString(struct {
					Code       string `json:"code"`
					IsReferral bool   `json:"is_referral"`
				}{
					Code:       code,
					IsReferral: promoCode.IsReferral,
				}),
			},
		})
	}()

	return &PromotionDisplayInfo{
		Title:    p.DisplayMessage(),
		ImageURL: p.ImageURL(),
	}, nil
}

// AssociatePromoCode associates the provided promo code (if it exists) with the email address entered. If the email
// does not identify an existing account, then a ParkedAccount is created with the promotion associated to it so that when
// the user signs up with the same email address, the promotion will be applied to their account. If the email does identify an
// existing account, then the promotion is applied to the account. Note that for security purposes, all work of
// creating a ParkedAccount or associating the code with an existing account can be done asynchronously if desired.
func AssociatePromoCode(email, state, code string, dataAPI api.DataAPI, authAPI api.AuthAPI, analyticsLogger analytics.Logger, async bool) (string, error) {
	var err error

	// lookup promotion
	promoCode, err := dataAPI.LookupPromoCode(code)
	if api.IsErrNotFound(err) {
		return "", ErrInvalidCode
	} else if err != nil {
		return "", err
	}

	var promotion *common.Promotion
	var referralProgram ReferralProgram
	if promoCode.IsReferral {
		rp, err := dataAPI.ReferralProgram(promoCode.ID, common.PromotionTypes)
		if err != nil {
			return "", err
		}
		referralProgram = rp.Data.(ReferralProgram)
		promotion = referralProgram.PromotionForReferredAccount(promoCode.Code)
	} else {
		promotion, err = dataAPI.Promotion(promoCode.ID, common.PromotionTypes)
		if err != nil {
			return "", err
		}
	}

	// Bind this coljure to the outer error message so we can view that error when executing synchronously
	associationAction := func() {
		// check if account exists
		var account *common.Account
		account, err = authAPI.AccountForEmail(email)
		if err != api.ErrLoginDoesNotExist && err != nil {
			golog.Errorf(err.Error())
			return
		}

		// account exists
		var accountID int64
		var parkedAccount *common.ParkedAccount
		if err == nil {
			// ensure that we are dealing with a patient account
			if account.Role != api.RolePatient {
				golog.Errorf("Attempt made to associate promotion with non-patient role for account id %d", account.ID)
				return
			}

			// associate the promotion with the patient account
			if err = promotion.Data.(Promotion).Associate(account.ID, promoCode.ID, promotion.Expires, dataAPI); err != nil {
				golog.Errorf(err.Error())
				return
			}

			if referralProgram != nil {
				if err = referralProgram.ReferredAccountAssociatedCode(account.ID, promoCode.ID, dataAPI); err != nil {
					golog.Errorf(err.Error())
					return
				}
			}
		} else {
			// given that a user account does not exist, park the email and the promotion
			// to be associated with the patient when the account is actually created
			parkedAccount = &common.ParkedAccount{
				Email:  email,
				State:  state,
				CodeID: promoCode.ID,
			}
			if _, err = dataAPI.CreateParkedAccount(parkedAccount); err != nil {
				golog.Errorf(err.Error())
				return
			}
		}

		extraJSON := struct {
			Code            string `json:"code"`
			State           string `json:"state"`
			IsReferral      bool   `json:"is_referral"`
			IsNewUser       bool   `json:"is_new_user"`
			ParkedAccountID int64  `json:"parked_account_id,omitempty"`
		}{
			Code:       code,
			State:      state,
			IsReferral: promoCode.IsReferral,
			IsNewUser:  account == nil,
		}
		if parkedAccount != nil {
			extraJSON.ParkedAccountID = parkedAccount.ID
		}
		analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Event:     "promo_code_associate",
				Timestamp: analytics.Time(time.Now()),
				AccountID: accountID,
				ExtraJSON: analytics.JSONString(extraJSON),
			},
		})
	}

	// If executing async then explicitly return nil error to avoid a race condition
	if async {
		conc.Go(associationAction)
		return promotion.Data.(Promotion).SuccessMessage(), nil
	}

	associationAction()
	return promotion.Data.(Promotion).SuccessMessage(), err
}

// PatientSignedup attempts to identify a ParkedAccount with the same email as the patient that just signed up,
// and then applies the pending promotion to the patient's account if one exists.
func PatientSignedup(accountID int64, email string, dataAPI api.DataAPI, analyticsLogger analytics.Logger) (string, error) {
	// check if a parked account exists
	parkedAccount, err := dataAPI.ParkedAccount(email)
	if api.IsErrNotFound(err) {
		return "", nil
	} else if err != nil {
		return "", err
	}

	// if it does, asynchronously assocate the promo code with this patient account
	// while returning the success message of the promotion
	var promotion *common.Promotion
	var referralProgram ReferralProgram
	if parkedAccount.IsReferral {
		rp, err := dataAPI.ReferralProgram(parkedAccount.CodeID, common.PromotionTypes)
		if err != nil {
			return "", err
		}
		referralProgram = rp.Data.(ReferralProgram)
		promotion = referralProgram.PromotionForReferredAccount(parkedAccount.Code)
	} else {
		promotion, err = dataAPI.Promotion(parkedAccount.CodeID, common.PromotionTypes)
		if err != nil {
			return "", err
		}
	}

	conc.Go(func() {
		if err := dataAPI.MarkParkedAccountAsAccountCreated(parkedAccount.ID); err != nil {
			golog.Errorf(err.Error())
			return
		}

		// associate the promotion with the patient account
		if err := promotion.Data.(Promotion).Associate(accountID, parkedAccount.CodeID, promotion.Expires, dataAPI); err != nil {
			golog.Errorf(err.Error())
			return
		}

		if referralProgram != nil {
			if err := referralProgram.ReferredAccountAssociatedCode(accountID, parkedAccount.CodeID, dataAPI); err != nil {
				golog.Errorf(err.Error())
				return
			}
		}

		analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Event:     "promo_code_signup",
				Timestamp: analytics.Time(time.Now()),
				AccountID: accountID,
				ExtraJSON: analytics.JSONString(struct {
					ParkedAccountID int64  `json:"parked_account_id"`
					Code            string `json:"code"`
					IsReferral      bool   `json:"is_referral"`
				}{
					ParkedAccountID: parkedAccount.ID,
					Code:            promotion.Code,
					IsReferral:      parkedAccount.IsReferral,
				}),
			},
		})
	})

	return promotion.Data.(Promotion).SuccessMessage(), nil
}
