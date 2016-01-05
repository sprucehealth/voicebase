package main

import (
	"flag"
	"net"
	"net/http"
	"time"

	"github.com/sprucehealth/backend/boot"
	"github.com/sprucehealth/backend/cmd/svc/excommsapi/internal/handlers"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/svc/excomms"
	"github.com/sprucehealth/go-proxy-protocol/proxyproto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var config struct {
	httpAddr          string
	proxyProtocol     bool
	excommsServiceURL string
}

func init() {
	flag.StringVar(&config.httpAddr, "http", "0.0.0.0:8900", "listen for http on `host:port`")
	flag.BoolVar(&config.proxyProtocol, "proxyproto", false, "enabled proxy protocol")
	flag.StringVar(&config.excommsServiceURL, "excomms.url", "localhost:5200", "url for events processor service. format `host:port`")
}

func main() {
	boot.ParseFlags("EXCOMMSAPI_")

	conn, err := grpc.Dial(config.excommsServiceURL, grpc.WithInsecure())
	if err != nil {
		golog.Fatalf("Unable to communicate with events processor service: %s", err.Error())
	}
	defer conn.Close()

	router := mux.NewRouter().StrictSlash(true)
	router.Handle("/twilio/{event}", handlers.NewTwilioRequestHandler(excomms.NewExCommsClient(conn)))

	webRequestLogger := func(ctx context.Context, ev *httputil.RequestEvent) {

		contextVals := []interface{}{
			"Method", ev.Request.Method,
			"URL", ev.URL.String(),
			"UserAgent", ev.Request.UserAgent(),
			"RequestID", httputil.RequestID(ctx),
			"RemoteAddr", ev.RemoteAddr,
			"StatusCode", ev.StatusCode,
		}

		log := golog.Context(contextVals...)

		if ev.Panic != nil {
			log.Criticalf("http: panic: %v\n%s", ev.Panic, ev.StackTrace)
		} else {
			log.Infof("excommsapi")
		}
	}

	h := httputil.LoggingHandler(router, webRequestLogger)
	h = httputil.RequestIDHandler(h)
	h = httputil.CompressResponse(httputil.DecompressRequest(h))
	serve(h)
}

func serve(handler httputil.ContextHandler) {
	listener, err := net.Listen("tcp", config.httpAddr)
	if err != nil {
		golog.Fatalf(err.Error())
	}
	if config.proxyProtocol {
		listener = &proxyproto.Listener{Listener: listener}
	}
	s := &http.Server{
		Handler:        httputil.FromContextHandler(handler),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	golog.Infof("Starting listener on %s...", config.httpAddr)
	// TODO: Only listen on secure connection.
	golog.Fatalf(s.Serve(listener).Error())
}
