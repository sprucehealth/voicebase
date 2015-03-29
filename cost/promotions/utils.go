package promotions

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"reflect"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
)

type promotionError struct {
	ErrorMsg string
}

func (p *promotionError) IsUserError() bool {
	return true
}

func (p *promotionError) UserError() string {
	return p.ErrorMsg
}

func (p *promotionError) Error() string {
	return p.ErrorMsg
}

func (p *promotionError) HTTPStatusCode() int {
	return http.StatusBadRequest
}

func init() {
	registerType(&percentDiscountPromotion{})
	registerType(&moneyDiscountPromotion{})
	registerType(&accountCreditPromotion{})
	registerType(&routeDoctorPromotion{})
	registerType(&giveReferralProgram{})
	registerType(&routeDoctorReferralProgram{})
}

func registerType(n common.Typed) {
	Types[n.TypeName()] = reflect.TypeOf(reflect.Indirect(reflect.ValueOf(n)).Interface())
}

type promoCodeParams struct {
	DisplayMsg string `json:"display_msg"`
	ImgURL     string `json:"image_url,omitempty"`
	ShortMsg   string `json:"short_msg"`
	SuccessMsg string `json:"success_msg"`
	PromoGroup string `json:"group"`
	ForNewUser bool   `json:"for_new_user"`
}

func (p *promoCodeParams) Validate() error {
	if p.DisplayMsg == "" {
		return errors.New("missing display msg")
	}
	if p.ShortMsg == "" {
		return errors.New("missing short msg")
	}
	if p.PromoGroup == "" {
		return errors.New("missing group")
	}
	if p.SuccessMsg == "" {
		return errors.New("missing success msg")
	}
	return nil
}

func (p *promoCodeParams) Group() string {
	return p.PromoGroup
}

func (p *promoCodeParams) DisplayMessage() string {
	return p.DisplayMsg
}

func (p *promoCodeParams) ShortMessage() string {
	return p.ShortMsg
}

func (p *promoCodeParams) SuccessMessage() string {
	return p.SuccessMsg
}

func (p *promoCodeParams) ImageURL() string {
	return p.ImgURL
}

type ShareTextParams struct {
	Facebook     string `json:"facebook"`
	Twitter      string `json:"twitter"`
	SMS          string `json:"sms"`
	Default      string `json:"default"`
	EmailBody    string `json:"email_body"`
	EmailSubject string `json:"email_subject"`
}

type HomeCardConfig struct {
	Text     string               `json:"text"`
	ImageURL *app_url.SpruceAsset `json:"image_url"`
}

type referralProgramParams struct {
	Title          string           `json:"title"`
	Description    string           `json:"description"`
	HomeCard       *HomeCardConfig  `json:"home_card"`
	ShareText      *ShareTextParams `json:"share_text_params"`
	OwnerAccountID int64            `json:"owner_account_id"`
}

func (r *referralProgramParams) Validate() error {
	return nil
}

const (
	percentOffType                = "promo_percent_off"
	moneyOffType                  = "promo_money_off"
	accountCreditType             = "promo_account_credit"
	routeDoctorType               = "promo_route_doctor"
	giveReferralType              = "referral_give"
	routeWithDiscountReferralType = "referral_route_discount"
)

func generateReferralCodeForDoctor(dataAPI api.DataAPI, doctor *common.Doctor) (string, error) {
	initialCode := fmt.Sprintf("dr%s", doctor.LastName)
	code := initialCode
	for i := 1; i <= 9; i++ {
		// check if the code alrady exists
		_, err := dataAPI.LookupPromoCode(code)
		if api.IsErrNotFound(err) {
			return code, nil
		} else if err != nil {
			return "", err
		}

		code = fmt.Sprintf("%s%d", initialCode, i)
	}

	return "", errors.New("Unable to generate promo code")
}

func canAssociatePromotionWithAccount(accountID, codeID int64, forNewUser bool, group string, dataAPI api.DataAPI) error {
	if codeExists, err := dataAPI.PromoCodeForAccountExists(accountID, codeID); codeExists {
		return PromotionAlreadyApplied
	} else if err != nil {
		return err
	}

	promotionGroup, err := dataAPI.PromotionGroup(group)
	if api.IsErrNotFound(err) {
		return InvalidCode
	} else if err != nil {
		return err
	}

	// ensure that the patient doesn't have the max codes applied against the group already
	if count, err := dataAPI.PromotionCountInGroupForAccount(accountID, group); err != nil {
		return err
	} else if promotionGroup.MaxAllowedPromos <= count {
		return PromotionAlreadyExists
	}

	if forNewUser {
		patientID, err := dataAPI.GetPatientIDFromAccountID(accountID)
		if err != nil {
			return err
		}

		if isNewUser, err := IsNewPatient(patientID, dataAPI); err != nil {
			return err
		} else if !isNewUser {
			return PromotionOnlyForNewUsersError
		}
	}

	return nil
}

// GeneratePromoCode generates a unique promo code using one of the prefixes in the
// database and then appending a random 4 digit number to the end
func GeneratePromoCode(dataAPI api.DataAPI) (string, error) {
	// pulling in all promo code prefixes here with the assumption that there aren't that many
	prefixes, err := dataAPI.PromoCodePrefixes()
	if err != nil {
		return "", err
	}

	for i := 0; i < 3; i++ {
		// randomly pick a prefix
		var prefix string
		if len(prefixes) > 0 {
			prefix = prefixes[rand.Intn(len(prefixes))]
		}

		randomNumber, err := common.GenerateRandomNumber(9999, 4)
		if err != nil {
			return "", err
		}

		promoCode := fmt.Sprintf("%s%s", prefix, randomNumber)

		// ensure that the promo code doesn't already exist
		_, err = dataAPI.LookupPromoCode(promoCode)
		if api.IsErrNotFound(err) {
			return promoCode, nil
		} else if err != nil {
			return "", err
		}
	}

	return "", errors.New("Unable to generate promo code")
}

func IsNewPatient(patientID int64, dataAPI api.DataAPI) (bool, error) {
	anyVisitsSubmitted, err := dataAPI.AnyVisitSubmitted(patientID)
	return !anyVisitsSubmitted, err
}
