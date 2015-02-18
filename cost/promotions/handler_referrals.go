package promotions

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"text/template"

	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/apiservice"
	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
)

type referralProgramHandler struct {
	domain  string
	dataAPI api.DataAPI
}

type shareTextInfo struct {
	EmailSubject string `json:"email_subject"`
	EmailBody    string `json:"email_body"`
	SMS          string `json:"sms"`
	Twitter      string `json:"twitter"`
	Facebook     string `json:"facebook"`
	Pasteboard   string `json:"pasteboard"`
	Default      string `json:"default"`
}

type referralDisplayInfo struct {
	CTATitle       string         `json:"account_screen_cta_title"`
	NavBarTitle    string         `json:"nav_bar_title"`
	Title          string         `json:"title"`
	Body           string         `json:"body_text"`
	URLDisplayText string         `json:"url_display_text"`
	URL            string         `json:"url"`
	ButtonTitle    string         `json:"button_title"`
	ShareText      *shareTextInfo `json:"share_text"`
}

func NewReferralProgramHandler(dataAPI api.DataAPI, domain string) http.Handler {
	return apiservice.AuthorizationRequired(&referralProgramHandler{
		dataAPI: dataAPI,
		domain:  domain,
	})
}

func (p *referralProgramHandler) IsAuthorized(r *http.Request) (bool, error) {
	ctxt := apiservice.GetContext(r)
	if ctxt.Role != api.PATIENT_ROLE {
		return false, apiservice.NewAccessForbiddenError()
	}

	if r.Method != apiservice.HTTP_GET {
		return false, apiservice.NewAccessForbiddenError()
	}

	return true, nil
}

func (p *referralProgramHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ctxt := apiservice.GetContext(r)

	// get the current active referral template
	referralProgramTemplate, err := p.dataAPI.ActiveReferralProgramTemplate(api.PATIENT_ROLE, Types)
	if api.IsErrNotFound(err) {
		apiservice.WriteResourceNotFoundError("No active referral program template found", w, r)
		return
	} else if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	referralProgram, err := p.dataAPI.ActiveReferralProgramForAccount(ctxt.AccountID, Types)
	if err != nil && !api.IsErrNotFound(err) {
		apiservice.WriteError(err, w, r)
		return
	}

	if api.IsErrNotFound(err) {
		// create a referral program for patient if it doesn't exist
		referralProgram, err = p.createReferralProgramFromTemplate(referralProgramTemplate, ctxt.AccountID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	} else if *referralProgram.TemplateID != referralProgramTemplate.ID {
		// create a new referral program for the patient if the current one is not the latest/active referral program
		referralProgram, err = p.createReferralProgramFromTemplate(referralProgramTemplate, ctxt.AccountID)
		if err != nil {
			apiservice.WriteError(err, w, r)
			return
		}
	}

	referralURL, err := url.Parse(fmt.Sprintf("https://%s/r/%s", p.domain, strings.ToLower(referralProgram.Code)))
	if err != nil {
		apiservice.WriteError(err, w, r)
		return
	}

	rp := referralProgram.Data.(ReferralProgram)
	shareTextParams := rp.ShareTextInfo()
	referralCtxt := &referralContext{
		ReferralURL: referralURL.String(),
	}

	emailSubject, err := populateReferralLink(shareTextParams.EmailSubject, referralCtxt)
	if err != nil {
		golog.Errorf(err.Error())
	}

	emailBody, err := populateReferralLink(shareTextParams.EmailBody, referralCtxt)
	if err != nil {
		golog.Errorf(err.Error())
	}

	twitter, err := populateReferralLink(shareTextParams.Twitter, referralCtxt)
	if err != nil {
		golog.Errorf(err.Error())
	}

	facebook, err := populateReferralLink(shareTextParams.Facebook, referralCtxt)
	if err != nil {
		golog.Errorf(err.Error())
	}

	sms, err := populateReferralLink(shareTextParams.SMS, referralCtxt)
	if err != nil {
		golog.Errorf(err.Error())
	}

	defaultTxt, err := populateReferralLink(shareTextParams.Default, referralCtxt)
	if err != nil {
		golog.Errorf(err.Error())
	}

	httputil.JSONResponse(w, http.StatusOK, referralDisplayInfo{
		CTATitle:       "Refer a Friend",
		NavBarTitle:    "Refer a Friend",
		Title:          rp.Title(),
		Body:           rp.Description(),
		URL:            referralURL.String(),
		URLDisplayText: referralURL.Host + referralURL.Path,
		ButtonTitle:    "Share Your Link",
		ShareText: &shareTextInfo{
			EmailSubject: emailSubject,
			EmailBody:    emailBody,
			Twitter:      twitter,
			Facebook:     facebook,
			SMS:          sms,
			Pasteboard:   referralURL.String(),
			Default:      defaultTxt,
		},
	})
}

type referralContext struct {
	ReferralURL string
}

func populateReferralLink(strTemplate string, ctxt *referralContext) (string, error) {
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

func (p *referralProgramHandler) createReferralProgramFromTemplate(referralProgramTemplate *common.ReferralProgramTemplate, accountID int64) (*common.ReferralProgram, error) {
	rp := referralProgramTemplate.Data.(ReferralProgram)
	rp.SetOwnerAccountID(accountID)

	promoCode, err := GeneratePromoCode(p.dataAPI)
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
		if err := p.dataAPI.CreateReferralProgram(referralProgram); err != nil {
			golog.Errorf(err.Error())
			return
		}
	})

	return referralProgram, nil
}
