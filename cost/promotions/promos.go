package promotions

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/golog"
)

// CreateReferralProgramForDoctor creates a route-to-doctor referral program for the doctor,
// if one doesn't already exist. The code that is generated for this program is unique to each doctor
// and deterministic in that its either the doctor's last name if its not already used as one of the promo codes
// or the lastName with a single digit code appended to the end
func CreateReferralProgramForDoctor(doctor *common.Doctor, dataAPI api.DataAPI) error {

	// check if the referral program for the doctor exists
	_, err := dataAPI.ActiveReferralProgramForAccount(doctor.AccountId.Int64(), Types)
	if err != nil && err != api.NoRowsError {
		return nil
	} else if err == nil {
		return nil
	}

	// if not, create one of the doctor

	displayMsg := fmt.Sprintf("Complete a Spruce Visit with %s", doctor.LongDisplayName)
	shortMsg := fmt.Sprintf("Visit with %s", doctor.ShortDisplayName)
	successMsg := fmt.Sprintf("You will be seen by %s", doctor.LongDisplayName)

	promotion, err := NewRouteDoctorPromotion(
		doctor.DoctorId.Int64(),
		doctor.LongDisplayName,
		doctor.ShortDisplayName,
		doctor.SmallThumbnailURL, "new_user",
		displayMsg, shortMsg, successMsg, 0, USDUnit)
	if err != nil {
		return err
	}

	rp := NewDoctorReferralProgram(
		doctor.AccountId.Int64(),
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
		AccountID: doctor.AccountId.Int64(),
		Code:      referralCode,
		Data:      rp,
		Status:    common.RSActive,
	}); err != nil {
		return err
	}
	return nil
}

type PromotionDisplayInfo struct {
	Title    string
	ImageURL string
}

// LookupPromoCode returns the display information for the provided code if the code exists
// as a promotion, and is not expired.
func LookupPromoCode(code string, dataAPI api.DataAPI, analyticsLogger analytics.Logger) (*PromotionDisplayInfo, error) {
	promoCode, err := dataAPI.LookupPromoCode(code)
	if err == api.NoRowsError {
		return nil, InvalidCode
	} else if err != nil {
		return nil, err
	}

	var promotion *common.Promotion
	if promoCode.IsReferral {
		rp, err := dataAPI.ReferralProgram(promoCode.ID, Types)
		if err != nil {
			return nil, err
		}
		promotion = rp.Data.(ReferralProgram).PromotionForReferredPatient(promoCode.Code)
	} else {
		promotion, err = dataAPI.Promotion(promoCode.ID, Types)
		if err != nil {
			return nil, err
		}
	}

	// ensure that the promotion has not expired
	if promotion.Expires != nil && promotion.Expires.Before(time.Now()) {
		return nil, PromotionExpired
	}

	p := promotion.Data.(Promotion)

	go func() {

		jsonData, err := json.Marshal(map[string]interface{}{
			"code":        code,
			"is_referral": promoCode.IsReferral,
		})
		if err != nil {
			golog.Errorf(err.Error())
		}

		analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Event:     "promo_code_lookup",
				Timestamp: analytics.Time(time.Now()),
				ExtraJSON: string(jsonData),
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
// creating a ParkedAccount or associating the code with an existing account is done asynchronously.
func AssociatePromoCode(email, state, code string, dataAPI api.DataAPI, authAPI api.AuthAPI, analyticsLogger analytics.Logger, done chan bool) (string, error) {
	// lookup promotion
	promoCode, err := dataAPI.LookupPromoCode(code)
	if err == api.NoRowsError {
		return "", InvalidCode
	} else if err != nil {
		return "", err
	}

	var promotion *common.Promotion
	var referralProgram ReferralProgram
	if promoCode.IsReferral {
		rp, err := dataAPI.ReferralProgram(promoCode.ID, Types)
		if err != nil {
			return "", err
		}
		referralProgram = rp.Data.(ReferralProgram)
		promotion = referralProgram.PromotionForReferredPatient(promoCode.Code)
	} else {
		promotion, err = dataAPI.Promotion(promoCode.ID, Types)
		if err != nil {
			return "", err
		}
	}
	// do the work of creating a parked account or associating the promotion
	// with an existing account in the background so that we give no
	// indication of whether or not an account exists
	go func() {
		defer func() {
			if done != nil {
				done <- true
			}
		}()

		// check if account exists
		account, err := authAPI.GetAccountForEmail(email)
		if err != api.LoginDoesNotExist && err != nil {
			golog.Errorf(err.Error())
			return
		}

		// account exists
		var patientID int64
		var parkedAccount *common.ParkedAccount
		if err == nil {
			// ensure that we are dealing with a patient account
			if account.Role != api.PATIENT_ROLE {
				golog.Errorf("Attempt made to associate promotion with non-patient role for account id %d", account.ID)
				return
			}

			patientID, err = dataAPI.GetPatientIdFromAccountId(account.ID)
			if err != nil {
				golog.Errorf(err.Error())
				return
			}

			// associate the promotion with the patient account
			if err := promotion.Data.(Promotion).Associate(patientID, promoCode.ID, promotion.Expires, dataAPI); err != nil {
				golog.Errorf(err.Error())
				return
			}

			if referralProgram != nil {
				if err := referralProgram.ReferredPatientAssociatedCode(patientID, promoCode.ID, dataAPI); err != nil {
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
			if _, err := dataAPI.CreateParkedAccount(parkedAccount); err != nil {
				golog.Errorf(err.Error())
				return
			}
		}

		extraJSON := map[string]interface{}{
			"code":        code,
			"state":       state,
			"is_referral": promoCode.IsReferral,
			"is_new_user": (account == nil),
		}
		if parkedAccount != nil {
			extraJSON["parked_account_id"] = parkedAccount.ID
		}
		jsonData, err := json.Marshal(extraJSON)
		if err != nil {
			golog.Errorf(err.Error())
		}
		analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Event:     "promo_code_associate",
				Timestamp: analytics.Time(time.Now()),
				PatientID: patientID,
				ExtraJSON: string(jsonData),
			},
		})
	}()

	return promotion.Data.(Promotion).SuccessMessage(), nil
}

// PatientSignedup attempts to identify a ParkedAccount with the same email as the patient that just signed up,
// and then applies the pending promotion to the patient's account if one exists.
func PatientSignedup(patientID int64, email string, dataAPI api.DataAPI, analyticsLogger analytics.Logger) (string, error) {
	// check if a parked account exists
	parkedAccount, err := dataAPI.ParkedAccount(email)
	if err == api.NoRowsError {
		return "", nil
	} else if err != nil {
		return "", err
	}

	// if it does, asynchronously assocate the promo code with this patient account
	// while returning the success message of the promotion
	var promotion *common.Promotion
	var referralProgram ReferralProgram
	if parkedAccount.IsReferral {
		rp, err := dataAPI.ReferralProgram(parkedAccount.CodeID, Types)
		if err != nil {
			return "", err
		}
		referralProgram = rp.Data.(ReferralProgram)
		promotion = referralProgram.PromotionForReferredPatient(parkedAccount.Code)
	} else {
		promotion, err = dataAPI.Promotion(parkedAccount.CodeID, Types)
		if err != nil {
			return "", err
		}
	}

	go func() {
		if err := dataAPI.MarkParkedAccountAsPatientCreated(parkedAccount.ID); err != nil {
			golog.Errorf(err.Error())
			return
		}

		// associate the promotion with the patient account
		if err := promotion.Data.(Promotion).Associate(patientID, parkedAccount.CodeID, promotion.Expires, dataAPI); err != nil {
			golog.Errorf(err.Error())
			return
		}

		if referralProgram != nil {
			if err := referralProgram.ReferredPatientAssociatedCode(patientID, parkedAccount.CodeID, dataAPI); err != nil {
				golog.Errorf(err.Error())
				return
			}
		}

		jsonData, err := json.Marshal(map[string]interface{}{
			"parked_account_id": parkedAccount.ID,
			"code":              promotion.Code,
			"is_referral":       parkedAccount.IsReferral,
		})
		if err != nil {
			golog.Errorf(err.Error())
		}

		analyticsLogger.WriteEvents([]analytics.Event{
			&analytics.ServerEvent{
				Event:     "promo_code_signup",
				Timestamp: analytics.Time(time.Now()),
				PatientID: patientID,
				ExtraJSON: string(jsonData),
			},
		})

	}()

	return promotion.Data.(Promotion).SuccessMessage(), nil
}
