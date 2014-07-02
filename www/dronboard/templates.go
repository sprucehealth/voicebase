package dronboard

import (
	"html/template"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/www"
)

var (
	signupTemplate     *template.Template
	credsTemplate      *template.Template
	uploadTemplate     *template.Template
	engagementTemplate *template.Template
)

func init() {
	signupTemplate = www.MustLoadTemplate("dronboard/signup.html", template.Must(www.BaseTemplate.Clone()))
	credsTemplate = www.MustLoadTemplate("dronboard/creds.html", template.Must(www.BaseTemplate.Clone()))
	uploadTemplate = www.MustLoadTemplate("dronboard/upload.html", template.Must(www.BaseTemplate.Clone()))
	engagementTemplate = www.MustLoadTemplate("dronboard/engagement.html", template.Must(www.BaseTemplate.Clone()))
}

type signupTemplateContext struct {
	Form       *signupRequest
	FormErrors map[string]string
	States     []*common.State
}

type credsTemplateContext struct {
	Form            *credentialsRequest
	FormErrors      map[string]string
	LicenseStatuses []string
	States          []*common.State
}

type uploadTemplateContext struct {
	Title string
}

type engagementTemplateContext struct {
	Form       *engagementRequest
	FormErrors map[string]string
}
