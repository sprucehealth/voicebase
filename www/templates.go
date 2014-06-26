package www

import (
	"bytes"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/sprucehealth/backend/third_party/github.com/cookieo9/resources-go"
)

var (
	BaseTemplate       *baseTemplate
	IndexTemplate      *indexTemplate
	SimpleBaseTemplate *simpleBaseTemplate
)

var ResourceBundle resources.BundleSequence

func init() {
	if p := os.Getenv("GOPATH"); p != "" {
		ResourceBundle = append(ResourceBundle, resources.OpenFS(path.Join(p, "src", "carefront", "resources")))
	}
	if p := os.Getenv("RESOURCEPATH"); p != "" {
		ResourceBundle = append(ResourceBundle, resources.OpenFS(p))
	}
	if exePath, err := resources.ExecutablePath(); err == nil {
		if exe, err := resources.OpenZip(exePath); err == nil {
			ResourceBundle = append(ResourceBundle, exe)
		}
	}

	fi, err := ResourceBundle.Open("templates/base.html")
	if err != nil {
		panic(err)
	}
	_ = fi

	BaseTemplate = &baseTemplate{MustLoadTemplate("", "base.html")}
	IndexTemplate = &indexTemplate{MustLoadTemplate("", "index.html")}
	SimpleBaseTemplate = &simpleBaseTemplate{MustLoadTemplate("", "simple_base.html")}
}

func MustLoadTemplate(name, pth string) *template.Template {
	f, err := ResourceBundle.Open(path.Join("templates", pth))
	if err != nil {
		panic(err)
	}
	src, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	f.Close()
	return template.Must(template.New(name).Parse(string(src)))
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
