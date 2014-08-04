package passreset

import (
	"html/template"
	"io"

	"github.com/sprucehealth/backend/www"
)

func init() {
	baseTemplate = www.MustLoadTemplate("password_reset/base.html", template.Must(www.BaseTemplate.Clone()))
	PromptTemplate = &promptTemplate{www.MustLoadTemplate("password_reset/prompt.html", template.Must(baseTemplate.Clone()))}
	ResetTemplate = &resetTemplate{www.MustLoadTemplate("password_reset/reset.html", template.Must(baseTemplate.Clone()))}
	VerifyTemplate = &verifyTemplate{www.MustLoadTemplate("password_reset/verify.html", template.Must(baseTemplate.Clone()))}
}

var baseTemplate *template.Template

// Reset Prompt Template

type promptTemplate struct {
	*template.Template
}

type PromptTemplateContext struct {
	Email        string
	Error        string
	Sent         bool
	SupportEmail string
}

var PromptTemplate *promptTemplate

func (t *promptTemplate) Execute(w io.Writer, ctx interface{}) error {
	return t.Template.Execute(w, &www.BaseTemplateContext{
		Title:      "Password Reset | Spruce",
		SubContext: ctx,
	})
}

// Verify Send Template

type verifyTemplate struct {
	*template.Template
}

type VerifyTemplateContext struct {
	Token         string
	Email         string
	LastTwoDigits string
	EnterCode     bool
	Code          string
	Errors        []string
	SupportEmail  string
}

var VerifyTemplate *verifyTemplate

func (t *verifyTemplate) Execute(w io.Writer, ctx interface{}) error {
	return t.Template.Execute(w, &www.BaseTemplateContext{
		Title:      "Password Reset Verification | Spruce",
		SubContext: ctx,
	})
}

// Reset Template

type resetTemplate struct {
	*template.Template
}

type ResetTemplateContext struct {
	Token        string
	Email        string
	Done         bool
	Errors       []string
	SupportEmail string
}

var ResetTemplate *resetTemplate

func (t *resetTemplate) Execute(w io.Writer, ctx interface{}) error {
	return t.Template.Execute(w, &www.BaseTemplateContext{
		Title:      "Password Reset | Spruce",
		SubContext: ctx,
	})
}
