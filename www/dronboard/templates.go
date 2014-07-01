package dronboard

import (
	"html/template"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/www"
)

var signupTemplate *template.Template

func init() {
	signupTemplate = www.MustLoadTemplate("dronboard/signup.html", template.Must(www.BaseTemplate.Clone()))
}

type signupTemplateContext struct {
	Form       *signupRequest
	FormErrors map[string]string
	States     []*common.State
}
