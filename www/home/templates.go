package home

import (
	"html/template"

	"github.com/sprucehealth/backend/www"
)

var (
	baseTemplate  *template.Template
	homeTemplate  *template.Template
	aboutTemplate *template.Template
	passTemplate  *template.Template
)

func init() {
	passTemplate = www.MustLoadTemplate("home/pass.html", template.Must(www.BaseTemplate.Clone()))
	baseTemplate = www.MustLoadTemplate("home/base.html", template.Must(www.BaseTemplate.Clone()))
	homeTemplate = www.MustLoadTemplate("home/home.html", template.Must(baseTemplate.Clone()))
	aboutTemplate = www.MustLoadTemplate("home/about.html", template.Must(baseTemplate.Clone()))
}

type passTemplateContext struct {
	Error string
}
