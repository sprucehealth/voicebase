package dronboard

import (
	"io/ioutil"
	"testing"

	"github.com/sprucehealth/backend/www"
)

type templateTest struct {
	name     string
	template www.Template
	context  interface{}
}

func TestTemplates(t *testing.T) {
	templates := []templateTest{
		{"creds", credsTemplate, &www.BaseTemplateContext{SubContext: &credsTemplateContext{Form: &credentialsForm{}}}},
		{"engagement", engagementTemplate, &www.BaseTemplateContext{SubContext: &engagementTemplateContext{Form: &engagementForm{}}}},
		{"financials", financialsTemplate, &www.BaseTemplateContext{SubContext: &financialsTemplateContext{Form: &financialsForm{}}}},
		{"register", registerTemplate, &www.BaseTemplateContext{SubContext: &registerTemplateContext{Form: &registerForm{}}}},
		{"upload", uploadTemplate, &www.BaseTemplateContext{SubContext: &uploadTemplateContext{}}},
	}
	for _, tc := range templates {
		if err := tc.template.Execute(ioutil.Discard, tc.context); err != nil {
			t.Fatalf("Failed to execute template '%s': %s", tc.name, err.Error())
		}
	}
}
