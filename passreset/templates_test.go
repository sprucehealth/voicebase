package passreset

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
		{"prompt", PromptTemplate, &PromptTemplateContext{}},
		{"reset", ResetTemplate, &ResetTemplateContext{}},
		{"verify", VerifyTemplate, &VerifyTemplateContext{}},
	}
	for _, tc := range templates {
		if err := tc.template.Execute(ioutil.Discard, tc.context); err != nil {
			t.Fatalf("Failed to execute template '%s': %s", tc.name, err.Error())
		}
	}
}
