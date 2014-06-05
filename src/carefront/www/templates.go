package www

import (
	"bytes"
	"html/template"
	"io"
	"path"
)

var (
	BaseTemplate       *baseTemplate
	IndexTemplate      *indexTemplate
	SimpleBaseTemplate *simpleBaseTemplate
)

func init() {
	templatePath := "../../www/templates"
	BaseTemplate = &baseTemplate{template.Must(template.ParseFiles(path.Join(templatePath, "base.html")))}
	IndexTemplate = &indexTemplate{template.Must(template.ParseFiles(path.Join(templatePath, "index.html")))}
	SimpleBaseTemplate = &simpleBaseTemplate{template.Must(template.ParseFiles(path.Join(templatePath, "simple_base.html")))}
}

// base

type BaseTemplateContext struct {
	Title template.HTML
	Head  template.HTML
	Body  template.HTML
}

type baseTemplate struct {
	*template.Template
}

func (t *baseTemplate) Execute(w io.Writer, ctx interface{}) error {
	return t.Render(w, ctx.(*BaseTemplateContext))
}

func (t *baseTemplate) Render(w io.Writer, ctx *BaseTemplateContext) error {
	if ctx.Title == "" {
		ctx.Title = "Spruce"
	} else {
		ctx.Title = "Spruce | " + ctx.Title
	}
	return t.Template.Execute(w, ctx)
}

// index

type IndexTemplateContext struct {
}

type indexTemplate struct {
	*template.Template
}

func (t *indexTemplate) Execute(w io.Writer, ctx interface{}) error {
	return t.Render(w, ctx.(*IndexTemplateContext))
}

func (t *indexTemplate) Render(w io.Writer, ctx *IndexTemplateContext) error {
	b := &bytes.Buffer{}
	if err := t.Template.Execute(b, ctx); err != nil {
		return err
	}
	return BaseTemplate.Execute(w, &BaseTemplateContext{
		Body: template.HTML(string(b.String())),
	})
}

// simple base

type SimpleBaseTemplateContext struct {
	Title template.HTML
	Head  template.HTML
	Body  template.HTML
	Tail  template.HTML
}

type simpleBaseTemplate struct {
	*template.Template
}

func (t *simpleBaseTemplate) Execute(w io.Writer, ctx interface{}) error {
	return t.Render(w, ctx.(*SimpleBaseTemplateContext))
}

func (t *simpleBaseTemplate) Render(w io.Writer, ctx *SimpleBaseTemplateContext) error {
	if ctx.Title == "" {
		ctx.Title = "Spruce"
	} else {
		ctx.Title = "Spruce | " + ctx.Title
	}
	return t.Template.Execute(w, ctx)
}
