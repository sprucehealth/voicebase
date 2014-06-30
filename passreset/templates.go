package passreset

import (
	"html/template"
	"io"

	"github.com/sprucehealth/backend/www"
)

func init() {
	PromptTemplate = &promptTemplate{www.MustLoadTemplate("password_reset/prompt.html", template.Must(www.SimpleBaseTemplate.Clone()))}
	ResetTemplate = &resetTemplate{www.MustLoadTemplate("password_reset/reset.html", template.Must(www.SimpleBaseTemplate.Clone()))}
	VerifyTemplate = &verifyTemplate{www.MustLoadTemplate("password_reset/verify.html", template.Must(www.SimpleBaseTemplate.Clone()))}
}

// Reset Prompt Template

type promptTemplate struct {
	*template.Template
}

type PromptTemplateContext struct {
	Email        string
	InvalidEmail bool
	Sent         bool
	SupportEmail string
}

var PromptTemplate *promptTemplate

func (t *promptTemplate) Execute(w io.Writer, ctx interface{}) error {
	return t.Template.Execute(w, &www.SimpleBaseTemplateContext{
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
	return t.Template.Execute(w, &www.SimpleBaseTemplateContext{
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
	return t.Template.Execute(w, &www.SimpleBaseTemplateContext{
		Title:      "Password Reset | Spruce",
		SubContext: ctx,
	})
}
