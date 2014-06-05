package passreset

import (
	"bytes"
	"carefront/www"
	"html/template"
	"io"
	"path"
)

func init() {
	templatePath := "../../www/templates"
	PromptTemplate = &promptTemplate{template.Must(template.ParseFiles(path.Join(templatePath, "reset_password_prompt.html")))}
	VerifyTemplate = &verifyTemplate{template.Must(template.ParseFiles(path.Join(templatePath, "reset_password_verify.html")))}
	ResetTemplate = &resetTemplate{template.Must(template.ParseFiles(path.Join(templatePath, "reset_password.html")))}
}

// Reset Prompt Template

type promptTemplate struct {
	*template.Template
}

type PromptTemplateContext struct {
	Email        string
	InvalidEmail bool
	Sent         bool
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
		Body:  template.HTML(string(b.String())),
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
	Token  string
	Email  string
	Done   bool
	Errors []string
}

var ResetTemplate *resetTemplate

func (t *resetTemplate) Execute(w io.Writer, ctx interface{}) error {
	return t.Render(w, ctx.(*ResetTemplateContext))
}

func (t *resetTemplate) Render(w io.Writer, ctx *ResetTemplateContext) error {
	b := &bytes.Buffer{}
	if err := t.Template.Execute(b, ctx); err != nil {
		return err
	}
	return www.SimpleBaseTemplate.Render(w, &www.SimpleBaseTemplateContext{
		Title: "Password Reset",
		Body:  template.HTML(string(b.String())),
	})
}
