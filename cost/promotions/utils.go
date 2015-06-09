package promotions

import (
	"bytes"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"reflect"
	"text/template"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/app_url"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/errors"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
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
	common.PromotionTypes[n.TypeName()] = reflect.TypeOf(reflect.Indirect(reflect.ValueOf(n)).Interface())
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
		return ErrPromotionAlreadyApplied
	} else if err != nil {
		return err
	}

	promotionGroup, err := dataAPI.PromotionGroup(group)
	if api.IsErrNotFound(err) {
		return ErrInvalidCode
	} else if err != nil {
		return err
	}

	// ensure that the patient doesn't have the max codes applied against the group already
	if count, err := dataAPI.PromotionCountInGroupForAccount(accountID, group); err != nil {
		return err
	} else if promotionGroup.MaxAllowedPromos <= count {
		return ErrPromotionAlreadyExists
	}

	if forNewUser {
		patientID, err := dataAPI.GetPatientIDFromAccountID(accountID)
		if err != nil {
			return err
		}

		if isNewUser, err := IsNewPatient(patientID, dataAPI); err != nil {
			return err
		} else if !isNewUser {
			return ErrPromotionOnlyForNewUsers
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

		randomNumber, err := common.GenerateRandomNumber(9999999, 7)
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

type ReferralContext struct {
	ReferralURL string
}

func PopulateReferralLink(strTemplate string, ctxt *ReferralContext) (string, error) {
	tmpl, err := template.New("").Parse(strTemplate)
	if err != nil {
		return "", err
	}

	var b bytes.Buffer
	if err := tmpl.Execute(&b, ctxt); err != nil {
		return "", err
	}

	return b.String(), nil
}

func CreateReferralProgramFromTemplate(referralProgramTemplate *common.ReferralProgramTemplate, accountID int64, dataAPI api.DataAPI) (*common.ReferralProgram, error) {
	rp := referralProgramTemplate.Data.(ReferralProgram)
	rp.SetOwnerAccountID(accountID)

	promoCode, err := GeneratePromoCode(dataAPI)
	if err != nil {
		return nil, err
	}

	referralProgram := &common.ReferralProgram{
		TemplateID: &referralProgramTemplate.ID,
		AccountID:  accountID,
		Code:       promoCode,
		Data:       rp,
		Status:     common.RSActive,
	}

	// asnychronously create the referral program so as to not impact
	// the latency on the API
	dispatch.RunAsync(func() {
		if err := dataAPI.CreateReferralProgram(referralProgram); err != nil {
			golog.Errorf(err.Error())
			return
		}
	})

	return referralProgram, nil
}

func CreateReferralDisplayInfo(dataAPI api.DataAPI, webDomain string, accountID int64) (*ReferralDisplayInfo, error) {
	// get the current active referral template
	referralProgramTemplate, err := dataAPI.ActiveReferralProgramTemplate(api.RolePatient, common.PromotionTypes)
	if api.IsErrNotFound(err) {
		return nil, errors.Trace(errors.New("No active referral program template found"))
	} else if err != nil {
		return nil, errors.Trace(err)
	}

	referralProgram, err := dataAPI.ActiveReferralProgramForAccount(accountID, common.PromotionTypes)
	if err != nil && !api.IsErrNotFound(err) {
		return nil, errors.Trace(err)
	}

	if api.IsErrNotFound(err) {
		// create a referral program for patient if it doesn't exist
		referralProgram, err = CreateReferralProgramFromTemplate(referralProgramTemplate, accountID, dataAPI)
		if err != nil {
			return nil, errors.Trace(err)
		}
	} else if *referralProgram.TemplateID != referralProgramTemplate.ID {
		// create a new referral program for the patient if the current one is not the latest/active referral program
		referralProgram, err = CreateReferralProgramFromTemplate(referralProgramTemplate, accountID, dataAPI)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	// Lookup the account code for the indicated account and if one doesn't exist yet, associate one.
	accountCode, err := dataAPI.AccountCode(accountID)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if accountCode == nil {
		newCode, err := dataAPI.AssociateRandomAccountCode(accountID)
		if err != nil {
			return nil, errors.Trace(err)
		}
		accountCode = &newCode
	}

	referralURL, err := url.Parse(fmt.Sprintf("https://%s/r/%d", webDomain, *accountCode))
	if err != nil {
		return nil, errors.Trace(err)
	}

	promotionReferralProgram := referralProgram.Data.(ReferralProgram)
	shareTextParams := promotionReferralProgram.ShareTextInfo()
	referralCtxt := &ReferralContext{
		ReferralURL: referralURL.String(),
	}

	emailSubject, err := PopulateReferralLink(shareTextParams.EmailSubject, referralCtxt)
	if err != nil {
		golog.Errorf(err.Error())
	}

	emailBody, err := PopulateReferralLink(shareTextParams.EmailBody, referralCtxt)
	if err != nil {
		golog.Errorf(err.Error())
	}

	twitter, err := PopulateReferralLink(shareTextParams.Twitter, referralCtxt)
	if err != nil {
		golog.Errorf(err.Error())
	}

	facebook, err := PopulateReferralLink(shareTextParams.Facebook, referralCtxt)
	if err != nil {
		golog.Errorf(err.Error())
	}

	sms, err := PopulateReferralLink(shareTextParams.SMS, referralCtxt)
	if err != nil {
		golog.Errorf(err.Error())
	}

	defaultTxt, err := PopulateReferralLink(shareTextParams.Default, referralCtxt)
	if err != nil {
		golog.Errorf(err.Error())
	}

	displayURL := referralURL.Host + referralURL.Path
	if displayURL[:4] == "www." {
		displayURL = displayURL[4:]
	}
	// Note: Temporarily return a placeholder image and the associated size.
	return &ReferralDisplayInfo{
		CTATitle:           "Refer a Friend",
		NavBarTitle:        "Refer a Friend",
		Title:              promotionReferralProgram.Title(),
		Body:               promotionReferralProgram.Description(),
		URL:                referralURL.String(),
		URLDisplayText:     displayURL,
		ButtonTitle:        "Share Link",
		DismissButtonTitle: "Okay",
		ImageURL:           "https://d2bln09x7zhlg8.cloudfront.net/icon_share_default_160_x_160.png",
		ImageWidth:         80,
		ImageHeight:        80,
		ShareText: &ShareTextInfo{
			EmailSubject: emailSubject,
			EmailBody:    emailBody,
			Twitter:      twitter,
			Facebook:     facebook,
			SMS:          sms,
			Pasteboard:   referralURL.String(),
			Default:      defaultTxt,
		},
	}, nil
}
