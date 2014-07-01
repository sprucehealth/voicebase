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
		{"signup", signupTemplate, &www.BaseTemplateContext{SubContext: &signupTemplateContext{Form: &signupRequest{}}}},
	}
	for _, tc := range templates {
		if err := tc.template.Execute(ioutil.Discard, tc.context); err != nil {
			t.Fatalf("Failed to execute template '%s': %s", tc.name, err.Error())
		}
	}
}
