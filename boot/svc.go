package boot

import (
	"flag"
	"net/http"
	_ "net/http/pprof" // imported for side-effect of registering HTTP handlers
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/samuel/go-metrics/metrics"
	"github.com/samuel/go-metrics/reporter"
	"github.com/sprucehealth/backend/environment"
	"github.com/sprucehealth/backend/libs/awsutil"
	"github.com/sprucehealth/backend/libs/golog"
	"google.golang.org/grpc"
)

// TODO: these get set in InitService. A bit unfortunate how this is setup. Will clean up later.
var (
	flagAWSAccessKey *string
	flagAWSSecretKey *string
	flagAWSToken     *string
	flagAWSRegion    *string
)

var (
	awsSessionOnce sync.Once
	awsSession     *session.Session
	awsSessionErr  error
)

// InitService should be called at the start of a service. It parses flags and sets up a mangement server.
func InitService(name string) metrics.Registry {
	var (
		flagDebug           = flag.Bool("debug", false, "Enable debug logging")
		flagEnv             = flag.String("env", "", "Execution environment")
		flagErrorSNSTopic   = flag.String("error_sns_topic", "", "SNS `topic` which to send errors")
		flagManagementAddr  = flag.String("management_addr", ":9000", "`host:port` of management HTTP server")
		flagLibratoUsername = flag.String("librato_username", "", "Librato metrics username")
		flagLibratoToken    = flag.String("librato_token", "", "Librato metrics token")
	)
	flagAWSAccessKey = flag.String("aws_access_key", "", "Access `key` for AWS")
	flagAWSSecretKey = flag.String("aws_secret_key", "", "Secret `key` for AWS")
	flagAWSToken = flag.String("aws_token", "", "Temporary access `token` for AWS")
	flagAWSRegion = flag.String("aws_region", "us-east-1", "AWS `region`")

	ParseFlags(strings.ToUpper(name) + "_")

	// Disable the built in grpc tracing and use our own
	grpc.EnableTracing = false

	if *flagEnv == "" {
		golog.Fatalf("-env flag required")
	}
	environment.SetCurrent(*flagEnv)

	if *flagDebug {
		golog.Default().SetLevel(golog.DEBUG)
	}

	// TODO: this can be expanded in the future to support registering custom health checks (e.g. checking connection to DB)
	http.Handle("/health-check", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}))

	// Start management server
	go func() {
		golog.Fatalf("%s", http.ListenAndServe(*flagManagementAddr, nil))
	}()

	metricsRegistry := metrics.NewRegistry()
	if *flagErrorSNSTopic != "" {
		awsSession, err := AWSSession()
		if err != nil {
			golog.Fatalf("Failed to create AWS session: %s", err)
		}
		golog.Default().SetHandler(SNSLogHandler(
			sns.New(awsSession), *flagErrorSNSTopic, environment.GetCurrent()+"/"+name,
			golog.Default().Handler(), nil, metricsRegistry.Scope("errorsns")))
	}

	metricsRegistry.Add("runtime", metrics.RuntimeMetrics)

	if *flagLibratoUsername != "" && *flagLibratoToken != "" {
		source := *flagEnv + "-" + name
		statsReporter := reporter.NewLibratoReporter(
			metricsRegistry, time.Minute, true, *flagLibratoUsername, *flagLibratoToken, source)
		statsReporter.Start()
	}

	return metricsRegistry.Scope("svc." + name)
}

// WaitForTermination waits for an INT or TERM signal.
func WaitForTermination() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	select {
	case sig := <-ch:
		golog.Infof("Quitting due to signal %s", sig.String())
	}
}

// AWSSession returns an AWS session. It must only be called after ParseFlags or InitService.
func AWSSession() (*session.Session, error) {
	awsSessionOnce.Do(func() {
		awsConfig, err := awsutil.Config(*flagAWSRegion, *flagAWSAccessKey, *flagAWSSecretKey, *flagAWSToken)
		if err != nil {
			awsSessionErr = err
			return
		}
		awsSession = session.New(awsConfig)
	})
	return awsSession, awsSessionErr
}
