package passreset

import (
	"bytes"
	"github.com/sprucehealth/backend/www"
	"html/template"
	"io"
)

func init() {
	PromptTemplate = &promptTemplate{www.MustLoadTemplate("", "password_reset/prompt.html")}
	VerifyTemplate = &verifyTemplate{www.MustLoadTemplate("", "password_reset/verify.html")}
	ResetTemplate = &resetTemplate{www.MustLoadTemplate("", "password_reset/reset.html")}
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
	return t.Render(w, ctx.(*PromptTemplateContext))
}

func (t *promptTemplate) Render(w io.Writer, ctx *PromptTemplateContext) error {
	b := &bytes.Buffer{}
	if err := t.Template.Execute(b, ctx); err != nil {
		return err
	}
	return www.SimpleBaseTemplate.Render(w, &www.SimpleBaseTemplateContext{
		Title: "Password Reset",
		Body:  template.HTML(b.String()),
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
	return t.Render(w, ctx.(*VerifyTemplateContext))
}

func (t *verifyTemplate) Render(w io.Writer, ctx *VerifyTemplateContext) error {
	b := &bytes.Buffer{}
	if err := t.Template.ExecuteTemplate(b, "tail", ctx); err != nil {
		return err
	}
	tail := template.HTML(b.String())
	b.Reset()
	if err := t.Template.ExecuteTemplate(b, "body", ctx); err != nil {
		return err
	}
	body := template.HTML(b.String())
	return www.SimpleBaseTemplate.Render(w, &www.SimpleBaseTemplateContext{
		Title: "Password Reset Verification",
		Body:  body,
		Tail:  tail,
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
	return t.Render(w, ctx.(*ResetTemplateContext))
}

func (t *resetTemplate) Render(w io.Writer, ctx *ResetTemplateContext) error {
	b := &bytes.Buffer{}
	if err := t.Template.ExecuteTemplate(b, "head", ctx); err != nil {
		return err
	}
	head := template.HTML(b.String())
	b.Reset()
	if err := t.Template.ExecuteTemplate(b, "body", ctx); err != nil {
		return err
	}
	body := template.HTML(b.String())
	return www.SimpleBaseTemplate.Render(w, &www.SimpleBaseTemplateContext{
		Title: "Password Reset",
		Head:  head,
		Body:  body,
	})
}
