package dronboard

import (
	"html/template"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/www"
)

var (
	baseTemplate             *template.Template
	registerTemplate         *template.Template
	credsTemplate            *template.Template
	uploadTemplate           *template.Template
	engagementTemplate       *template.Template
	financialsTemplate       *template.Template
	insuranceTemplate        *template.Template
	successTemplate          *template.Template
	financialsVerifyTemplate *template.Template
)

func init() {
	baseTemplate = www.MustLoadTemplate("dronboard/base.html", template.Must(www.BaseTemplate.Clone()))
	registerTemplate = www.MustLoadTemplate("dronboard/register.html", template.Must(baseTemplate.Clone()))
	credsTemplate = www.MustLoadTemplate("dronboard/creds.html", template.Must(baseTemplate.Clone()))
	uploadTemplate = www.MustLoadTemplate("dronboard/upload.html", template.Must(baseTemplate.Clone()))
	engagementTemplate = www.MustLoadTemplate("dronboard/engagement.html", template.Must(baseTemplate.Clone()))
	financialsTemplate = www.MustLoadTemplate("dronboard/financials.html", template.Must(baseTemplate.Clone()))
	insuranceTemplate = www.MustLoadTemplate("dronboard/insurance.html", template.Must(baseTemplate.Clone()))
	successTemplate = www.MustLoadTemplate("dronboard/success.html", template.Must(baseTemplate.Clone()))
	financialsVerifyTemplate = www.MustLoadTemplate("dronboard/financials_verify.html", template.Must(baseTemplate.Clone()))
}

type registerTemplateContext struct {
	Form       *registerForm
	FormErrors map[string]string
	States     []*common.State
}

type credsTemplateContext struct {
	Form            *credentialsForm
	FormErrors      map[string]string
	LicenseStatuses []common.MedicalLicenseStatus
	States          []*common.State
}

type uploadTemplateContext struct {
	Title   string
	NextURL string
}

type engagementTemplateContext struct {
	Form       *engagementForm
	FormErrors map[string]string
}

type insuranceTemplateContext struct {
	Form       *insuranceForm
	FormErrors map[string]string
}

type financialsTemplateContext struct {
	Form       *financialsForm
	FormErrors map[string]string
	StripeKey  string
}

type successTemplateContext struct {
}

type financialsVerifyTemplateContext struct {
	Form         *financialsVerifyForm
	FormErrors   map[string]string
	Initial      bool
	Pending      bool
	Failed       bool
	SupportEmail string
}
