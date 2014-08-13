package www

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sync"

	"github.com/sprucehealth/backend/third_party/github.com/cookieo9/resources-go"
)

type resourceFileSystem struct{}

// ResourceFileSystem implements the http.Filesystem interface to provide
// static file serving.
var (
	ResourceFileSystem = resourceFileSystem{}
	ResourcesPath      string
)

type tmpl struct {
	tmpl   *template.Template
	parent string
}

type TemplateLoader struct {
	templates map[string]*tmpl
	mu        sync.Mutex
	opener    func(path string) (io.ReadCloser, error)
}

func NewTemplateLoader(opener func(path string) (io.ReadCloser, error)) *TemplateLoader {
	if opener == nil {
		opener = func(path string) (io.ReadCloser, error) { return os.Open(path) }
	}
	return &TemplateLoader{
		templates: make(map[string]*tmpl),
		opener:    opener,
	}
}

func (t *TemplateLoader) SetOpener(opener func(path string) (io.ReadCloser, error)) {
	t.opener = opener
}

func (t *TemplateLoader) MustLoadTemplate(path, parent string, funcMap map[string]interface{}) *template.Template {
	tm, err := t.LoadTemplate(path, parent, funcMap)
	if err != nil {
		panic(err.Error())
	}
	return tm
}

func (t *TemplateLoader) LoadTemplate(path, parent string, funcMap map[string]interface{}) (*template.Template, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	tm := t.templates[path]
	if tm != nil {
		if tm.parent != parent {
			return nil, fmt.Errorf("trying to reload template %s with a different parent %s (was %s)", path, parent, tm.parent)
		}
		return tm.tmpl, nil
	}

	var p *template.Template
	if parent != "" {
		if tm = t.templates[parent]; tm == nil {
			return nil, fmt.Errorf("parent template %s not found", parent)
		} else {
			p = tm.tmpl
		}
		var err error
		p, err = p.Clone()
		if err != nil {
			return nil, err
		}
	} else {
		p = template.New("")
	}
	if funcMap != nil {
		p = p.Funcs(funcMap)
	}

	f, err := t.opener(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	src, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	tm = &tmpl{tmpl: template.Must(p.Parse(string(src))), parent: parent}
	t.templates[path] = tm
	return tm.tmpl, nil
}

func init() {
	resources.DefaultBundle = nil
	if p := os.Getenv("GOPATH"); p != "" {
		ResourcesPath = path.Join(p, "src", "github.com", "sprucehealth", "backend", "resources")
		resources.DefaultBundle = append(resources.DefaultBundle, resources.OpenFS(ResourcesPath))
	}
	if p := os.Getenv("RESOURCEPATH"); p != "" {
		resources.DefaultBundle = append(resources.DefaultBundle, resources.OpenFS(p))
	}
	if exePath, err := resources.ExecutablePath(); err == nil {
		if exe, err := resources.OpenZip(exePath); err == nil {
			resources.DefaultBundle = append(resources.DefaultBundle, exe)
		}
	}

	// Make sure the resources can be loaded
	fi, err := resources.DefaultBundle.Open("templates/base.html")
	if err != nil {
		panic(err)
	}
	fi.Close()
}

func (resourceFileSystem) Open(name string) (http.File, error) {
	if ResourcesPath == "" {
		return nil, os.ErrNotExist
	}
	f, err := os.Open(ResourcesPath + "/static" + name)
	if err != nil {
		return nil, err
	}
	// Don't allow opening directories
	if s, err := f.Stat(); err != nil {
		f.Close()
		return nil, err
	} else if s.IsDir() {
		f.Close()
		return nil, os.ErrNotExist
	}
	return httpFile{f}, nil
}

type httpFile struct {
	*os.File
}

func (httpFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

type BaseTemplateContext struct {
	Title      template.HTML
	SubContext interface{}
}
