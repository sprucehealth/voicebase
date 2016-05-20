package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/libs/golog"
	"golang.org/x/net/context"
)

type config struct {
	Environment                   string
	ListenPort                    int
	StaticResourceURL             string
	ContentURL                    string
	WebURL                        string
	DBHost                        string
	DBName                        string
	DBUserName                    string
	DBPassword                    string
	DBPort                        int
	YelpToken                     string
	YelpTokenSecret               string
	YelpConsumerKey               string
	YelpConsumerSecret            string
	GoogleStaticMapsKey           string
	GoogleStatciMapsURLSigningKey string
	BehindProxy                   bool
}

var c = &config{}

func init() {
	flag.StringVar(&c.Environment, "env", "dev", "Server environment")
	flag.IntVar(&c.ListenPort, "listen_port", 8200, "Listening port for web server.")
	flag.StringVar(&c.StaticResourceURL, "static_resource_url", "", "Static Resource URL.")
	flag.StringVar(&c.ContentURL, "content_url", "", "Carefinder Content URL.")
	flag.StringVar(&c.WebURL, "web_url", "", "Spruce Health web url including the carefinder path prefix.")
	flag.StringVar(&c.DBHost, "db_host", "", "Database host for carefinder")
	flag.StringVar(&c.DBName, "db_name", "", "Database name for carefinder")
	flag.StringVar(&c.DBUserName, "db_username", "", "Database username for carefinder")
	flag.StringVar(&c.DBPassword, "db_password", "", "Database password for carefinder")
	flag.IntVar(&c.DBPort, "db_port", 5432, "Database port for carefinder")
	flag.StringVar(&c.YelpToken, "yelp_token", "", "Yelp token to query yelp api")
	flag.StringVar(&c.YelpTokenSecret, "yelp_token_secret", "", "Yelp token secret to query yelp api")
	flag.StringVar(&c.YelpConsumerKey, "yelp_consumer_key", "", "Consumer key to query yelp api")
	flag.StringVar(&c.YelpConsumerSecret, "yelp_consumer_secret", "", "Consumer secret to query yelp api")
	flag.StringVar(&c.GoogleStaticMapsKey, "google_static_map_key", "", "Key for using google static maps api to generate map urls.")
	flag.StringVar(&c.GoogleStatciMapsURLSigningKey, "google_static_map_url_signing_key", "", "URL signing key to sign urls generated for google static maps.")
	flag.BoolVar(&c.BehindProxy, "behind_proxy", false, "Set this flag if behind a proxy")
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
