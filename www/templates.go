package www

import (
	"html/template"
	"io/ioutil"
	"os"
	"path"

	"github.com/sprucehealth/backend/third_party/github.com/cookieo9/resources-go"
)

var (
	BaseTemplate       *template.Template
	IndexTemplate      *template.Template
	SimpleBaseTemplate *template.Template
	LoginTemplate      *template.Template
)

var ResourceBundle resources.BundleSequence

func init() {
	if p := os.Getenv("GOPATH"); p != "" {
		ResourceBundle = append(ResourceBundle, resources.OpenFS(path.Join(p, "src", "github.com", "sprucehealth", "backend", "resources")))
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

	BaseTemplate = MustLoadTemplate("base.html", nil)
	IndexTemplate = MustLoadTemplate("index.html", template.Must(BaseTemplate.Clone()))
	LoginTemplate = MustLoadTemplate("login.html", template.Must(BaseTemplate.Clone()))

	SimpleBaseTemplate = MustLoadTemplate("simple_base.html", nil)
}

func MustLoadTemplate(pth string, parent *template.Template) *template.Template {
	if parent == nil {
		parent = template.New("")
	}
	f, err := ResourceBundle.Open(path.Join("templates", pth))
	if err != nil {
		panic(err)
	}
	src, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	f.Close()
	return template.Must(parent.Parse(string(src)))
}

type BaseTemplateContext struct {
	Title      template.HTML
	SubContext interface{}
}

type SimpleBaseTemplateContext struct {
	Title      template.HTML
	SubContext interface{}
}

type LoginTemplateContext struct {
	Email string
	Next  string
	Error string
}
