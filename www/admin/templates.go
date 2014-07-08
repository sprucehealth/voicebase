package admin

import (
	"html/template"

	"github.com/sprucehealth/backend/common"
	"github.com/sprucehealth/backend/www"
)

var (
	drOnboardTemplate *template.Template
)

func init() {
	drOnboardTemplate = www.MustLoadTemplate("admin/dr-onboard.html", template.Must(www.BaseTemplate.Clone()))
}

type drOnboardTemplateContext struct {
	Doctor     *common.Doctor
	Attributes map[string]string
}
