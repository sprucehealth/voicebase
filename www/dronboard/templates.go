package dronboard

import (
	"html/template"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/www"
)

var (
	registerTemplate   *template.Template
	credsTemplate      *template.Template
	uploadTemplate     *template.Template
	engagementTemplate *template.Template
	financialsTemplate *template.Template
	insuranceTemplate  *template.Template
)

func init() {
	registerTemplate = www.MustLoadTemplate("dronboard/signup.html", template.Must(www.BaseTemplate.Clone()))
	credsTemplate = www.MustLoadTemplate("dronboard/creds.html", template.Must(www.BaseTemplate.Clone()))
	uploadTemplate = www.MustLoadTemplate("dronboard/upload.html", template.Must(www.BaseTemplate.Clone()))
	engagementTemplate = www.MustLoadTemplate("dronboard/engagement.html", template.Must(www.BaseTemplate.Clone()))
	financialsTemplate = www.MustLoadTemplate("dronboard/financials.html", template.Must(www.BaseTemplate.Clone()))
	insuranceTemplate = www.MustLoadTemplate("dronboard/insurance.html", template.Must(www.BaseTemplate.Clone()))
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
	Title string
}

type engagementTemplateContext struct {
	Form       *engagementForm
	FormErrors map[string]string
}

type financialsTemplateContext struct {
	Form       *financialsForm
	FormErrors map[string]string
}

type insuranceTemplateContext struct {
	Form       *insuranceForm
	FormErrors map[string]string
}
