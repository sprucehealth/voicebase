package main

import (
	"flag"
	"net/http"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/boot"

	"log"

	"github.com/sprucehealth/backend/libs/golog"
	"golang.org/x/net/context"
)

type config struct {
	Environment       string
	ListenPort        int
	StaticResourceURL string
	WebURL            string
}

var c = &config{}

func init() {
	flag.StringVar(&c.Environment, "env", "dev", "Server environment")
	flag.IntVar(&c.ListenPort, "listen_port", 8200, "Listening port for web server.")
	flag.StringVar(&c.StaticResourceURL, "resource_url", "", "Static Resource URL.")
	flag.StringVar(&c.WebURL, "web_url", "", "Spruce Health web url including the carefinder path prefix.")
}

func main() {
	boot.ParseFlags("CAREFINDER_")

	c.StaticResourceURL = strings.Replace(c.StaticResourceURL, "{BuildNumber}", boot.BuildNumber, -1)
	if c.StaticResourceURL == "" {
		c.StaticResourceURL = "/static"
	}

	hand := buildCareFinder(c)

	// setup the router to serve pages for carefinder
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hand.ServeHTTP(context.Background(), w, r)
		}),
		Addr: ":" + strconv.Itoa(c.ListenPort),
	}

	// start server in non SSL mode assuming that SSL
	// termination has already happened at EBS or restapi layer.
	golog.Infof("Starting server...")
	log.Fatal(server.ListenAndServe())
}
