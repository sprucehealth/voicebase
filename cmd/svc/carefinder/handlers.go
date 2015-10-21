package main

import (
	"io"

	resources "github.com/cookieo9/resources-go"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/dal"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/handlers"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/service"
	"github.com/sprucehealth/backend/cmd/svc/carefinder/internal/yelp"
	configlib "github.com/sprucehealth/backend/common/config"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/www"
	"golang.org/x/net/context"
)

func buildCareFinder(c *config) httputil.ContextHandler {

	// connect to database
	configDB := &configlib.DB{
		User:     c.DBUserName,
		Password: c.DBPassword,
		Host:     c.DBHost,
		Port:     c.DBPort,
		Name:     c.DBName,
	}

	db, err := configDB.ConnectPostgres()
	if err != nil {
		panic(err)
	}

	cityDAL := dal.NewCityDAL(db)
	doctorDAL := dal.NewDoctorDAL(db)
	cityService := service.NewForCity(cityDAL, doctorDAL, c.WebURL, c.ContentURL)

	yelpClient := yelp.NewClient(c.YelpConsumerKey, c.YelpConsumerSecret, c.YelpToken, c.YelpTokenSecret)
	doctorService := service.NewForDoctor(cityDAL, doctorDAL, yelpClient, c.WebURL, c.ContentURL, c.StaticResourceURL, c.GoogleStaticMapsKey, c.GoogleStatciMapsURLSigningKey)

	// initialize resources for the app
	www.MustInitializeResources("cmd/svc/carefinder/resources")

	templateLoader := www.NewTemplateLoader(func(path string) (io.ReadCloser, error) {
		return resources.DefaultBundle.Open("templates/" + path)
	})

	templateLoader.RegisterFunctions(map[string]interface{}{
		"staticURL": func(path string) string {
			return c.StaticResourceURL + "/" + path
		},
	})

	templateLoader.MustLoadTemplate("base.html", "", nil)

	router := mux.NewRouter()
	router.PathPrefix("/static").Handler(httputil.StripPrefix("/static", httputil.FileServer(www.ResourceFileSystem)))
	router.PathPrefix("/dermatologist-near-me/md-{doctor}").Handler(handlers.NewDoctorPageHandler(templateLoader, doctorService))
	router.Handle("/dermatologist-near-me/{city}", handlers.NewCityPageHandler(templateLoader, cityService))

	webRequestLogger := func(ctx context.Context, ev *httputil.RequestEvent) {
		log := golog.Context(
			"Method", ev.Request.Method,
			"URL", ev.URL.String(),
			"UserAgent", ev.Request.UserAgent(),
			"RemoteAddr", ev.Request.Referer(),
			"StatusCode", ev.StatusCode,
		)
		if ev.Panic != nil {
			log.Criticalf("http: panic: %v\n%s", ev.Panic, ev.StackTrace)
		} else {
			log.Infof("carefinder")
		}
	}
	return httputil.LoggingHandler(router, webRequestLogger)
}
