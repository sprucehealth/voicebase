package main

import (
	"flag"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/cookieo9/resources-go"
	"github.com/julienschmidt/httprouter"
	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/www"
)

type appConfig struct {
	Environment       string
	ListenPort        string
	StaticResourceURL string
	SlackWebhookURL   string
}

var c = &appConfig{}

var templateLoader *www.TemplateLoader

type page struct {
	Title             string
	ErrorMessage      string
	Environment       string
	StaticResourceURL string
}

type joinCommunityPOSTRequest struct {
	FirstName            string `json:"first_name"`
	LastName             string `json:"last_name"`
	Email                string `json:"email"`
	LicensedLocations    string `json:"licensed_locations"`
	ReasonsInterested    string `json:"reasons_interested"`
	DermatologyInterests string `json:"dermatology_interests"`
	ReferralSource       string `json:"referral_source"`
}

var robotsTXT = []byte(`Sitemap: https://www.chatcurbside.com/sitemap.xml
User-agent: *
Disallow:
`)

var sitemapXML = []byte(`<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url>
		<loc>https://www.chatcurbside.com</loc>
		<changefreq>daily</changefreq>
	</url>
	<url>
		<loc>https://www.chatcurbside.com/apply</loc>
		<changefreq>daily</changefreq>
	</url>
</urlset>
`)

var fiveHundredTemplate *template.Template
var fourOhFourTemplate *template.Template
var indexTemplate *template.Template
var applyTemplate *template.Template
var thanksTemplate *template.Template

func init() {
	flag.StringVar(&c.Environment, "env", "dev", "Server environment")
	flag.StringVar(&c.ListenPort, "listen_port", "8100", "Listening port for web server. Defaults to 8100.")
	flag.StringVar(&c.StaticResourceURL, "static_resource_url", "", "Static resource URL.")

	// Load templates on program initialisation
	www.MustInitializeResources("cmd/svc/curbside/build")
	templateLoader = www.NewTemplateLoader(func(path string) (io.ReadCloser, error) {
		return resources.DefaultBundle.Open("templates/" + path)
	})
	templateLoader.RegisterFunctions(map[string]interface{}{
		"staticURL": func(path string) string {
			return c.StaticResourceURL + path
		},
	})
	templateLoader.MustLoadTemplate("base.html", "", nil)
	fiveHundredTemplate = templateLoader.MustLoadTemplate("500.html", "base.html", nil)
	fourOhFourTemplate = templateLoader.MustLoadTemplate("404.html", "base.html", nil)
	indexTemplate = templateLoader.MustLoadTemplate("index.html", "base.html", nil)
	applyTemplate = templateLoader.MustLoadTemplate("apply.html", "base.html", nil)
	thanksTemplate = templateLoader.MustLoadTemplate("thanks.html", "base.html", nil)
}

func panicHandler(w http.ResponseWriter, r *http.Request, p interface{}) {
	// TODO: report/log panics

	http.Redirect(w, r, "/500", http.StatusTemporaryRedirect)
}

type notFound struct{}

func (f notFound) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/404", http.StatusTemporaryRedirect)
}

func fiveHundredHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// TODO: report/log 500s

	p := &page{
		Environment:       c.Environment,
		StaticResourceURL: c.StaticResourceURL,
		Title:             "Something Went Wrong | Curbside",
	}
	www.TemplateResponse(w, http.StatusOK, fiveHundredTemplate, p)
}

func fourOhFourHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// TODO: report/log 404s?

	p := &page{
		Environment:       c.Environment,
		StaticResourceURL: c.StaticResourceURL,
		Title:             "Page Not Found | Curbside",
	}
	www.TemplateResponse(w, http.StatusOK, fourOhFourTemplate, p)
}

func indexHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	p := &page{
		Environment:       c.Environment,
		StaticResourceURL: c.StaticResourceURL,
		Title:             "Curbside",
	}
	www.TemplateResponse(w, http.StatusOK, indexTemplate, p)
}

func applyHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	p := &page{
		Environment:       c.Environment,
		StaticResourceURL: c.StaticResourceURL,
		Title:             "Join the Community | Curbside",
	}
	www.TemplateResponse(w, http.StatusOK, applyTemplate, p)
}

func thanksHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	p := &page{
		Environment:       c.Environment,
		StaticResourceURL: c.StaticResourceURL,
		Title:             "Thank You | Curbside",
	}
	www.TemplateResponse(w, http.StatusOK, thanksTemplate, p)
}

func robotsTxtHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "text/plain")
	// TODO: set cache headers
	if _, err := w.Write(robotsTXT); err != nil {
		golog.Errorf(err.Error())
	}
}

func sitemapXMLHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "application/xml")
	// TODO: set cache headers
	if _, err := w.Write(sitemapXML); err != nil {
		golog.Errorf(err.Error())
	}
}

func main() {
	flag.Parse()

	c.StaticResourceURL = strings.Replace(c.StaticResourceURL, "{BuildNumber}", boot.BuildNumber, -1)

	router := httprouter.New()
	router.GET("/", indexHandler)
	router.GET("/apply", applyHandler)
	router.GET("/thanks", thanksHandler)
	router.POST("/submit", submitHandler)
	router.GET("/500", fiveHundredHandler)
	router.GET("/404", fourOhFourHandler)
	router.ServeFiles("/img/*filepath", http.Dir("build/img/"))
	router.ServeFiles("/fonts/*filepath", http.Dir("build/fonts/"))
	router.ServeFiles("/css/*filepath", http.Dir("build/css/"))
	router.ServeFiles("/js/*filepath", http.Dir("build/js/"))
	router.GET("/robots.txt", robotsTxtHandler)
	router.GET("/sitemap.xml", sitemapXMLHandler)
	router.PanicHandler = panicHandler
	router.NotFound = notFound{}

	log.Fatal(http.ListenAndServe(":"+c.ListenPort, router))
}
